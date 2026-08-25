package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/config"
)

func TestLoadConfig_NotInitialized(t *testing.T) {
	newCLIEnv(t)

	_, err := loadConfig()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not initialized yet") {
		t.Errorf("error = %q, want it to mention dotmgr isn't initialized", err)
	}
}

func TestLoadConfig_Success(t *testing.T) {
	newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Remote = "git@github.com:you/dotfiles.git"
	cfg.Repo.Path = "/some/repo"
	writeConfig(t, cfg)

	got, err := loadConfig()
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if got.Repo.Remote != cfg.Repo.Remote || got.Repo.Path != cfg.Repo.Path {
		t.Errorf("loadConfig() = %+v, want %+v", got, cfg)
	}
}

func TestLoadConfig_OtherErrorIsPropagated(t *testing.T) {
	newCLIEnv(t)
	path, err := config.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	// A directory where the config file should be: os.Open succeeds but
	// reading it as a file fails, a different error than "not exist".
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err = loadConfig()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if strings.Contains(err.Error(), "not initialized yet") {
		t.Errorf("error = %q, want the raw underlying error, not the not-initialized message", err)
	}
}

func TestLoadConfig_DefaultPathError(t *testing.T) {
	newCLIEnv(t)
	t.Setenv("HOME", "")
	t.Setenv("XDG_CONFIG_HOME", "")

	if _, err := loadConfig(); err == nil {
		t.Fatal("expected error when neither $HOME nor $XDG_CONFIG_HOME is set, got nil")
	}
}
