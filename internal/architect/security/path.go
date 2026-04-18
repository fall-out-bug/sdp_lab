//go:build !windows

package security

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"golang.org/x/sys/unix"
)

// PathValidator provides TOCTOU-safe path validation using openat() with O_NOFOLLOW.
// Platform support: Linux and macOS only. Use build tag "!windows".
type PathValidator struct {
	repoRoot string
	rootFd   int
	mu       sync.Mutex
}

// NewPathValidator creates a new path validator for a repository root.
// The repo root is opened once and reused for all validations.
func NewPathValidator(repoRoot string) (*PathValidator, error) {
	// Resolve the repo root to an absolute path
	absRoot, err := filepath.Abs(repoRoot)
	if err != nil {
		return nil, fmt.Errorf("resolve repo root: %w", err)
	}

	// Open the repo root directory to obtain a dirfd (anchored)
	rootFd, err := unix.Open(absRoot, unix.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, fmt.Errorf("open repo root %q: %w", absRoot, err)
	}

	return &PathValidator{
		repoRoot: absRoot,
		rootFd:   rootFd,
	}, nil
}

// ValidatePath ensures the resolved path is within repoRoot using TOCTOU-safe
// openat() with O_NOFOLLOW. It returns an io.ReadSeekCloser — callers can only
// Read/Seek/Close, no path operations.
//
// The validation process:
// 1. Open repo root directory as dirfd anchor
// 2. Clean and validate the relative path
// 3. Open file relative to rootFd with O_NOFOLLOW (prevents symlink attacks)
// 4. Get real path from file descriptor
// 5. Verify real path is within resolved repo root
// 6. Return ValidatedFile wrapper with only Read/Seek/Close operations
func (pv *PathValidator) ValidatePath(rawPath string) (io.ReadSeekCloser, error) {
	pv.mu.Lock()
	defer pv.mu.Unlock()

	// Step 1: Clean the relative path
	relPath := filepath.Clean(rawPath)

	// Step 1.5: Reject empty paths
	if rawPath == "" || relPath == "" || relPath == "." {
		return nil, &PathValidationError{
			Path:   rawPath,
			Reason: "empty path not allowed",
		}
	}

	// Step 2: Reject absolute paths — must be relative to repoRoot
	if filepath.IsAbs(relPath) {
		return nil, &PathValidationError{
			Path: rawPath,
			Reason: "absolute path not allowed; must be relative to repo root",
		}
	}

	// Step 3: Reject path traversal attempts
	if strings.HasPrefix(relPath, "..") {
		return nil, &PathValidationError{
			Path: rawPath,
			Reason: "path escapes repo root",
		}
	}

	// Check for path traversal anywhere in the path
	parts := strings.Split(relPath, string(filepath.Separator))
	for _, part := range parts {
		if part == ".." {
			return nil, &PathValidationError{
				Path: rawPath,
				Reason: "path traversal (..) not allowed",
			}
		}
	}

	// Step 4: Open the file relative to rootFd with O_NOFOLLOW
	fd, err := unix.Openat(pv.rootFd, relPath, unix.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, &PathValidationError{
			Path: rawPath,
			Reason: fmt.Sprintf("open failed: %v", err),
			Err: err,
		}
	}

	// Step 5: Get the real path from the open fd
	realPath, err := realPathFromFD(fd)
	if err != nil {
		unix.Close(fd)
		return nil, &PathValidationError{
			Path: rawPath,
			Reason: fmt.Sprintf("resolve real path: %v", err),
			Err: err,
		}
	}

	// Step 6: Resolve the repo root
	rootResolved, err := filepath.EvalSymlinks(pv.repoRoot)
	if err != nil {
		unix.Close(fd)
		return nil, &PathValidationError{
			Path: rawPath,
			Reason: fmt.Sprintf("resolve repo root: %v", err),
			Err: err,
		}
	}

	// Step 7: Verify the real path is within the resolved repo root
	if !strings.HasPrefix(realPath, rootResolved+string(filepath.Separator)) && realPath != rootResolved {
		unix.Close(fd)
		return nil, &PathValidationError{
			Path: rawPath,
			Reason: fmt.Sprintf("path resolves to %q which is outside repo root %q", realPath, rootResolved),
		}
	}

	// Step 8: Return the ValidatedFile wrapper
	vf := &ValidatedFile{
		fd: fd,
		path: realPath,
	}
	runtime.SetFinalizer(vf, func(f *ValidatedFile) {
		unix.Close(f.fd)
	})

	return vf, nil
}

// Close closes the root file descriptor.
func (pv *PathValidator) Close() error {
	pv.mu.Lock()
	defer pv.mu.Unlock()
	if pv.rootFd >= 0 {
		err := unix.Close(pv.rootFd)
		pv.rootFd = -1
		return err
	}
	return nil
}

// ValidatedFile wraps an open file descriptor after path validation.
// Exposes only Read, Seek, Close — no path-based operations.
// Implements io.ReadSeekCloser.
type ValidatedFile struct {
	fd   int
	path string // For debugging only; not exposed via interface
}

// Read reads up to len(p) bytes from the validated file.
func (f *ValidatedFile) Read(p []byte) (n int, err error) {
	return unix.Read(f.fd, p)
}

// Seek sets the offset for the next Read on the validated file.
func (f *ValidatedFile) Seek(offset int64, whence int) (int64, error) {
	return unix.Seek(f.fd, offset, whence)
}

// Close closes the underlying file descriptor and clears the finalizer.
func (f *ValidatedFile) Close() error {
	runtime.SetFinalizer(f, nil)
	return unix.Close(f.fd)
}

// PathValidationError represents an error that occurs during path validation.
type PathValidationError struct {
	Path   string
	Reason string
	Err    error
}

func (e *PathValidationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("path validation failed for %q: %s (%v)", e.Path, e.Reason, e.Err)
	}
	return fmt.Sprintf("path validation failed for %q: %s", e.Path, e.Reason)
}

func (e *PathValidationError) Unwrap() error {
	return e.Err
}

// ValidatePath is a convenience function that creates a validator, validates
// a single path, and cleans up. For multiple validations, create a validator
// and reuse it.
func ValidatePath(rawPath, repoRoot string) (io.ReadSeekCloser, error) {
	pv, err := NewPathValidator(repoRoot)
	if err != nil {
		return nil, err
	}
	defer pv.Close()

	return pv.ValidatePath(rawPath)
}

// SafeReadFile reads a file safely using path validation.
func SafeReadFile(rawPath, repoRoot string) ([]byte, error) {
	f, err := ValidatePath(rawPath, repoRoot)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}

// SafeOpenFile opens a file safely using path validation.
func SafeOpenFile(rawPath, repoRoot string) (*os.File, error) {
	f, err := ValidatePath(rawPath, repoRoot)
	if err != nil {
		return nil, err
	}

	// Convert to os.File for easier use
	// Note: This loses the TOCTOU protection after conversion
	// The caller should use ReadSeekCloser interface when possible
	if vf, ok := f.(*ValidatedFile); ok {
		// Duplicate the file descriptor
		newFd, err := unix.Dup(vf.fd)
		if err != nil {
			f.Close()
			return nil, err
		}
		return os.NewFile(uintptr(newFd), vf.path), nil
	}

	return nil, fmt.Errorf("invalid file type")
}

