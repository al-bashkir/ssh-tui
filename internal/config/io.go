package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

func DefaultPath() (string, error) {
	if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
		return filepath.Join(v, "ssh-tui", "config.toml"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if home == "" {
		return "", errors.New("home directory not found")
	}
	return filepath.Join(home, ".config", "ssh-tui", "config.toml"), nil
}

func Load(path string) (Config, string, error) {
	if path == "" {
		p, err := DefaultPath()
		if err != nil {
			return DefaultConfig(), "", err
		}
		path = p
	}

	path = filepath.Clean(path)
	st, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), path, nil
		}
		return DefaultConfig(), path, err
	}
	if st.IsDir() {
		return DefaultConfig(), path, fmt.Errorf("config path is a directory: %s", path)
	}

	cfg := DefaultConfig()
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return DefaultConfig(), path, err
	}

	if cfg.Version == 0 {
		cfg.Version = 1
	}
	for _, g := range cfg.Groups {
		if err := ValidateGroupName(g.Name); err != nil {
			return DefaultConfig(), path, fmt.Errorf("config: group %q: %w", g.Name, err)
		}
	}
	return cfg, path, nil
}

func Save(path string, cfg Config) (string, error) {
	if path == "" {
		p, err := DefaultPath()
		if err != nil {
			return "", err
		}
		path = p
	}

	path = filepath.Clean(path)
	cfg.Version = 1

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return path, err
	}

	tmp, err := os.CreateTemp(dir, ".config.toml.*")
	if err != nil {
		return path, err
	}
	tmpPath := filepath.Clean(tmp.Name())
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	enc := toml.NewEncoder(tmp)
	if err := enc.Encode(cfg); err != nil {
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
