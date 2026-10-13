package linker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestUninstall_ReplacesSymlinkWithCopy(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	target := filepath.Join(home, ".config", "fish", "config.fish")
	writeFile(t, target, "set -x EDITOR vim\n")
	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Add(cfg, target, false); err != nil {
		t.Fatalf("setup Add: %v", err)
	}

	res, err := Uninstall(cfg, false, false)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if len(res.Restored) != 1 || res.Failed != 0 {
		t.Fatalf("res = %+v, want 1 restored, 0 failed", res)
	}

	info, err := os.Lstat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("target is still a symlink")
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("mode = %v, want 0600", info.Mode().Perm())
	}
	if got := mustReadFile(t, target); got != "set -x EDITOR vim\n" {
		t.Errorf("content = %q", got)
	}
	// Copied, not moved: the clone still has it until the caller deletes it.
	if !lexists(t, filepath.Join(cfg.Repo.Path, "files", ".config", "fish", "config.fish")) {
		t.Error("repo copy was removed; Uninstall should copy, not move")
	}
}

func TestUninstall_DryRunChangesNothing(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	target := filepath.Join(home, ".gitconfig")
	writeFile(t, target, "[user]\n")
	if _, err := Add(cfg, target, false); err != nil {
		t.Fatalf("setup Add: %v", err)
	}

	res, err := Uninstall(cfg, false, true)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if len(res.Restored) != 1 {
		t.Errorf("Restored = %v, want 1 entry", res.Restored)
	}
	info, _ := os.Lstat(target)
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("dry run replaced the symlink")
	}
}

func TestUninstall_SkipsFilesNotLinked(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", ".gitconfig"), "repo\n")
	writeFile(t, filepath.Join(home, ".gitconfig"), "unrelated\n")

	res, err := Uninstall(cfg, true, false)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if len(res.Skipped) != 1 || len(res.Restored) != 0 {
		t.Errorf("res = %+v, want 1 skipped, 0 restored", res)
	}
	if got := mustReadFile(t, filepath.Join(home, ".gitconfig")); got != "unrelated\n" {
		t.Errorf("unrelated file was changed: %q", got)
	}
}

func TestUninstall_ReportsBackupsWithoutRestoring(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", ".gitconfig"), "repo\n")
	backup := filepath.Join(home, ".gitconfig.bak.20260101120000")
	writeFile(t, backup, "original\n")
	if _, err := Install(cfg, false); err != nil {
		t.Fatalf("setup Install: %v", err)
	}

	res, err := Uninstall(cfg, false, false)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if len(res.Backups) != 1 || res.Backups[0] != backup {
		t.Errorf("Backups = %v, want [%s]", res.Backups, backup)
	}
	if got := mustReadFile(t, filepath.Join(home, ".gitconfig")); got != "repo\n" {
		t.Errorf("content = %q, want the repo version", got)
	}
}

func TestUninstall_RestoresOldestBackup(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", ".gitconfig"), "repo\n")
	target := filepath.Join(home, ".gitconfig")
	oldest := target + ".bak.20250101000000"
	newer := target + ".bak.20260101000000"
	writeFile(t, oldest, "original\n")
	writeFile(t, newer, "later\n")
	writeFile(t, target+".bak.notatimestamp", "ignored\n")
	if _, err := Install(cfg, false); err != nil {
		t.Fatalf("setup Install: %v", err)
	}

	res, err := Uninstall(cfg, true, false)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if got := mustReadFile(t, target); got != "original\n" {
		t.Errorf("content = %q, want the oldest backup", got)
	}
	if lexists(t, oldest) {
		t.Error("oldest backup should have been moved into place")
	}
	if len(res.Backups) != 1 || res.Backups[0] != newer {
		t.Errorf("Backups = %v, want only [%s] left", res.Backups, newer)
	}
}
