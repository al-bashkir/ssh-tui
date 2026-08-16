package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func configDir() (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "ssh-tui"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", errors.New("home directory not found")
	}
	return filepath.Join(home, ".config", "ssh-tui"), nil
}

func DefaultPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

// defaultHostsPath returns the default path for the hosts inventory file.
func defaultHostsPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "hosts.toml"), nil
}

// HostsPathFromConfigPath derives the hosts.toml path from a config.toml path
// by replacing the filename in the same directory.
func HostsPathFromConfigPath(configPath string) string {
	return filepath.Join(filepath.Dir(configPath), "hosts.toml")
}

// loadTOML decodes path over the defaults in v. A missing file is not an
// error: v keeps its defaults. kind names the file in error messages.
func loadTOML[T any](path string, defaultPath func() (string, error), kind string, v *T) (string, error) {
	if path == "" {
		p, err := defaultPath()
		if err != nil {
			return "", err
		}
		path = p
	}

	path = filepath.Clean(path)
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return path, nil
		}
		return path, err
	}
	if st.IsDir() {
		return path, fmt.Errorf("%s path is a directory: %s", kind, path)
	}

	_, err = toml.DecodeFile(path, v)
	return path, err
}

// saveTOML writes v to path atomically, with 0600 permissions.
func saveTOML[T any](path string, defaultPath func() (string, error), tmpPattern string, v T) (string, error) {
	if path == "" {
		p, err := defaultPath()
		if err != nil {
			return "", err
		}
		path = p
	}
	path = filepath.Clean(path)

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return path, err
	}

	tmp, err := os.CreateTemp(dir, tmpPattern)
	if err != nil {
		return path, err
	}
	tmpPath := filepath.Clean(tmp.Name())
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if err := toml.NewEncoder(tmp).Encode(v); err != nil {
		return path, err
	}
	if err := tmp.Sync(); err != nil {
		return path, err
	}
	if err := tmp.Close(); err != nil {
		return path, err
	}

	// #nosec G703 -- path is sanitized via filepath.Clean above; taint propagation is a gosec limitation.
	if err := os.Rename(tmpPath, path); err != nil {
		return path, err
	}
	if err := os.Chmod(path, 0o600); err != nil {
		return path, err
	}
	return path, nil
}

// Load reads config.toml. If path is empty, DefaultPath is used.
// A missing file is not an error — DefaultConfig is returned.
func Load(path string) (Config, string, error) {
	cfg := DefaultConfig()
	used, err := loadTOML(path, DefaultPath, "config", &cfg)
	if err != nil {
		return DefaultConfig(), used, err
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	return cfg, used, nil
}

// Save atomically writes the config to path.
func Save(path string, cfg Config) (string, error) {
	cfg.Version = 1
	return saveTOML(path, DefaultPath, ".config.toml.*", cfg)
}

// LoadInventory loads the hosts/groups inventory from path.
// If path is empty, defaultHostsPath is used.
// A missing file is not an error — DefaultInventory is returned.
func LoadInventory(path string) (Inventory, string, error) {
	inv := DefaultInventory()
	used, err := loadTOML(path, defaultHostsPath, "hosts", &inv)
	if err != nil {
		return DefaultInventory(), used, err
	}
	if inv.Version == 0 {
		inv.Version = 1
	}
	for _, g := range inv.Groups {
		if err := ValidateGroupName(g.Name); err != nil {
			return DefaultInventory(), used, fmt.Errorf("hosts: group %q: %w", g.Name, err)
		}
	}
	return inv, used, nil
}

// SaveInventory atomically writes the inventory to path.
// If path is empty, defaultHostsPath is used.
func SaveInventory(path string, inv Inventory) (string, error) {
	inv.Version = 1
	return saveTOML(path, defaultHostsPath, ".hosts.toml.*", inv)
}
