package linker

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestAddDir(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(cfg.Repo.Path, ".dotmgrignore"), "**/.DS_Store\n")
	kitty := filepath.Join(home, ".config", "kitty")
	writeFile(t, filepath.Join(kitty, "kitty.conf"), "font_size 14\n")
	writeFile(t, filepath.Join(kitty, "themes", "dark.conf"), "background #000\n")
	writeFile(t, filepath.Join(kitty, ".DS_Store"), "junk")
	if err := os.Symlink("/etc/hosts", filepath.Join(kitty, "unrelated-link")); err != nil {
		t.Fatal(err)
	}

	res, err := AddDir(cfg, "~/.config/kitty")
	if err != nil {
		t.Fatalf("AddDir: %v", err)
	}
	sort.Strings(res.Done)
	want := []string{".config/kitty/kitty.conf", ".config/kitty/themes/dark.conf"}
	if len(res.Done) != 2 || res.Done[0] != want[0] || res.Done[1] != want[1] {
		t.Errorf("Done = %v, want %v", res.Done, want)
	}
	if len(res.Ignored) != 1 {
		t.Errorf("Ignored = %v, want the .DS_Store", res.Ignored)
	}
	for _, rel := range want {
		if got := mustReadFile(t, filepath.Join(cfg.Repo.Path, "files", rel)); got == "" {
			t.Errorf("%s not moved into the repo", rel)
		}
		info, _ := os.Lstat(filepath.Join(home, rel))
		if info == nil || info.Mode()&os.ModeSymlink == 0 {
			t.Errorf("~/%s is not a symlink", rel)
		}
	}
	if lexists(t, filepath.Join(cfg.Repo.Path, "files", ".config", "kitty", "unrelated-link")) {
		t.Error("a symlink that isn't dotmgr's was moved into the repo")
	}

	// Re-running only picks up new files.
	writeFile(t, filepath.Join(kitty, "new.conf"), "x\n")
	res, err = AddDir(cfg, "~/.config/kitty")
	if err != nil {
		t.Fatalf("second AddDir: %v", err)
	}
	if len(res.Done) != 1 || res.Done[0] != ".config/kitty/new.conf" {
		t.Errorf("second run Done = %v, want only the new file", res.Done)
	}
	if len(res.Already) != 2 {
		t.Errorf("second run Already = %v, want 2", res.Already)
	}
}

func TestAddDir_SkipsDeniedFiles(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(home, ".ssh", "config"), "Host *\n")
	writeFile(t, filepath.Join(home, ".ssh", "id_ed25519"), "PRIVATE")

	res, err := AddDir(cfg, "~/.ssh")
	if err != nil {
		t.Fatalf("AddDir: %v", err)
	}
	if len(res.Denied) != 1 || res.Denied[0] != filepath.Join(".ssh", "id_ed25519") {
		t.Errorf("Denied = %v, want the private key", res.Denied)
	}
	if got := mustReadFile(t, filepath.Join(home, ".ssh", "id_ed25519")); got != "PRIVATE" {
		t.Error("private key was touched")
	}
	if len(res.Done) != 1 {
		t.Errorf("Done = %v, want only .ssh/config", res.Done)
	}
}

func TestAddDir_RejectsRepoAndOutsideHome(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	if _, err := AddDir(cfg, t.TempDir()); err == nil {
		t.Error("expected error for a directory outside $HOME")
	}
	cfg.Repo.Path = filepath.Join(home, ".local", "share", "dotmgr", "repo")
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", "x"), "x")
	if _, err := AddDir(cfg, cfg.Repo.Path); err == nil {
		t.Error("expected error for the repo itself")
	}
}

func TestRemoveDir(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	kitty := filepath.Join(home, ".config", "kitty")
	writeFile(t, filepath.Join(kitty, "kitty.conf"), "font_size 14\n")
	writeFile(t, filepath.Join(kitty, "themes", "dark.conf"), "background #000\n")
	if _, err := AddDir(cfg, kitty); err != nil {
		t.Fatalf("setup AddDir: %v", err)
	}
	writeFile(t, filepath.Join(kitty, "untracked.conf"), "x\n")

	res, err := RemoveDir(cfg, kitty)
	if err != nil {
		t.Fatalf("RemoveDir: %v", err)
	}
	if len(res.Done) != 2 {
		t.Errorf("Done = %v, want 2 files", res.Done)
	}
	for _, rel := range []string{"kitty.conf", "themes/dark.conf", "untracked.conf"} {
		info, err := os.Lstat(filepath.Join(kitty, rel))
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			t.Errorf("%s should be a regular file after RemoveDir", rel)
		}
	}
}
