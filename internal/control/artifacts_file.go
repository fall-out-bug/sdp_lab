package control

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
)

// FileArtifactStore implements ArtifactStore using the local filesystem.
// Artifacts are stored under {projectRoot}/.sdp/artifacts/{cardID}/{type}/{filename}.
type FileArtifactStore struct {
	projectRoot string
}

// NewFileArtifactStore creates a new file-based artifact store.
func NewFileArtifactStore(projectRoot string) *FileArtifactStore {
	return &FileArtifactStore{projectRoot: projectRoot}
}

func (s *FileArtifactStore) artifactDir(cardID string, artType ArtifactType) string {
	return filepath.Join(s.projectRoot, ".sdp", "artifacts", cardID, string(artType))
}

func (s *FileArtifactStore) Store(ctx context.Context, cardID string, ref ArtifactRef, data []byte) (ArtifactRef, error) {
	if ctx.Err() != nil {
		return ArtifactRef{}, ctx.Err()
	}

	dir := s.artifactDir(cardID, ref.Type)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ArtifactRef{}, fmt.Errorf("mkdir artifact dir: %w", err)
	}

	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	filename := filepath.Base(ref.Path)
	if filename == "." || filename == "" {
		filename = fmt.Sprintf("%s_%x", ref.Type, hashStr[:8])
	}

	fullPath := filepath.Join(dir, filename)
	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return ArtifactRef{}, fmt.Errorf("write artifact: %w", err)
	}

	return ArtifactRef{
		Type:      ref.Type,
		Path:      fullPath,
		Hash:      hashStr,
		CreatedAt: ref.CreatedAt,
		Size:      int64(len(data)),
	}, nil
}

func (s *FileArtifactStore) Load(ctx context.Context, ref ArtifactRef) ([]byte, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	// ref.Path may be absolute or relative to projectRoot
	path := ref.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.projectRoot, ref.Path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read artifact %s: %w", path, err)
	}
	return data, nil
}

func (s *FileArtifactStore) List(ctx context.Context, cardID string) ([]ArtifactRef, error) {
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	baseDir := filepath.Join(s.projectRoot, ".sdp", "artifacts", cardID)
	var refs []ArtifactRef

	entries, err := os.ReadDir(baseDir)
	if os.IsNotExist(err) {
		return refs, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list artifacts: %w", err)
	}

	for _, typeEntry := range entries {
		if !typeEntry.IsDir() {
			continue
		}
		artType := ArtifactType(typeEntry.Name())
		typeDir := filepath.Join(baseDir, typeEntry.Name())

		files, err := os.ReadDir(typeDir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if f.IsDir() {
				continue
			}
			info, err := f.Info()
			if err != nil {
				continue
			}
			refs = append(refs, ArtifactRef{
				Type:      artType,
				Path:      filepath.Join(typeDir, f.Name()),
				CreatedAt: info.ModTime(),
				Size:      info.Size(),
			})
		}
	}

	return refs, nil
}

func (s *FileArtifactStore) Delete(ctx context.Context, ref ArtifactRef) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	path := ref.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(s.projectRoot, ref.Path)
	}

	if err := os.Remove(path); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("delete artifact: %w", err)
	}
	return nil
}
