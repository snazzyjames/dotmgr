package cli

import (
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/config"
)

// captureOutput redirects os.Stdout and os.Stderr for the duration of fn and
// returns what was written to each, plus fn's return value. Output is small
// in these tests, so no concurrent draining is needed.
func captureOutput(t *testing.T, fn func() int) (stdout, stderr string, code int) {
	t.Helper()

	outR, outW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	errR, errW, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	oldOut, oldErr := os.Stdout, os.Stderr
	os.Stdout, os.Stderr = outW, errW
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()

	code = fn()

	outW.Close()
	errW.Close()
	outBytes, _ := io.ReadAll(outR)
	errBytes, _ := io.ReadAll(errR)
	return string(outBytes), string(errBytes), code
}

// newCLIEnv points $HOME, $XDG_CONFIG_HOME, and $XDG_DATA_HOME at fresh temp
// dirs for the duration of the test, isolating dotmgr's config/data
// discovery from the host environment. It returns $HOME and the default
// local repo path cmdInit would pick.
func newCLIEnv(t *testing.T) (home, defaultRepoPath string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, ".config"))
	dataHome := filepath.Join(home, ".local", "share")
	t.Setenv("XDG_DATA_HOME", dataHome)
	return home, filepath.Join(dataHome, "dotmgr", "repo")
}

// writeConfig saves a config.Config for the current env (see newCLIEnv) so
// loadConfig() succeeds without going through cmdInit.
func writeConfig(t *testing.T, cfg *config.Config) {
	t.Helper()
	path, err := config.DefaultPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := config.Save(path, cfg); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func lexists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

func mustReadFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(b)
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func configureIdentity(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "config", "user.email", "dotmgr-test@example.com")
	runGit(t, dir, "config", "user.name", "dotmgr test")
	runGit(t, dir, "config", "commit.gpgsign", "false")
}

// newBareRemote creates an empty local bare repo to use as a git remote.
func newBareRemote(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	return dir
}
