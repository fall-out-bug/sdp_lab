package session

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const (
	defaultMemoryStoragePath = ".sdp/mem"
	defaultMaxMemories       = 20
	defaultMaxProfileItems   = 10
)

type memoryConfig struct {
	StoragePath     string `json:"storagePath"`
	MaxMemories     int    `json:"maxMemories"`
	MaxProfileItems int    `json:"maxProfileItems"`
}

type profileData struct {
	Preferences []struct {
		Category    string   `json:"category"`
		Description string   `json:"description"`
		Confidence  float64  `json:"confidence"`
		Evidence    []string `json:"evidence"`
	} `json:"preferences"`
}

type profileRow struct {
	DisplayName string
	UserName    string
	UserEmail   string
	ProfileData string
}

type memoryRow struct {
	Content   string
	CreatedAt int64
}

func LoadMemoryContext(projectRoot string) (string, error) {
	if memoryDisabled() {
		return "", nil
	}

	storagePath, cfg, err := resolveMemoryStoragePath(projectRoot)
	if err != nil {
		return "", err
	}
	if storagePath == "" {
		return "", nil
	}

	if _, err := os.Stat(storagePath); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("stat memory storage: %w", err)
	}

	profileSection, err := loadProfileSection(storagePath, cfg.MaxProfileItems)
	if err != nil {
		return "", err
	}

	gitRepoURL, err := getGitRemoteURL(projectRoot)
	if err != nil {
		return "", err
	}
	memorySection, err := loadProjectMemorySection(storagePath, gitRepoURL, cfg.MaxMemories)
	if err != nil {
		return "", err
	}

	if profileSection == "" && memorySection == "" {
		return "", nil
	}

	var b strings.Builder
	b.WriteString("[MEMORY]\n")
	if profileSection != "" {
		b.WriteString(profileSection)
		b.WriteString("\n")
	}
	if memorySection != "" {
		b.WriteString(memorySection)
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String()), nil
}

func memoryDisabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("SDP_MEMORY_ENABLED")))
	return v == "0" || v == "false" || v == "no" || v == "off"
}

func resolveMemoryStoragePath(projectRoot string) (string, memoryConfig, error) {
	cfg := memoryConfig{
		StoragePath:     defaultMemoryStoragePath,
		MaxMemories:     defaultMaxMemories,
		MaxProfileItems: defaultMaxProfileItems,
	}

	cfgPath := filepath.Join(projectRoot, ".opencode", "mem-config.json")
	if data, err := os.ReadFile(cfgPath); err == nil {
		if err := json.Unmarshal(data, &cfg); err != nil {
			return "", cfg, fmt.Errorf("parse %s: %w", cfgPath, err)
		}
	}

	override := strings.TrimSpace(os.Getenv("SDP_MEMORY_PATH"))
	if override != "" {
		cfg.StoragePath = override
	}

	if cfg.StoragePath == "" {
		cfg.StoragePath = defaultMemoryStoragePath
	}
	if cfg.MaxMemories <= 0 {
		cfg.MaxMemories = defaultMaxMemories
	}
	if cfg.MaxProfileItems <= 0 {
		cfg.MaxProfileItems = defaultMaxProfileItems
	}

	path := cfg.StoragePath
	if !filepath.IsAbs(path) {
		path = filepath.Join(projectRoot, path)
	}
	return path, cfg, nil
}

func loadProfileSection(storagePath string, maxItems int) (string, error) {
	dbPath := filepath.Join(storagePath, "user-profiles.db")
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("stat profile db: %w", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return "", fmt.Errorf("open profile db: %w", err)
	}
	defer func() { _ = db.Close() }()

	row := db.QueryRow(`
		SELECT display_name, user_name, user_email, profile_data
		FROM user_profiles
		WHERE is_active = 1
		ORDER BY last_analyzed_at DESC
		LIMIT 1
	`)

	var p profileRow
	if err := row.Scan(&p.DisplayName, &p.UserName, &p.UserEmail, &p.ProfileData); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", fmt.Errorf("query profile: %w", err)
	}

	var pd profileData
	if err := json.Unmarshal([]byte(p.ProfileData), &pd); err != nil {
		return "", fmt.Errorf("decode profile_data: %w", err)
	}

	if len(pd.Preferences) == 0 {
		return "", nil
	}

	if maxItems > len(pd.Preferences) {
		maxItems = len(pd.Preferences)
	}

	var b strings.Builder
	b.WriteString("User Profile:\n")
	if p.DisplayName != "" {
		b.WriteString("- Display: ")
		b.WriteString(p.DisplayName)
		b.WriteString("\n")
	}
	if p.UserName != "" {
		b.WriteString("- Username: ")
		b.WriteString(p.UserName)
		b.WriteString("\n")
	}
	if p.UserEmail != "" {
		b.WriteString("- Email: ")
		b.WriteString(p.UserEmail)
		b.WriteString("\n")
	}
	b.WriteString("Preferences:\n")
	for i := range maxItems {
		pref := pd.Preferences[i]
		if pref.Description == "" {
			continue
		}
		b.WriteString("- ")
		if pref.Category != "" {
			b.WriteString("[")
			b.WriteString(pref.Category)
			b.WriteString("] ")
		}
		b.WriteString(pref.Description)
		if pref.Confidence > 0 {
			fmt.Fprintf(&b, " (%.2f)", pref.Confidence)
		}
		b.WriteString("\n")
	}

	return strings.TrimSpace(b.String()), nil
}

func loadProjectMemorySection(storagePath, gitRepoURL string, maxMemories int) (string, error) {
	metadataPath := filepath.Join(storagePath, "metadata.db")
	if _, err := os.Stat(metadataPath); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", fmt.Errorf("stat metadata db: %w", err)
	}

	metaDB, err := sql.Open("sqlite3", metadataPath)
	if err != nil {
		return "", fmt.Errorf("open metadata db: %w", err)
	}
	defer func() { _ = metaDB.Close() }()

	rows, err := metaDB.Query(`
		SELECT db_path
		FROM shards
		WHERE scope = 'project'
		ORDER BY shard_index DESC
	`)
	if err != nil {
		return "", fmt.Errorf("query project shards: %w", err)
	}
	defer func() { _ = rows.Close() }()

	remaining := maxMemories
	if remaining <= 0 {
		remaining = defaultMaxMemories
	}

	collected := make([]memoryRow, 0, remaining)
	for rows.Next() && remaining > 0 {
		var dbPath string
		if err := rows.Scan(&dbPath); err != nil {
			return "", fmt.Errorf("scan shard path: %w", err)
		}

		shardPath := filepath.Join(storagePath, filepath.FromSlash(dbPath))
		if _, err := os.Stat(shardPath); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("stat shard db %s: %w", shardPath, err)
		}

		shardRows, err := queryShardMemories(shardPath, gitRepoURL, remaining)
		if err != nil {
			return "", err
		}
		collected = append(collected, shardRows...)
		remaining = maxMemories - len(collected)
	}
	if err := rows.Err(); err != nil {
		return "", fmt.Errorf("iterate shard rows: %w", err)
	}

	if len(collected) == 0 {
		return "", nil
	}

	if len(collected) > maxMemories {
		collected = collected[:maxMemories]
	}

	var b strings.Builder
	b.WriteString("Project Memories:")
	for _, m := range collected {
		if strings.TrimSpace(m.Content) == "" {
			continue
		}
		b.WriteString("\n- ")
		if m.CreatedAt > 0 {
			tm := time.UnixMilli(m.CreatedAt).UTC()
			b.WriteString("[")
			b.WriteString(tm.Format(time.RFC3339))
			b.WriteString("] ")
		}
		b.WriteString(m.Content)
	}

	return strings.TrimSpace(b.String()), nil
}

func queryShardMemories(shardPath, gitRepoURL string, limit int) ([]memoryRow, error) {
	db, err := sql.Open("sqlite3", shardPath)
	if err != nil {
		return nil, fmt.Errorf("open shard db %s: %w", shardPath, err)
	}
	defer func() { _ = db.Close() }()

	const qWithRepo = `
		SELECT content, created_at
		FROM memories
		WHERE content <> ''
		  AND (? = '' OR git_repo_url = ?)
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := db.Query(qWithRepo, gitRepoURL, gitRepoURL, limit)
	if err != nil {
		return nil, fmt.Errorf("query shard memories %s: %w", shardPath, err)
	}
	defer func() { _ = rows.Close() }()

	out := make([]memoryRow, 0, limit)
	for rows.Next() {
		var row memoryRow
		if err := rows.Scan(&row.Content, &row.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan shard memory %s: %w", shardPath, err)
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate shard memories %s: %w", shardPath, err)
	}
	return out, nil
}

func getGitRemoteURL(projectRoot string) (string, error) {
	cmd := exec.CommandContext(context.Background(), "git", "-C", projectRoot, "config", "--get", "remote.origin.url")
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return "", fmt.Errorf("read git remote url: %s", strings.TrimSpace(string(exitErr.Stderr)))
		}
		return "", nil
	}
	return strings.TrimSpace(string(out)), nil
}
