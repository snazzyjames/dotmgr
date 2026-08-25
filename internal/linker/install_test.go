package linker

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/config"
)

func TestInstall_LinksTrackedFiles(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", ".gitconfig"), "a")
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", ".config", "fish", "config.fish"), "b")

	res, err := Install(cfg, false)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if res.Linked != 2 {
		t.Errorf("Linked = %d, want 2 (%+v)", res.Linked, res)
	}
	if res.Skipped != 0 || res.BackedUp != 0 || res.Failed != 0 {
		t.Errorf("unexpected non-zero counters: %+v", res)
	}

	link := filepath.Join(home, ".gitconfig")
	if !isCorrectSymlink(link, filepath.Join(cfg.Repo.Path, "files", ".gitconfig")) {
		t.Error(".gitconfig was not linked correctly")
	}
	nested := filepath.Join(home, ".config", "fish", "config.fish")
	if !isCorrectSymlink(nested, filepath.Join(cfg.Repo.Path, "files", ".config", "fish", "config.fish")) {
		t.Error("nested config.fish was not linked correctly")
	}
}

func TestInstall_SkipsAlreadyCorrectLinks(t *testing.T) {
	newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", ".gitconfig"), "a")

	if _, err := Install(cfg, false); err != nil {
		t.Fatalf("first Install: %v", err)
	}

	res, err := Install(cfg, false)
	if err != nil {
		t.Fatalf("second Install: %v", err)
	}
	if res.Skipped != 1 || res.Linked != 0 {
		t.Errorf("second Install = %+v, want Skipped=1 Linked=0", res)
	}
}

func TestInstall_BacksUpConflict(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", ".gitconfig"), "repo content")

	target := filepath.Join(home, ".gitconfig")
	writeFile(t, target, "pre-existing content")

	res, err := Install(cfg, false)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if res.BackedUp != 1 || res.Linked != 1 {
		t.Errorf("Install = %+v, want BackedUp=1 Linked=1", res)
	}

	if !isCorrectSymlink(target, filepath.Join(cfg.Repo.Path, "files", ".gitconfig")) {
		t.Error("target is not a correct symlink after backing up the conflict")
	}

	matches, _ := filepath.Glob(target + ".bak.*")
	if len(matches) != 1 {
		t.Fatalf("expected exactly one backup file, found %v", matches)
	}
	if got := mustReadFile(t, matches[0]); got != "pre-existing content" {
		t.Errorf("backup content = %q, want %q", got, "pre-existing content")
	}
}

func TestInstall_DryRunConflict(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", ".gitconfig"), "repo content")
	target := filepath.Join(home, ".gitconfig")
	writeFile(t, target, "pre-existing content")

	res, err := Install(cfg, true)
	if err != nil {
		t.Fatalf("Install dry-run: %v", err)
	}
	if res.BackedUp != 1 {
		t.Errorf("BackedUp = %d, want 1 (dry run should still report what it would do)", res.BackedUp)
	}

	if got := mustReadFile(t, target); got != "pre-existing content" {
		t.Errorf("dry run modified the target file: got %q", got)
	}
	matches, _ := filepath.Glob(target + ".bak.*")
	if len(matches) != 0 {
		t.Errorf("dry run created a backup file: %v", matches)
	}
}

func TestInstall_DryRunNoConflict(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", ".gitconfig"), "repo content")
	target := filepath.Join(home, ".gitconfig")

	res, err := Install(cfg, true)
	if err != nil {
		t.Fatalf("Install dry-run: %v", err)
	}
	if res.Linked != 1 {
		t.Errorf("Linked = %d, want 1", res.Linked)
	}
	if lexists(t, target) {
		t.Error("dry run created a symlink; it should only report the count")
	}
}

func TestInstall_FailureIsRecordedNotFatal(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	// A tracked file whose parent directory can't be created because a
	// regular file already occupies that path component.
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", "blocked", "file.txt"), "x")
	// A second, unrelated tracked file that should still install fine.
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", ".gitconfig"), "a")

	if err := os.WriteFile(filepath.Join(home, "blocked"), []byte("not a dir"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := Install(cfg, false)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if res.Failed != 1 {
		t.Errorf("Failed = %d, want 1 (%+v)", res.Failed, res)
	}
	if len(res.Errors) != 1 {
		t.Errorf("Errors = %v, want exactly one entry", res.Errors)
	}
	if res.Linked != 1 {
		t.Errorf("Linked = %d, want 1: a failure on one file should not stop the rest", res.Linked)
	}
}

func TestInstall_MissingFilesDir(t *testing.T) {
	newTestHome(t)
	cfg := &config.Config{}
	cfg.Repo.Path = filepath.Join(t.TempDir(), "never-initialized")

	if _, err := Install(cfg, false); err == nil {
		t.Fatal("expected error when the repo's files/ tree doesn't exist, got nil")
	}
}
