package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// legacyConfig is the old single-file format that contained both application
// settings and host/group data in one config.toml.
type legacyConfig struct {
	Version     int      `toml:"version"`
	HiddenHosts []string `toml:"hidden_hosts,omitempty"`
	Defaults    Defaults `toml:"defaults"`
	Hosts       []Host   `toml:"hosts"`
	Groups      []Group  `toml:"groups"`
}

// Migrate checks whether host/group data still lives in config.toml and, if
// so, extracts it into hosts.toml.  The migration is skipped when hosts.toml
// already exists.  Both files are written atomically.
//
// configPath is the resolved path to config.toml (must not be empty).
// hostsPath is the resolved path to hosts.toml (must not be empty).
func Migrate(configPath, hostsPath string) error {
	configPath = filepath.Clean(configPath)
	hostsPath = filepath.Clean(hostsPath)

	// If hosts.toml already exists, migration is not needed.
	if _, err := os.Stat(hostsPath); err == nil {
		return nil
	}

	// If config.toml does not exist, nothing to migrate.
	if _, err := os.Stat(configPath); err != nil {
		return nil
	}

	// Read the old config.toml into the legacy struct that has all fields.
	var legacy legacyConfig
	if _, err := toml.DecodeFile(configPath, &legacy); err != nil {
		return err
	}

	// Nothing to migrate if the old config has no host/group data.
	if len(legacy.Hosts) == 0 && len(legacy.Groups) == 0 && len(legacy.HiddenHosts) == 0 {
		return nil
	}

	// Build the new inventory from the legacy data.
	inv := Inventory{
		Version:     1,
		HiddenHosts: legacy.HiddenHosts,
		Hosts:       legacy.Hosts,
		Groups:      legacy.Groups,
	}
	if _, err := SaveInventory(hostsPath, inv); err != nil {
		return err
	}

	// Re-save config.toml without host/group data.
	cfg := Config{
		Version:  1,
		Defaults: legacy.Defaults,
	}
	if _, err := Save(configPath, cfg); err != nil {
		return err
	}

	return nil
}
