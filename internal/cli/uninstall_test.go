package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/config"
)

// setupInitialized runs cmdInit against a fresh bare remote, tracks one file,
// and pushes, leaving a clean, fully pushed clone.
func setupInitialized(t *testing.T) (home, repoPath string) {
	t.Helper()
	home, repoPath = newCLIEnv(t)
	remote := newBareRemote(t)
	if code := runQuiet(t, func() int { return cmdInit([]string{remote}) }); code != 0 {
		t.Fatalf("setup init failed, code = %d", code)
	}
	configureIdentity(t, repoPath)
	writeFile(t, filepath.Join(home, ".gitconfig"), "[user]\n")
	if code := runQuiet(t, func() int { return cmdAdd([]string{"~/.gitconfig"}) }); code != 0 {
		t.Fatalf("setup add failed, code = %d", code)
	}
	if code := runQuiet(t, func() int { return cmdPush(nil) }); code != 0 {
		t.Fatalf("setup push failed, code = %d", code)
	}
	return home, repoPath
}

func withStdin(t *testing.T, input string) {
	t.Helper()
	old := stdin
	stdin = strings.NewReader(input)
	t.Cleanup(func() { stdin = old })
}

func TestCmdUninstall_Success(t *testing.T) {
	home, repoPath := setupInitialized(t)
	cfgPath, _ := config.DefaultPath()

	stdout, stderr, code := captureOutput(t, func() int { return cmdUninstall([]string{"--yes"}) })
	if code != 0 {
		t.Fatalf("code = %d, want 0\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}

	info, err := os.Lstat(filepath.Join(home, ".gitconfig"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("~/.gitconfig is still a symlink")
	}
	if got := mustReadFile(t, filepath.Join(home, ".gitconfig")); got != "[user]\n" {
		t.Errorf("content = %q", got)
	}
	if lexists(repoPath) {
		t.Error("local clone was not deleted")
	}
	if lexists(filepath.Dir(repoPath)) {
		t.Error("empty dotmgr data dir was not deleted")
	}
	if lexists(cfgPath) {
		t.Error("config was not deleted")
	}
	if !strings.Contains(stdout, "To remove the dotmgr binary itself") {
		t.Errorf("stdout = %q, want binary removal hint", stdout)
	}
}

func TestCmdUninstall_DeclinedPromptChangesNothing(t *testing.T) {
	home, repoPath := setupInitialized(t)
	withStdin(t, "n\n")

	stdout, _, code := captureOutput(t, func() int { return cmdUninstall(nil) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout, "Aborted") {
		t.Errorf("stdout = %q, want it to say it aborted", stdout)
	}
	info, _ := os.Lstat(filepath.Join(home, ".gitconfig"))
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("symlink was replaced despite declining")
	}
	if !lexists(repoPath) {
		t.Error("clone was deleted despite declining")
	}
}

func TestCmdUninstall_ConfirmedPrompt(t *testing.T) {
	_, repoPath := setupInitialized(t)
	withStdin(t, "y\n")

	if code := runQuiet(t, func() int { return cmdUninstall(nil) }); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if lexists(repoPath) {
		t.Error("local clone was not deleted")
	}
}

func TestCmdUninstall_DryRun(t *testing.T) {
	home, repoPath := setupInitialized(t)

	stdout, _, code := captureOutput(t, func() int { return cmdUninstall([]string{"--dry-run"}) })
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "~/.gitconfig") {
		t.Errorf("stdout = %q, want it to list ~/.gitconfig", stdout)
	}
	info, _ := os.Lstat(filepath.Join(home, ".gitconfig"))
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("dry run replaced the symlink")
	}
	if !lexists(repoPath) {
		t.Error("dry run deleted the clone")
	}
}

func TestCmdUninstall_RefusesUnpushedChanges(t *testing.T) {
	home, repoPath := setupInitialized(t)
	writeFile(t, filepath.Join(home, ".zshrc"), "export X=1\n")
	if code := runQuiet(t, func() int { return cmdAdd([]string{"~/.zshrc"}) }); code != 0 {
		t.Fatalf("setup add failed, code = %d", code)
	}

	_, stderr, code := captureOutput(t, func() int { return cmdUninstall([]string{"--yes"}) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "dotmgr push") {
		t.Errorf("stderr = %q, want it to suggest dotmgr push", stderr)
	}
	if !lexists(repoPath) {
		t.Error("clone was deleted despite unpushed changes")
	}
}

func TestCmdUninstall_ForceIgnoresUnpushedChanges(t *testing.T) {
	home, repoPath := setupInitialized(t)
	writeFile(t, filepath.Join(home, ".zshrc"), "export X=1\n")
	if code := runQuiet(t, func() int { return cmdAdd([]string{"~/.zshrc"}) }); code != 0 {
		t.Fatalf("setup add failed, code = %d", code)
	}

	if code := runQuiet(t, func() int { return cmdUninstall([]string{"--yes", "--force"}) }); code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if got := mustReadFile(t, filepath.Join(home, ".zshrc")); got != "export X=1\n" {
		t.Errorf("~/.zshrc content = %q", got)
	}
	if lexists(repoPath) {
		t.Error("local clone was not deleted")
	}
}

func TestCmdUninstall_RefusesSuspiciousRepoPath(t *testing.T) {
	home, _ := newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = home
	writeConfig(t, cfg)

	_, stderr, code := captureOutput(t, func() int { return cmdUninstall([]string{"--yes", "--force"}) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "refusing to uninstall") {
		t.Errorf("stderr = %q, want a refusal", stderr)
	}
	if !lexists(home) {
		t.Fatal("home was deleted")
	}
}

func TestCmdUninstall_NotInitialized(t *testing.T) {
	newCLIEnv(t)
	_, stderr, code := captureOutput(t, func() int { return cmdUninstall([]string{"--yes"}) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "not initialized") {
		t.Errorf("stderr = %q, want not-initialized error", stderr)
	}
}

func TestCmdUninstall_FlagParseError(t *testing.T) {
	newCLIEnv(t)
	if code := runQuiet(t, func() int { return cmdUninstall([]string{"--bogus-flag"}) }); code != 2 {
		t.Errorf("code = %d, want 2", code)
	}
}
