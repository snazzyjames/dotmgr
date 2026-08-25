package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/config"
)

func TestCmdInit_FreshEmptyRemote(t *testing.T) {
	newCLIEnv(t)
	remote := newBareRemote(t)

	stdout, _, code := captureOutput(t, func() int { return cmdInit([]string{remote}) })
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stdout)
	}
	if !strings.Contains(stdout, "Done.") {
		t.Errorf("stdout = %q, want a completion message", stdout)
	}

	cfgPath, err := config.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatalf("Load written config: %v", err)
	}
	if cfg.Repo.Remote != remote {
		t.Errorf("Repo.Remote = %q, want %q", cfg.Repo.Remote, remote)
	}
	if _, err := os.Stat(filepath.Join(cfg.Repo.Path, "files")); err != nil {
		t.Errorf("expected files/ dir: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.Repo.Path, ".dotmgrignore")); err != nil {
		t.Errorf("expected default .dotmgrignore to be written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfg.Repo.Path, "README.md")); err != nil {
		t.Errorf("expected default README.md to be written: %v", err)
	}
}

func TestCmdInit_ExistingRemoteWithOwnFiles(t *testing.T) {
	newCLIEnv(t)
	remote := newBareRemote(t)

	// Seed the remote with its own .dotmgrignore and README, as if it were
	// an existing dotfiles repo from a previous "dotmgr init" elsewhere.
	seed := filepath.Join(t.TempDir(), "seed")
	runGit(t, t.TempDir(), "clone", "-q", remote, seed)
	configureIdentity(t, seed)
	writeFile(t, filepath.Join(seed, ".dotmgrignore"), "custom-pattern\n")
	writeFile(t, filepath.Join(seed, "README.md"), "custom readme\n")
	runGit(t, seed, "add", "-A")
	runGit(t, seed, "commit", "-q", "-m", "seed")
	runGit(t, seed, "push", "-q")

	stdout, _, code := captureOutput(t, func() int { return cmdInit([]string{remote}) })
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stdout)
	}
	if !strings.Contains(stdout, "cloned existing repo") {
		t.Errorf("stdout = %q, want it to mention cloning the existing repo", stdout)
	}

	cfgPath, _ := config.DefaultPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if got := mustReadFile(t, filepath.Join(cfg.Repo.Path, ".dotmgrignore")); got != "custom-pattern\n" {
		t.Errorf(".dotmgrignore = %q, want the repo's own content preserved, not overwritten", got)
	}
	if got := mustReadFile(t, filepath.Join(cfg.Repo.Path, "README.md")); got != "custom readme\n" {
		t.Errorf("README.md = %q, want the repo's own content preserved, not overwritten", got)
	}
}

func TestCmdInit_CloneFailureFallsBackToInitNew(t *testing.T) {
	newCLIEnv(t)
	remote := filepath.Join(t.TempDir(), "not-a-real-remote")

	stdout, _, code := captureOutput(t, func() int { return cmdInit([]string{remote}) })
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stdout)
	}
	if !strings.Contains(stdout, "clone failed") {
		t.Errorf("stdout = %q, want it to mention the fallback", stdout)
	}

	cfgPath, _ := config.DefaultPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.TrimSpace(runGit(t, cfg.Repo.Path, "remote", "get-url", "origin"))
	if got != remote {
		t.Errorf("origin = %q, want %q", got, remote)
	}
}

func TestCmdInit_AlreadyInitialized(t *testing.T) {
	newCLIEnv(t)
	remote := newBareRemote(t)
	if code := runQuiet(t, func() int { return cmdInit([]string{remote}) }); code != 0 {
		t.Fatalf("first cmdInit failed, code = %d", code)
	}

	_, stderr, code := captureOutput(t, func() int { return cmdInit([]string{remote}) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "already initialized") {
		t.Errorf("stderr = %q, want it to mention already being initialized", stderr)
	}
}

func TestCmdInit_WrongArgCount(t *testing.T) {
	newCLIEnv(t)
	_, stderr, code := captureOutput(t, func() int { return cmdInit(nil) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "expected exactly one argument") {
		t.Errorf("stderr = %q, want it to mention the argument count", stderr)
	}
}

func TestCmdInit_FlagParseError(t *testing.T) {
	newCLIEnv(t)
	_, _, code := captureOutput(t, func() int { return cmdInit([]string{"--bogus-flag"}) })
	if code != 2 {
		t.Errorf("code = %d, want 2", code)
	}
}
