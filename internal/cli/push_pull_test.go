package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/config"
)

func TestCmdPush_Success(t *testing.T) {
	newCLIEnv(t)
	remote := newBareRemote(t)
	local := filepath.Join(t.TempDir(), "local")
	runGit(t, t.TempDir(), "clone", "-q", remote, local)
	configureIdentity(t, local)
	writeFile(t, filepath.Join(local, "f.txt"), "content\n")

	cfg := &config.Config{}
	cfg.Repo.Path = local
	writeConfig(t, cfg)

	stdout, _, code := captureOutput(t, func() int { return cmdPush(nil) })
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "done") {
		t.Errorf("stdout = %q, want a completion message", stdout)
	}

	check := filepath.Join(t.TempDir(), "check")
	runGit(t, t.TempDir(), "clone", "-q", remote, check)
	if _, err := os.Stat(filepath.Join(check, "f.txt")); err != nil {
		t.Errorf("pushed file not present on remote: %v", err)
	}
}

func TestCmdPush_GitError(t *testing.T) {
	newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = t.TempDir() // not a git repo at all
	writeConfig(t, cfg)

	_, stderr, code := captureOutput(t, func() int { return cmdPush(nil) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if stderr == "" {
		t.Error("expected an error message on stderr")
	}
}

func TestCmdPull_Success(t *testing.T) {
	newCLIEnv(t)
	remote := newBareRemote(t)

	writer := filepath.Join(t.TempDir(), "writer")
	runGit(t, t.TempDir(), "clone", "-q", remote, writer)
	configureIdentity(t, writer)
	writeFile(t, filepath.Join(writer, "f.txt"), "v1\n")
	runGit(t, writer, "add", "-A")
	runGit(t, writer, "commit", "-q", "-m", "v1")
	runGit(t, writer, "push", "-q")

	reader := filepath.Join(t.TempDir(), "reader")
	runGit(t, t.TempDir(), "clone", "-q", remote, reader)

	writeFile(t, filepath.Join(writer, "f.txt"), "v2\n")
	runGit(t, writer, "commit", "-q", "-am", "v2")
	runGit(t, writer, "push", "-q")

	cfg := &config.Config{}
	cfg.Repo.Path = reader
	writeConfig(t, cfg)

	stdout, _, code := captureOutput(t, func() int { return cmdPull(nil) })
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "done") {
		t.Errorf("stdout = %q, want a completion message", stdout)
	}
	if got := mustReadFile(t, filepath.Join(reader, "f.txt")); got != "v2\n" {
		t.Errorf("f.txt = %q, want %q", got, "v2\n")
	}
}

func TestCmdPull_GitError(t *testing.T) {
	newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = t.TempDir() // not a git repo at all
	writeConfig(t, cfg)

	_, stderr, code := captureOutput(t, func() int { return cmdPull(nil) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if stderr == "" {
		t.Error("expected an error message on stderr")
	}
}
