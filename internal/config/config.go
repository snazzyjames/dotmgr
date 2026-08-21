// Package config reads and writes dotmgr's own settings file
// (~/.config/dotmgr/config.toml) — where the dotfiles repo lives locally
// and which git remote it tracks.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Repo struct {
		Remote string
		Path   string
	}
	Machine struct {
		Name string
	}
}

// DefaultPath returns ~/.config/dotmgr/config.toml, honoring $XDG_CONFIG_HOME.
func DefaultPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "dotmgr", "config.toml"), nil
}

// Load reads a config file. Only the flat [section]/key = "value" shape this
// package itself writes is supported — it is not a general TOML parser.
func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &Config{}
	section := ""
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSuffix(strings.TrimPrefix(line, "["), "]")
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"`)

		switch section {
		case "repo":
			switch key {
			case "remote":
				cfg.Repo.Remote = value
			case "path":
				cfg.Repo.Path = value
			}
		case "machine":
			if key == "name" {
				cfg.Machine.Name = value
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes cfg to path, creating parent directories as needed.
func Save(path string, cfg *Config) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	var b strings.Builder
	fmt.Fprintln(&b, "[repo]")
	fmt.Fprintf(&b, "remote = %q\n", cfg.Repo.Remote)
	fmt.Fprintf(&b, "path   = %q\n", cfg.Repo.Path)
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "[machine]")
	fmt.Fprintf(&b, "name = %q\n", cfg.Machine.Name)

	return os.WriteFile(path, []byte(b.String()), 0o644)
}
