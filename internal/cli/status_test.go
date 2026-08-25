package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/config"
)

func TestCmdStatus_EmptyRepo(t *testing.T) {
	newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = t.TempDir()
	if err := os.MkdirAll(filepath.Join(cfg.Repo.Path, "files"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeConfig(t, cfg)

	stdout, _, code := captureOutput(t, func() int { return cmdStatus(nil) })
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "nothing tracked yet") {
		t.Errorf("stdout = %q, want it to mention nothing is tracked", stdout)
	}
}

func TestCmdStatus_TrackedAndCandidates(t *testing.T) {
	home, repoPath := newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = repoPath
	writeConfig(t, cfg)

	// Already tracked and correctly linked.
	writeFile(t, filepath.Join(repoPath, "files", ".gitconfig"), "[user]\n")
	if code := runQuiet(t, func() int { return cmdInstall(nil) }); code != 0 {
		t.Fatalf("setup Install failed, code = %d", code)
	}

	// An untracked file a known tool (zsh) owns.
	writeFile(t, filepath.Join(home, ".zshrc"), "export PATH=$PATH\n")

	stdout, _, code := captureOutput(t, func() int { return cmdStatus(nil) })
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "ok") || !strings.Contains(stdout, ".gitconfig") {
		t.Errorf("stdout = %q, want it to list .gitconfig as ok", stdout)
	}
	if !strings.Contains(stdout, ".zshrc") {
		t.Errorf("stdout = %q, want it to list .zshrc as a candidate", stdout)
	}
}

func TestCmdStatus_LinkerError(t *testing.T) {
	newCLIEnv(t)
	_, stderr, code := captureOutput(t, func() int { return cmdStatus(nil) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "not initialized yet") {
		t.Errorf("stderr = %q, want it to mention dotmgr isn't initialized", stderr)
	}
}
