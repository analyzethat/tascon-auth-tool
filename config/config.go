package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const encryptedPrefix = "enc:"

type Config struct {
	Server   string `json:"server"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
}

func DefaultConfig() *Config {
	return &Config{
		Server:   "tascon.database.windows.net",
		Database: "dwh",
	}
}

func Load() (*Config, error) {
	configPath, err := getConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if os.IsNotExist(err) {
		return DefaultConfig(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse stored config
	var stored Config
	if err := json.Unmarshal(data, &stored); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Start with defaults for non-sensitive fields
	cfg := DefaultConfig()
	if stored.Server != "" {
		cfg.Server = stored.Server
	}
	if stored.Database != "" {
		cfg.Database = stored.Database
	}

	// Decrypt sensitive fields if master key is available
	key, hasKey := GetMasterKey()

	// Username
	if strings.HasPrefix(stored.Username, encryptedPrefix) && hasKey {
		decrypted, err := Decrypt(strings.TrimPrefix(stored.Username, encryptedPrefix), key)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt username: %w", err)
		}
		cfg.Username = decrypted
	} else if strings.HasPrefix(stored.Username, encryptedPrefix) {
		// Encrypted but no key - leave empty
		cfg.Username = ""
	} else {
		cfg.Username = stored.Username
	}

	// Password
	if strings.HasPrefix(stored.Password, encryptedPrefix) && hasKey {
		decrypted, err := Decrypt(strings.TrimPrefix(stored.Password, encryptedPrefix), key)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt password: %w", err)
		}
		cfg.Password = decrypted
	} else if strings.HasPrefix(stored.Password, encryptedPrefix) {
		// Encrypted but no key - leave empty
		cfg.Password = ""
	} else {
		cfg.Password = stored.Password
	}

	return cfg, nil
}

func (c *Config) Save() error {
	configPath, err := getConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Prepare stored config with encrypted fields
	stored := Config{
		Server:   c.Server,
		Database: c.Database,
	}

	// Encrypt sensitive fields if master key is available
	key, hasKey := GetMasterKey()

	if hasKey && c.Username != "" {
		encrypted, err := Encrypt(c.Username, key)
		if err != nil {
			return fmt.Errorf("failed to encrypt username: %w", err)
		}
		stored.Username = encryptedPrefix + encrypted
	} else {
		stored.Username = c.Username
	}

	if hasKey && c.Password != "" {
		encrypted, err := Encrypt(c.Password, key)
		if err != nil {
			return fmt.Errorf("failed to encrypt password: %w", err)
		}
		stored.Password = encryptedPrefix + encrypted
	} else {
		stored.Password = c.Password
	}

	data, err := json.MarshalIndent(stored, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write with restrictive permissions (owner only)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}

func getConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config dir: %w", err)
	}
	return filepath.Join(configDir, "powerbi-access-tool", "config.json"), nil
}
