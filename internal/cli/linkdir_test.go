package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/config"
)

func TestCmdAdd_LinkDir(t *testing.T) {
	home, repoPath := newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = repoPath
	writeConfig(t, cfg)
	writeFile(t, filepath.Join(home, ".config", "kitty", "kitty.conf"), "font_size 14\n")

	stdout, stderr, code := captureOutput(t, func() int { return cmdAdd([]string{"--link-dir", "~/.config/kitty"}) })
	if code != 0 {
		t.Fatalf("code = %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if !strings.Contains(stdout, "Linked ~/.config/kitty as a whole") {
		t.Errorf("stdout = %q", stdout)
	}

	stdout, _, _ = captureOutput(t, func() int { return cmdStatus(nil) })
	if !strings.Contains(stdout, "~/.config/kitty/  (linked as a whole)") {
		t.Errorf("status = %q, want the linked dir listed", stdout)
	}
}

func TestCmdAdd_LinkDirRejectsForce(t *testing.T) {
	home, repoPath := newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = repoPath
	writeConfig(t, cfg)
	writeFile(t, filepath.Join(home, ".config", "kitty", "kitty.conf"), "x")

	if code := runQuiet(t, func() int { return cmdAdd([]string{"--link-dir", "--force", "~/.config/kitty"}) }); code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
}

func TestCmdStatus_DivergedHintAndRelink(t *testing.T) {
	home, repoPath := newCLIEnv(t)
	cfg := &config.Config{}
	cfg.Repo.Path = repoPath
	writeConfig(t, cfg)
	target := filepath.Join(home, ".gitconfig")
	writeFile(t, target, "old\n")
	if code := runQuiet(t, func() int { return cmdAdd([]string{"~/.gitconfig"}) }); code != 0 {
		t.Fatal("setup add failed")
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	writeFile(t, target, "saved by a program\n")

	stdout, _, _ := captureOutput(t, func() int { return cmdStatus(nil) })
	if !strings.Contains(stdout, "diverged") || !strings.Contains(stdout, "dotmgr relink") {
		t.Errorf("status = %q, want diverged with a relink hint", stdout)
	}

	if code := runQuiet(t, func() int { return cmdRelink([]string{"~/.gitconfig"}) }); code != 0 {
		t.Fatalf("relink code = %d", code)
	}
	stdout, _, _ = captureOutput(t, func() int { return cmdStatus(nil) })
	if strings.Contains(stdout, "diverged") {
		t.Errorf("status after relink = %q", stdout)
	}
}

func TestCmdRelink_WrongArgCount(t *testing.T) {
	newCLIEnv(t)
	if code := runQuiet(t, func() int { return cmdRelink(nil) }); code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
}
