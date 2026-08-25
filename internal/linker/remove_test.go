package linker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRemove_HappyPath(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	src := filepath.Join(home, ".gitconfig")
	writeFile(t, src, "[user]\nname = test\n")

	if _, err := Add(cfg, "~/.gitconfig", false); err != nil {
		t.Fatalf("Add: %v", err)
	}

	rel, err := Remove(cfg, "~/.gitconfig")
	if err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if rel != ".gitconfig" {
		t.Errorf("rel = %q, want %q", rel, ".gitconfig")
	}

	info, err := os.Lstat(src)
	if err != nil {
		t.Fatalf("stat restored file: %v", err)
	}
	if info.Mode()&os.ModeSymlink != 0 {
		t.Error("path is still a symlink after Remove")
	}
	if got := mustReadFile(t, src); got != "[user]\nname = test\n" {
		t.Errorf("restored content = %q, want original content", got)
	}
	if lexists(t, filepath.Join(cfg.Repo.Path, "files", ".gitconfig")) {
		t.Error("file still present in repo's files/ tree after Remove")
	}
}

func TestRemove_NoSuchFile(t *testing.T) {
	newTestHome(t)
	cfg := newTestConfig(t)

	_, err := Remove(cfg, "~/does-not-exist")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no such file") {
		t.Errorf("error = %q, want it to mention 'no such file'", err)
	}
}

func TestRemove_NotASymlink(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(home, ".gitconfig"), "content")

	_, err := Remove(cfg, "~/.gitconfig")
	if err == nil {
		t.Fatal("expected error removing a regular file, got nil")
	}
	if !strings.Contains(err.Error(), "not a symlink") {
		t.Errorf("error = %q, want it to mention 'not a symlink'", err)
	}
}

func TestRemove_SymlinkNotManagedByDotmgr(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)

	elsewhere := filepath.Join(t.TempDir(), "elsewhere.txt")
	writeFile(t, elsewhere, "content")
	link := filepath.Join(home, ".gitconfig")
	if err := os.Symlink(elsewhere, link); err != nil {
		t.Fatal(err)
	}

	_, err := Remove(cfg, "~/.gitconfig")
	if err == nil {
		t.Fatal("expected error removing a symlink outside the repo, got nil")
	}
	if !strings.Contains(err.Error(), "does not point into the dotfiles repo") {
		t.Errorf("error = %q, want it to mention the repo", err)
	}
}

func TestRemove_BrokenTrackedSymlink(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)

	// A symlink that points into files/ but whose target was never
	// actually created there (e.g. the repo file was deleted by hand).
	dest := filepath.Join(cfg.Repo.Path, "files", ".gitconfig")
	link := filepath.Join(home, ".gitconfig")
	if err := os.Symlink(dest, link); err != nil {
		t.Fatal(err)
	}

	_, err := Remove(cfg, "~/.gitconfig")
	if err == nil {
		t.Fatal("expected error when the repo-side file is missing, got nil")
	}
	if !strings.Contains(err.Error(), "moving") {
		t.Errorf("error = %q, want it to mention the failed move", err)
	}
}
