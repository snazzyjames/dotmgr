package linker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupKitty(t *testing.T, home string) string {
	t.Helper()
	kitty := filepath.Join(home, ".config", "kitty")
	writeFile(t, filepath.Join(kitty, "kitty.conf"), "font_size 14\n")
	writeFile(t, filepath.Join(kitty, "themes", "dark.conf"), "background #000\n")
	return kitty
}

func isSymlink(t *testing.T, path string) bool {
	t.Helper()
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func TestLinkDir(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	kitty := setupKitty(t, home)

	res, err := LinkDir(cfg, kitty)
	if err != nil {
		t.Fatalf("LinkDir: %v", err)
	}
	if res.Rel != filepath.Join(".config", "kitty") {
		t.Errorf("Rel = %q", res.Rel)
	}
	if !isSymlink(t, kitty) {
		t.Fatal("~/.config/kitty is not a symlink")
	}
	if got := mustReadFile(t, filepath.Join(kitty, "themes", "dark.conf")); got != "background #000\n" {
		t.Errorf("content through the link = %q", got)
	}
	dirs, _ := LinkedDirs(cfg)
	if len(dirs) != 1 || dirs[0] != filepath.Join(".config", "kitty") {
		t.Errorf("LinkedDirs = %v", dirs)
	}

	// New files land in the repo automatically.
	writeFile(t, filepath.Join(kitty, "new.conf"), "x\n")
	if !lexists(t, filepath.Join(cfg.Repo.Path, "files", ".config", "kitty", "new.conf")) {
		t.Error("a file created after linking didn't land in the repo")
	}

	// Files inside are already tracked; add, AddDir and Relink say so.
	if _, err := Add(cfg, filepath.Join(kitty, "new.conf"), false); err == nil || !strings.Contains(err.Error(), "linked as a whole") {
		t.Errorf("Add inside linked dir: err = %v", err)
	}
	if _, err := AddDir(cfg, filepath.Join(kitty, "themes")); err == nil || !strings.Contains(err.Error(), "linked as a whole") {
		t.Errorf("AddDir inside linked dir: err = %v", err)
	}
	if _, err := LinkDir(cfg, kitty); err == nil {
		t.Error("linking an already linked dir should fail")
	}
}

func TestLinkDir_ConvertsPerFileLinks(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	kitty := setupKitty(t, home)
	if _, err := AddDir(cfg, kitty); err != nil {
		t.Fatalf("setup AddDir: %v", err)
	}

	res, err := LinkDir(cfg, kitty)
	if err != nil {
		t.Fatalf("LinkDir: %v", err)
	}
	if len(res.Converted) != 2 {
		t.Errorf("Converted = %v, want 2", res.Converted)
	}
	if !isSymlink(t, kitty) {
		t.Fatal("dir is not a symlink")
	}
	if isSymlink(t, filepath.Join(kitty, "kitty.conf")) {
		t.Error("kitty.conf inside the repo is still a per-file symlink")
	}
	if got := mustReadFile(t, filepath.Join(kitty, "kitty.conf")); got != "font_size 14\n" {
		t.Errorf("content = %q", got)
	}
}

func TestLinkDir_Refusals(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(home, ".config", "x", "a"), "a")
	writeFile(t, filepath.Join(home, ".config", "gh", "config.yml"), "x")
	writeFile(t, filepath.Join(home, ".config", "gh", "id_ed25519"), "PRIVATE")
	writeFile(t, filepath.Join(home, "file"), "x")

	tests := []struct {
		name, input, want string
	}{
		{"broad dir", "~/.config", "too broad"},
		{"home", "~", "too broad"},
		{"secret inside", "~/.config/gh", "secrets denylist"},
		{"a file", "~/file", "not a directory"},
		{"missing", "~/nope", "no such directory"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := LinkDir(cfg, tt.input)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %v, want it to mention %q", err, tt.want)
			}
		})
	}
	if isSymlink(t, filepath.Join(home, ".config", "gh")) {
		t.Error("refused dir was linked anyway")
	}

	// Overlap with an already linked directory.
	if _, err := LinkDir(cfg, "~/.config/x"); err != nil {
		t.Fatalf("LinkDir: %v", err)
	}
	writeFile(t, filepath.Join(home, ".config", "x2", "a"), "a")
	if _, err := LinkDir(cfg, "~/.config/x2"); err != nil {
		t.Errorf("a sibling with a shared prefix should be allowed: %v", err)
	}
}

func TestLinkDir_InstallStatusRemove(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	kitty := setupKitty(t, home)
	if _, err := LinkDir(cfg, kitty); err != nil {
		t.Fatalf("LinkDir: %v", err)
	}

	// Simulate a new machine: the app already made its own default dir.
	if err := os.Remove(kitty); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kitty, "kitty.conf"), "defaults\n")

	st, err := Status(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if len(st) != 1 || !st[0].Dir || st[0].State != StateConflict {
		t.Errorf("Status = %+v, want one dir in conflict", st)
	}

	res, err := Install(cfg, false)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if res.Linked != 1 || res.BackedUp != 1 || res.Failed != 0 {
		t.Errorf("Install = %+v, want 1 linked (the dir, not its files), 1 backed up", res)
	}
	if !isSymlink(t, kitty) {
		t.Fatal("dir not linked by install")
	}
	backups, _ := findBackups(kitty)
	if len(backups) != 1 || mustReadFile(t, filepath.Join(backups[0], "kitty.conf")) != "defaults\n" {
		t.Errorf("backups = %v, want the app's default dir", backups)
	}

	st, _ = Status(cfg)
	if len(st) != 1 || st[0].State != StateOK {
		t.Errorf("Status after install = %+v", st)
	}

	// Remove puts the real directory back and forgets it.
	if _, err := Remove(cfg, "~/.config/kitty"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if isSymlink(t, kitty) || mustReadFile(t, filepath.Join(kitty, "kitty.conf")) != "font_size 14\n" {
		t.Error("Remove didn't restore the real directory")
	}
	if dirs, _ := LinkedDirs(cfg); len(dirs) != 0 {
		t.Errorf("LinkedDirs after Remove = %v", dirs)
	}
	if lexists(t, filepath.Join(cfg.Repo.Path, ".gitignore")) {
		t.Error(".gitignore should be removed once nothing is linked")
	}
}

func TestLinkDir_Uninstall(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	kitty := setupKitty(t, home)
	if err := os.Symlink("kitty.conf", filepath.Join(kitty, "alias.conf")); err != nil {
		t.Fatal(err)
	}
	if _, err := LinkDir(cfg, kitty); err != nil {
		t.Fatalf("LinkDir: %v", err)
	}

	res, err := Uninstall(cfg, false, false)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if len(res.Restored) != 1 || res.Failed != 0 {
		t.Errorf("Uninstall = %+v, want the dir restored once", res)
	}
	if isSymlink(t, kitty) {
		t.Fatal("dir is still a symlink")
	}
	if got := mustReadFile(t, filepath.Join(kitty, "themes", "dark.conf")); got != "background #000\n" {
		t.Errorf("content = %q", got)
	}
	if !isSymlink(t, filepath.Join(kitty, "alias.conf")) {
		t.Error("a symlink inside the dir wasn't kept as a symlink")
	}
}

func TestLinkDir_UninstallRestoresBackupDir(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	kitty := setupKitty(t, home)
	if _, err := LinkDir(cfg, kitty); err != nil {
		t.Fatalf("LinkDir: %v", err)
	}
	writeFile(t, filepath.Join(kitty+".bak.20250101000000", "kitty.conf"), "original\n")

	if _, err := Uninstall(cfg, true, false); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if got := mustReadFile(t, filepath.Join(kitty, "kitty.conf")); got != "original\n" {
		t.Errorf("content = %q, want the backup", got)
	}
}

func TestSyncGitignore(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(cfg.Repo.Path, ".dotmgrignore"), "**/.DS_Store\n.config/kitty/local.conf\n.other/x\n")
	writeFile(t, filepath.Join(cfg.Repo.Path, ".gitignore"), "my-own-rule\n")
	kitty := setupKitty(t, home)
	if _, err := LinkDir(cfg, kitty); err != nil {
		t.Fatalf("LinkDir: %v", err)
	}

	got := mustReadFile(t, filepath.Join(cfg.Repo.Path, ".gitignore"))
	for _, want := range []string{
		"my-own-rule\n",
		"/files/.config/kitty/**/.DS_Store\n",
		"/files/.config/kitty/local.conf\n",
		"/files/.config/kitty/**/*_ed25519\n",
	} {
		if !strings.Contains(got, want) {
			t.Errorf(".gitignore missing %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, ".other/x") {
		t.Errorf(".gitignore has a pattern for a path outside linked dirs:\n%s", got)
	}

	// Re-syncing is stable and keeps the user's own lines.
	if err := SyncGitignore(cfg); err != nil {
		t.Fatal(err)
	}
	if again := mustReadFile(t, filepath.Join(cfg.Repo.Path, ".gitignore")); again != got {
		t.Errorf("second sync changed .gitignore:\n%s\nvs\n%s", got, again)
	}
	if _, err := Remove(cfg, kitty); err != nil {
		t.Fatal(err)
	}
	if got := mustReadFile(t, filepath.Join(cfg.Repo.Path, ".gitignore")); got != "my-own-rule\n" {
		t.Errorf(".gitignore after unlinking = %q, want only the user's line", got)
	}
}
