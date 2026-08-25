package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/config"
)

func TestCmdInstall_Success(t *testing.T) {
	home, repoPath := newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = repoPath
	writeConfig(t, cfg)
	writeFile(t, filepath.Join(repoPath, "files", ".gitconfig"), "[user]\n")

	stdout, _, code := captureOutput(t, func() int { return cmdInstall(nil) })
	if code != 0 {
		t.Fatalf("code = %d, want 0: %s", code, stdout)
	}
	if !strings.Contains(stdout, "Linked  : 1") {
		t.Errorf("stdout = %q, want it to report one linked file", stdout)
	}
	if !strings.Contains(stdout, "Done.") {
		t.Errorf("stdout = %q, want a completion message", stdout)
	}
	if !lexists(filepath.Join(home, ".gitconfig")) {
		t.Error("expected .gitconfig to be linked into $HOME")
	}
}

func TestCmdInstall_DryRunFlags(t *testing.T) {
	for _, flag := range []string{"--dry-run", "-n"} {
		t.Run(flag, func(t *testing.T) {
			home, repoPath := newCLIEnv(t)
			cfg := &config.Config{}
			cfg.Repo.Path = repoPath
			writeConfig(t, cfg)
			writeFile(t, filepath.Join(repoPath, "files", ".gitconfig"), "[user]\n")

			stdout, _, code := captureOutput(t, func() int { return cmdInstall([]string{flag}) })
			if code != 0 {
				t.Fatalf("code = %d, want 0", code)
			}
			if !strings.Contains(stdout, "dry run") {
				t.Errorf("stdout = %q, want it to mention the dry run", stdout)
			}
			if lexists(filepath.Join(home, ".gitconfig")) {
				t.Error("dry run should not have created a symlink")
			}
		})
	}
}

func TestCmdInstall_ReportsFailures(t *testing.T) {
	home, repoPath := newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = repoPath
	writeConfig(t, cfg)
	// A tracked file whose target parent can't be created (blocked by a
	// same-named regular file), and one unrelated file that should still
	// succeed.
	writeFile(t, filepath.Join(repoPath, "files", "blocked", "file.txt"), "x")
	writeFile(t, filepath.Join(repoPath, "files", ".gitconfig"), "[user]\n")
	if err := os.WriteFile(filepath.Join(home, "blocked"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := captureOutput(t, func() int { return cmdInstall(nil) })
	if code != 1 {
		t.Fatalf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout, "Failed  : 1") {
		t.Errorf("stdout = %q, want it to report one failure", stdout)
	}
}

func TestCmdInstall_LinkerError(t *testing.T) {
	newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = filepath.Join(t.TempDir(), "never-initialized")
	writeConfig(t, cfg)

	_, stderr, code := captureOutput(t, func() int { return cmdInstall(nil) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if stderr == "" {
		t.Error("expected an error message on stderr")
	}
}

func TestCmdInstall_FlagParseError(t *testing.T) {
	newCLIEnv(t)
	_, _, code := captureOutput(t, func() int { return cmdInstall([]string{"--bogus-flag"}) })
	if code != 2 {
		t.Errorf("code = %d, want 2", code)
	}
}
