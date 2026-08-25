package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/config"
)

func TestCmdAdd_Success(t *testing.T) {
	home, repoPath := newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = repoPath
	writeConfig(t, cfg)
	writeFile(t, filepath.Join(home, ".gitconfig"), "[user]\n")

	stdout, _, code := captureOutput(t, func() int { return cmdAdd([]string{"~/.gitconfig"}) })
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Tracked .gitconfig") {
		t.Errorf("stdout = %q, want it to mention the tracked file", stdout)
	}
	if !lexists(filepath.Join(repoPath, "files", ".gitconfig")) {
		t.Error("file was not moved into the repo")
	}
}

func TestCmdAdd_ForceOverridesDenylist(t *testing.T) {
	home, repoPath := newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = repoPath
	writeConfig(t, cfg)
	writeFile(t, filepath.Join(home, ".ssh", "id_rsa"), "private key")

	_, stderr, code := captureOutput(t, func() int { return cmdAdd([]string{"~/.ssh/id_rsa"}) })
	if code != 1 {
		t.Fatalf("code = %d, want 1 (denylisted without --force)", code)
	}
	if !strings.Contains(stderr, "denylist") {
		t.Errorf("stderr = %q, want it to mention the denylist", stderr)
	}

	stdout, _, code := captureOutput(t, func() int { return cmdAdd([]string{"--force", "~/.ssh/id_rsa"}) })
	if code != 0 {
		t.Fatalf("code = %d, want 0 with --force", code)
	}
	if !strings.Contains(stdout, "Tracked") {
		t.Errorf("stdout = %q, want it to mention the tracked file", stdout)
	}
}

func TestCmdAdd_LinkerError(t *testing.T) {
	home, repoPath := newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = repoPath
	writeConfig(t, cfg)

	_, stderr, code := captureOutput(t, func() int { return cmdAdd([]string{"~/does-not-exist"}) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "no such file") {
		t.Errorf("stderr = %q, want it to mention the missing file", stderr)
	}
	_ = home
}

func TestCmdAdd_WrongArgCount(t *testing.T) {
	newCLIEnv(t)
	_, stderr, code := captureOutput(t, func() int { return cmdAdd(nil) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "expected exactly one path") {
		t.Errorf("stderr = %q, want it to mention the argument count", stderr)
	}
}

func TestCmdAdd_FlagParseError(t *testing.T) {
	newCLIEnv(t)
	_, _, code := captureOutput(t, func() int { return cmdAdd([]string{"--bogus-flag"}) })
	if code != 2 {
		t.Errorf("code = %d, want 2", code)
	}
}
