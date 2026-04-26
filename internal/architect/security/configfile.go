package security

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// CheckConfigFilePermissions verifies that sensitive config files are not
// world-readable. Returns a warning message if permissions are too open,
// empty string if OK, or error if the file cannot be checked.
func CheckConfigFilePermissions(configPath string) (string, error) {
	info, err := os.Stat(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil // File doesn't exist, no warning
		}
		return "", fmt.Errorf("stat config: %w", err)
	}

	// Check if file is world-readable
	mode := info.Mode()
	if mode.Perm()&0o044 != 0 {
		return fmt.Sprintf("WARNING: Config file %s is world-readable (permissions: %s). "+
			"This may expose secrets. Run: chmod go-r %s",
			configPath, mode, configPath), nil
	}

	return "", nil
}

// CheckSDPConfigFiles scans common SDP config files for permission issues.
// Returns a list of warnings (empty if all OK).
func CheckSDPConfigFiles(repoRoot string) []string {
	configs := []string{
		".sdp/config.yml",
		".sdp/config.yaml",
		".sdp/credentials.yml",
		".sdp/credentials.yaml",
		".sdp/secrets.yml",
		".sdp/secrets.yaml",
	}

	var warnings []string
	for _, cfg := range configs {
		fullPath := filepath.Join(repoRoot, cfg)
		warn, err := CheckConfigFilePermissions(fullPath)
		if err != nil {
			// Log but don't fail on errors checking individual files
			warnings = append(warnings, fmt.Sprintf("ERROR checking %s: %v", cfg, err))
		} else if warn != "" {
			warnings = append(warnings, warn)
		}
	}

	return warnings
}

// IsWindows returns true if running on Windows.
// Windows doesn't support Unix file permissions in the same way.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}
