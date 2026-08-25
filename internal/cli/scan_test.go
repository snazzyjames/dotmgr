package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/config"
)

func TestCmdScan_NothingFound(t *testing.T) {
	newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = t.TempDir()
	writeConfig(t, cfg)

	stdout, _, code := captureOutput(t, func() int { return cmdScan(nil) })
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "nothing new found") {
		t.Errorf("stdout = %q, want it to say nothing new was found", stdout)
	}
}

func TestCmdScan_FindsCandidates(t *testing.T) {
	home, repoPath := newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = repoPath
	writeConfig(t, cfg)
	writeFile(t, filepath.Join(home, ".gitconfig"), "[user]\n")

	stdout, _, code := captureOutput(t, func() int { return cmdScan(nil) })
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout, ".gitconfig") {
		t.Errorf("stdout = %q, want it to list .gitconfig", stdout)
	}
	if !strings.Contains(stdout, "dotmgr add") {
		t.Errorf("stdout = %q, want it to hint at 'dotmgr add'", stdout)
	}
}

func TestCmdScan_LinkerError(t *testing.T) {
	newCLIEnv(t)
	_, stderr, code := captureOutput(t, func() int { return cmdScan(nil) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "not initialized yet") {
		t.Errorf("stderr = %q, want it to mention dotmgr isn't initialized", stderr)
	}
}
