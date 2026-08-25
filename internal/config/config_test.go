package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultPath_HonorsXDGConfigHome(t *testing.T) {
	xdg := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", xdg)

	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	want := filepath.Join(xdg, "dotmgr", "config.toml")
	if got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestDefaultPath_FallsBackToHomeConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	got, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	want := filepath.Join(home, ".config", "dotmgr", "config.toml")
	if got != want {
		t.Errorf("DefaultPath() = %q, want %q", got, want)
	}
}

func TestSaveAndLoad_RoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.toml")

	cfg := &Config{}
	cfg.Repo.Remote = "git@github.com:you/dotfiles.git"
	cfg.Repo.Path = "/home/you/.local/share/dotmgr/repo"
	cfg.Machine.Name = "workstation"

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if *got != *cfg {
		t.Errorf("round trip = %+v, want %+v", got, cfg)
	}
}

func TestLoad_SkipsCommentsAndBlankLinesAndUnknownKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	content := `# a leading comment

[repo]
# comment inside a section
remote = "git@github.com:you/dotfiles.git"
path   = "/home/you/repo"
unknown_key = "ignored"
this line has no equals sign

[machine]
name = "laptop"

[unknown_section]
name = "should not affect machine.name"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Repo.Remote != "git@github.com:you/dotfiles.git" {
		t.Errorf("Repo.Remote = %q", cfg.Repo.Remote)
	}
	if cfg.Repo.Path != "/home/you/repo" {
		t.Errorf("Repo.Path = %q", cfg.Repo.Path)
	}
	if cfg.Machine.Name != "laptop" {
		t.Errorf("Machine.Name = %q, want %q (should not be overwritten by the unknown section)", cfg.Machine.Name, "laptop")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err == nil {
		t.Fatal("expected error loading a missing file, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("error = %v, want it to satisfy os.ErrNotExist", err)
	}
}

func TestSave_CreatesParentDirs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "a", "b", "c", "config.toml")
	if err := Save(path, &Config{}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist: %v", err)
	}
}
