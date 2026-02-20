package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadMissingReturnsDefaults(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "missing.toml")

	cfg, used, err := Load(p)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if used != p {
		t.Fatalf("used=%q, want %q", used, p)
	}
	if !reflect.DeepEqual(cfg, DefaultConfig()) {
		t.Fatalf("cfg=%#v, want defaults", cfg)
	}
}

func TestSaveThenLoadRoundTrip(t *testing.T) {
	d := t.TempDir()
	p := filepath.Join(d, "config.toml")

	cfg := DefaultConfig()
	cfg.Defaults.User = "me"
	cfg.Hosts = []Host{{
		Host:         "db1.example",
		User:         "ubuntu",
		Port:         2222,
		IdentityFile: "~/.ssh/db1_ed25519",
		ExtraArgs:    []string{"-o", "ServerAliveInterval=30"},
	}}
	cfg.Groups = []Group{{
		Name:         "prod",
		User:         "admin",
		Port:         22,
		IdentityFile: "~/.ssh/prod_ed25519",
		ExtraArgs:    []string{"-o", "ServerAliveInterval=30"},
		OpenMode:     "tmux-window",
		Hosts:        []string{"db1.example", "[10.0.0.1]:2222"},
	}}

	if _, err := Save(p, cfg); err != nil {
		t.Fatalf("Save error: %v", err)
	}

	st, err := os.Stat(p)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Fatalf("mode=%o, want 600", st.Mode().Perm())
	}

	got, used, err := Load(p)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}
	if used != p {
		t.Fatalf("used=%q, want %q", used, p)
	}

	// Save normalizes version.
	cfg.Version = 1
	if !reflect.DeepEqual(got, cfg) {
		t.Fatalf("got=%#v\nwant=%#v", got, cfg)
	}
}
