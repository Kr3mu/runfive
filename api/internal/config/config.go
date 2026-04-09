// Package config provides application configuration loaded from environment variables.
//
// Encryption keys are auto-generated on first startup if not provided,
// then persisted to a local key file for subsequent runs.
package config

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
)

// Config holds all runtime configuration for the application.
//
// TODO: Add ServersDir field (path to the servers/ directory). On startup,
// scan this directory for subdirectories containing server.toml files.
// Each TOML defines a managed FiveM server instance (name, port, paths,
// resource config, etc.). Needs a typed ServerConfig struct for the TOML
// schema and a loader that returns []ServerConfig keyed by directory name.
//
// TODO: Add ArtifactsDir field for the shared cfx.re server-binary pool.
// Artifacts (cfx binaries) are shared across all managed servers — one
// download, many servers reference it via an `artifact_version` field in
// their server.toml. Needs integration with Kr3mu's existing artifact
// scraper: expose a management endpoint to list/download/select available
// artifact versions. Note: resources (scripts) are per-server and live
// inside each servers/<name>/ dir — artifacts are the only shared asset.
type Config struct {
	// HTTP listen port
	Port string
	// SessionEncryptKey is the AES-256 key used to encrypt session data at rest in the sessions table.
	SessionEncryptKey [32]byte
	// CfxAPIKeySecret is the AES-256 key used to encrypt stored Cfx.re API keys in the users table.
	CfxAPIKeySecret [32]byte
	// BaseURL is the public base URL used for OAuth redirect URIs (e.g. "http://localhost:5000").
	BaseURL string
}

// LoadConfig reads configuration from environment variables.
//
// If SESSION_ENCRYPT_KEY or CFX_API_KEY_SECRET are not set, random 32-byte
// keys are generated and persisted to ".runfive-keys" beside the binary.
func LoadConfig() (*Config, error) {
	cfg := &Config{
		Port:    envOrDefault("PORT", "5000"),
		BaseURL: envOrDefault("BASE_URL", "http://localhost:5000"),
	}

	sessionKey, err := loadOrGenerateKey("SESSION_ENCRYPT_KEY", "session_encrypt_key")
	if err != nil {
		return nil, fmt.Errorf("session encrypt key: %w", err)
	}
	copy(cfg.SessionEncryptKey[:], sessionKey)

	cfxKey, err := loadOrGenerateKey("CFX_API_KEY_SECRET", "cfx_api_key_secret")
	if err != nil {
		return nil, fmt.Errorf("cfx api key secret: %w", err)
	}
	copy(cfg.CfxAPIKeySecret[:], cfxKey)

	return cfg, nil
}

func envOrDefault(key string, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// loadOrGenerateKey checks the environment variable first, then falls back
// to the key file. If neither exists, a new random key is generated and
// written to the key file.
func loadOrGenerateKey(envVar string, fileKey string) ([]byte, error) {
	if v := os.Getenv(envVar); v != "" {
		decoded, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return nil, fmt.Errorf("decode %s: %w", envVar, err)
		}
		if len(decoded) != 32 {
			return nil, fmt.Errorf("%s must be exactly 32 bytes (got %d)", envVar, len(decoded))
		}
		return decoded, nil
	}

	keyFilePath := keyFilePath()
	entries := loadKeyFile(keyFilePath)

	if val, ok := entries[fileKey]; ok {
		decoded, err := base64.StdEncoding.DecodeString(val)
		if err != nil {
			return nil, fmt.Errorf("decode key file entry %s: %w", fileKey, err)
		}
		if len(decoded) == 32 {
			return decoded, nil
		}
	}

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate random key: %w", err)
	}

	entries[fileKey] = base64.StdEncoding.EncodeToString(key)
	if err := saveKeyFile(keyFilePath, entries); err != nil {
		return nil, fmt.Errorf("persist key file: %w", err)
	}

	return key, nil
}

func keyFilePath() string {
	return ".runfive-keys"
}

func loadKeyFile(path string) map[string]string {
	entries := make(map[string]string)
	data, err := os.ReadFile(path) //nolint:gosec // path is hardcoded to .runfive-keys, not user input
	if err != nil {
		return entries
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			entries[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return entries
}

func saveKeyFile(path string, entries map[string]string) error {
	lines := make([]string, 0, len(entries))
	for k, v := range entries {
		lines = append(lines, k+"="+v)
	}
	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}
