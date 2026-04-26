package security

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckConfigFilePermissions(t *testing.T) {
	t.Run("NonExistentFile", func(t *testing.T) {
		warn, err := CheckConfigFilePermissions("nonexistent.yml")
		if err != nil {
			t.Fatalf("expected no error for nonexistent file, got: %v", err)
		}
		if warn != "" {
			t.Fatalf("expected no warning for nonexistent file, got: %s", warn)
		}
	})

	t.Run("WorldReadableFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "config.yml")

		// Create file with world-readable permissions
		if err := os.WriteFile(testFile, []byte("test: data"), 0o644); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		warn, err := CheckConfigFilePermissions(testFile)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if warn == "" {
			t.Error("expected warning for world-readable file, got empty string")
		}
	})

	t.Run("SecureFile", func(t *testing.T) {
		tmpDir := t.TempDir()
		testFile := filepath.Join(tmpDir, "config.yml")

		// Create file with restricted permissions
		if err := os.WriteFile(testFile, []byte("test: data"), 0o600); err != nil {
			t.Fatalf("failed to create test file: %v", err)
		}

		warn, err := CheckConfigFilePermissions(testFile)
		if err != nil {
			t.Fatalf("expected no error, got: %v", err)
		}
		if warn != "" {
			t.Errorf("expected no warning for secure file, got: %s", warn)
		}
	})
}

func TestCheckSDPConfigFiles(t *testing.T) {
	t.Run("NoConfigFiles", func(t *testing.T) {
		tmpDir := t.TempDir()
		warnings := CheckSDPConfigFiles(tmpDir)
		if len(warnings) != 0 {
			t.Errorf("expected no warnings, got %d: %v", len(warnings), warnings)
		}
	})

	t.Run("WorldReadableConfig", func(t *testing.T) {
		tmpDir := t.TempDir()
		sdpDir := filepath.Join(tmpDir, ".sdp")
		if err := os.Mkdir(sdpDir, 0o755); err != nil {
			t.Fatalf("failed to create .sdp dir: %v", err)
		}

		configFile := filepath.Join(sdpDir, "config.yml")
		if err := os.WriteFile(configFile, []byte("key: value"), 0o644); err != nil {
			t.Fatalf("failed to create config file: %v", err)
		}

		warnings := CheckSDPConfigFiles(tmpDir)
		if len(warnings) == 0 {
			t.Error("expected warning for world-readable config, got none")
		}
	})

	t.Run("SecureConfig", func(t *testing.T) {
		tmpDir := t.TempDir()
		sdpDir := filepath.Join(tmpDir, ".sdp")
		if err := os.Mkdir(sdpDir, 0o755); err != nil {
			t.Fatalf("failed to create .sdp dir: %v", err)
		}

		configFile := filepath.Join(sdpDir, "config.yml")
		if err := os.WriteFile(configFile, []byte("key: value"), 0o600); err != nil {
			t.Fatalf("failed to create config file: %v", err)
		}

		warnings := CheckSDPConfigFiles(tmpDir)
		if len(warnings) != 0 {
			t.Errorf("expected no warnings for secure config, got: %v", warnings)
		}
	})
}

func TestIsWindows(t *testing.T) {
	// Just ensure it doesn't panic
	_ = IsWindows()
}
