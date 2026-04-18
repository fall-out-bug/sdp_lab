//go:build !windows

package security

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPathValidator_NewPathValidator(t *testing.T) {
	tempDir := t.TempDir()

	pv, err := NewPathValidator(tempDir)
	require.NoError(t, err)
	require.NotNil(t, pv)

	assert.Equal(t, tempDir, pv.repoRoot)
	assert.GreaterOrEqual(t, pv.rootFd, 0)

	err = pv.Close()
	require.NoError(t, err)
}

func TestPathValidator_ValidatePath_ValidFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := "test.txt"
	testPath := filepath.Join(tempDir, testFile)
	err := os.WriteFile(testPath, []byte("test content"), 0644)
	require.NoError(t, err)

	pv, err := NewPathValidator(tempDir)
	require.NoError(t, err)
	defer pv.Close()

	// Validate the file
	f, err := pv.ValidatePath(testFile)
	require.NoError(t, err)
	require.NotNil(t, f)
	defer f.Close()

	// Read and verify content
	content := make([]byte, 100)
	n, err := f.Read(content)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content[:n]))
}

func TestPathValidator_ValidatePath_RelativePath(t *testing.T) {
	tempDir := t.TempDir()

	// Create a subdirectory
	subDir := "subdir"
	err := os.Mkdir(filepath.Join(tempDir, subDir), 0755)
	require.NoError(t, err)

	// Create a test file in subdirectory
	testFile := filepath.Join(subDir, "test.txt")
	testPath := filepath.Join(tempDir, testFile)
	err = os.WriteFile(testPath, []byte("test content"), 0644)
	require.NoError(t, err)

	pv, err := NewPathValidator(tempDir)
	require.NoError(t, err)
	defer pv.Close()

	// Validate with relative path
	f, err := pv.ValidatePath(testFile)
	require.NoError(t, err)
	require.NotNil(t, f)
	defer f.Close()

	// Read and verify
	content, err := os.ReadFile(testPath)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(content))
}

func TestPathValidator_ValidatePath_AbsolutePath_Rejected(t *testing.T) {
	tempDir := t.TempDir()

	pv, err := NewPathValidator(tempDir)
	require.NoError(t, err)
	defer pv.Close()

	// Try to validate an absolute path
	_, err = pv.ValidatePath("/etc/passwd")
	require.Error(t, err)

	var pathErr *PathValidationError
	require.ErrorAs(t, err, &pathErr)
	assert.Contains(t, pathErr.Reason, "absolute path")
}

func TestPathValidator_ValidatePath_PathTraversal_Rejected(t *testing.T) {
	tempDir := t.TempDir()

	pv, err := NewPathValidator(tempDir)
	require.NoError(t, err)
	defer pv.Close()

	// Try various path traversal attempts
	traversalPaths := []string{
		"../../../etc/passwd",
		"../../secret.txt",
		"subdir/../../../etc/passwd",
		"./../../etc/passwd",
		"foo/../../bar",
	}

	for _, path := range traversalPaths {
		t.Run(path, func(t *testing.T) {
			_, err := pv.ValidatePath(path)
			require.Error(t, err, "should reject path: %s", path)

			var pathErr *PathValidationError
			require.ErrorAs(t, err, &pathErr)
			assert.Contains(t, pathErr.Reason, "escapes repo root")
		})
	}
}

func TestPathValidator_ValidatePath_SymlinkToFile_Rejected(t *testing.T) {
	tempDir := t.TempDir()

	// Create a file outside tempDir
	outsideFile := filepath.Join(t.TempDir(), "outside.txt")
	err := os.WriteFile(outsideFile, []byte("outside content"), 0644)
	require.NoError(t, err)

	// Create a symlink to the outside file
	symlinkPath := filepath.Join(tempDir, "symlink.txt")
	err = os.Symlink(outsideFile, symlinkPath)
	require.NoError(t, err)

	pv, err := NewPathValidator(tempDir)
	require.NoError(t, err)
	defer pv.Close()

	// Try to open the symlink - should fail due to O_NOFOLLOW
	_, err = pv.ValidatePath("symlink.txt")
	require.Error(t, err)
}

func TestPathValidator_ValidatePath_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()

	pv, err := NewPathValidator(tempDir)
	require.NoError(t, err)
	defer pv.Close()

	_, err = pv.ValidatePath("nonexistent.txt")
	require.Error(t, err)

	var pathErr *PathValidationError
	require.ErrorAs(t, err, &pathErr)
	assert.Contains(t, pathErr.Reason, "open failed")
}

func TestValidatedFile_ReadSeekClose(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := "test.txt"
	testPath := filepath.Join(tempDir, testFile)
	content := []byte("0123456789")
	err := os.WriteFile(testPath, content, 0644)
	require.NoError(t, err)

	pv, err := NewPathValidator(tempDir)
	require.NoError(t, err)
	defer pv.Close()

	f, err := pv.ValidatePath(testFile)
	require.NoError(t, err)
	defer f.Close()

	// Test Read
	buf := make([]byte, 5)
	n, err := f.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("01234"), buf)

	// Test Seek
	offset, err := f.Seek(0, 0) // Seek to start
	require.NoError(t, err)
	assert.Equal(t, int64(0), offset)

	n, err = f.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("01234"), buf)

	// Test Seek to middle
	offset, err = f.Seek(5, 0) // Seek to position 5
	require.NoError(t, err)
	assert.Equal(t, int64(5), offset)

	n, err = f.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, []byte("56789"), buf)

	// Test Close
	err = f.Close()
	require.NoError(t, err)

	// Read after close should fail
	_, err = f.Read(buf)
	require.Error(t, err)
}

func TestPathValidator_MultipleValidations(t *testing.T) {
	tempDir := t.TempDir()

	// Create multiple test files
	files := []string{"file1.txt", "file2.txt", "file3.txt"}
	for _, name := range files {
		path := filepath.Join(tempDir, name)
		err := os.WriteFile(path, []byte(name+" content"), 0644)
		require.NoError(t, err)
	}

	pv, err := NewPathValidator(tempDir)
	require.NoError(t, err)
	defer pv.Close()

	// Validate all files
	for _, name := range files {
		f, err := pv.ValidatePath(name)
		require.NoError(t, err, "should validate %s", name)
		f.Close()
	}
}

func TestPathValidationError_Error(t *testing.T) {
	err := &PathValidationError{
		Path:   "/etc/passwd",
		Reason: "absolute path not allowed",
	}

	errStr := err.Error()
	assert.Contains(t, errStr, "/etc/passwd")
	assert.Contains(t, errStr, "absolute path")
}

func TestPathValidationError_Unwrap(t *testing.T) {
	originalErr := os.ErrNotExist
	err := &PathValidationError{
		Path:   "test.txt",
		Reason: "open failed",
		Err:    originalErr,
	}

	assert.Equal(t, originalErr, err.Unwrap())
}

func TestValidatePath_ConvenienceFunction(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := "test.txt"
	testPath := filepath.Join(tempDir, testFile)
	err := os.WriteFile(testPath, []byte("test content"), 0644)
	require.NoError(t, err)

	// Use convenience function
	f, err := ValidatePath(testFile, tempDir)
	require.NoError(t, err)
	require.NotNil(t, f)
	defer f.Close()

	// Verify content
	buf := make([]byte, 100)
	n, err := f.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(buf[:n]))
}

func TestSafeReadFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := "test.txt"
	testPath := filepath.Join(tempDir, testFile)
	content := []byte("test content")
	err := os.WriteFile(testPath, content, 0644)
	require.NoError(t, err)

	// Read file safely
	readContent, err := SafeReadFile(testFile, tempDir)
	require.NoError(t, err)
	assert.Equal(t, content, readContent)
}

func TestSafeReadFile_PathTraversal(t *testing.T) {
	tempDir := t.TempDir()

	// Try to read file outside tempDir
	_, err := SafeReadFile("../../../etc/passwd", tempDir)
	require.Error(t, err)

	var pathErr *PathValidationError
	require.ErrorAs(t, err, &pathErr)
}

func TestSafeOpenFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := "test.txt"
	testPath := filepath.Join(tempDir, testFile)
	content := []byte("test content")
	err := os.WriteFile(testPath, content, 0644)
	require.NoError(t, err)

	// Open file safely
	file, err := SafeOpenFile(testFile, tempDir)
	require.NoError(t, err)
	require.NotNil(t, file)
	defer file.Close()

	// Verify it's a valid os.File
	info, err := file.Stat()
	require.NoError(t, err)
	assert.Equal(t, int64(len(content)), info.Size())
}

func TestPathValidator_SubdirectoryTraversal(t *testing.T) {
	tempDir := t.TempDir()

	// Create nested directories
	nestedPath := filepath.Join(tempDir, "a", "b", "c")
	err := os.MkdirAll(nestedPath, 0755)
	require.NoError(t, err)

	// Create a file in the nested directory
	testFile := filepath.Join("a", "b", "c", "test.txt")
	fullPath := filepath.Join(tempDir, testFile)
	err = os.WriteFile(fullPath, []byte("nested content"), 0644)
	require.NoError(t, err)

	pv, err := NewPathValidator(tempDir)
	require.NoError(t, err)
	defer pv.Close()

	// Should allow access to nested file
	f, err := pv.ValidatePath(testFile)
	require.NoError(t, err)
	defer f.Close()

	content, err := os.ReadFile(fullPath)
	require.NoError(t, err)
	assert.Equal(t, "nested content", string(content))
}

func TestPathValidator_DotSlashPrefix(t *testing.T) {
	tempDir := t.TempDir()

	// Create a test file
	testFile := "test.txt"
	testPath := filepath.Join(tempDir, testFile)
	err := os.WriteFile(testPath, []byte("test content"), 0644)
	require.NoError(t, err)

	pv, err := NewPathValidator(tempDir)
	require.NoError(t, err)
	defer pv.Close()

	// Should handle ./ prefix correctly
	f, err := pv.ValidatePath("./test.txt")
	require.NoError(t, err)
	defer f.Close()

	// Verify it works
	buf := make([]byte, 100)
	n, err := f.Read(buf)
	require.NoError(t, err)
	assert.Equal(t, "test content", string(buf[:n]))
}

func TestPathValidator_EmptyPath(t *testing.T) {
	tempDir := t.TempDir()

	pv, err := NewPathValidator(tempDir)
	require.NoError(t, err)
	defer pv.Close()

	// Empty path should be handled (likely as open error)
	_, err = pv.ValidatePath("")
	require.Error(t, err)
}

func TestPathValidator_DirectoryTraversal_WithDotDot(t *testing.T) {
	tempDir := t.TempDir()

	pv, err := NewPathValidator(tempDir)
	require.NoError(t, err)
	defer pv.Close()

	// Try path with .. in the middle
	_, err = pv.ValidatePath("subdir/../../etc/passwd")
	require.Error(t, err)

	var pathErr *PathValidationError
	require.ErrorAs(t, err, &pathErr)
	assert.Contains(t, pathErr.Reason, "escapes repo root")
}
