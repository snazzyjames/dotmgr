package cli

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/config"
)

func TestCmdRemove_Success(t *testing.T) {
	home, repoPath := newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = repoPath
	writeConfig(t, cfg)
	writeFile(t, filepath.Join(home, ".gitconfig"), "[user]\n")

	if code := runQuiet(t, func() int { return cmdAdd([]string{"~/.gitconfig"}) }); code != 0 {
		t.Fatalf("setup Add failed, code = %d", code)
	}

	stdout, _, code := captureOutput(t, func() int { return cmdRemove([]string{"~/.gitconfig"}) })
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Untracked .gitconfig") {
		t.Errorf("stdout = %q, want it to mention the untracked file", stdout)
	}
	if !lexists(filepath.Join(home, ".gitconfig")) {
		t.Error("file was not moved back to $HOME")
	}
}

func TestCmdRemove_LinkerError(t *testing.T) {
	_, repoPath := newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = repoPath
	writeConfig(t, cfg)

	_, stderr, code := captureOutput(t, func() int { return cmdRemove([]string{"~/does-not-exist"}) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "no such file") {
		t.Errorf("stderr = %q, want it to mention the missing file", stderr)
	}
}

func TestCmdRemove_WrongArgCount(t *testing.T) {
	newCLIEnv(t)
	_, stderr, code := captureOutput(t, func() int { return cmdRemove(nil) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "expected exactly one path") {
		t.Errorf("stderr = %q, want it to mention the argument count", stderr)
	}
}

func TestCmdRemove_FlagParseError(t *testing.T) {
	newCLIEnv(t)
	_, _, code := captureOutput(t, func() int { return cmdRemove([]string{"--bogus-flag"}) })
	if code != 2 {
		t.Errorf("code = %d, want 2", code)
	}
}

// runQuiet runs fn with stdout/stderr discarded, for setup steps whose
// output isn't under test.
func runQuiet(t *testing.T, fn func() int) int {
	t.Helper()
	_, _, code := captureOutput(t, fn)
	return code
}
