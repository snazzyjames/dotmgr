package linker

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRelink_KeepsLocalVersion(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	target := filepath.Join(home, ".gitconfig")
	writeFile(t, target, "old\n")
	if _, err := Add(cfg, target, false); err != nil {
		t.Fatal(err)
	}
	// A program saves by replacing the link with a new file.
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	writeFile(t, target, "new\n")

	if _, err := Relink(cfg, target); err != nil {
		t.Fatalf("Relink: %v", err)
	}
	if !isSymlink(t, target) {
		t.Error("not relinked")
	}
	if got := mustReadFile(t, filepath.Join(cfg.Repo.Path, "files", ".gitconfig")); got != "new\n" {
		t.Errorf("repo content = %q, want this machine's version", got)
	}
}

func TestRelink_Errors(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(home, ".linked"), "x")
	if _, err := Add(cfg, "~/.linked", false); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", ".missing"), "x")
	writeFile(t, filepath.Join(home, ".untracked"), "x")

	tests := []struct{ input, want string }{
		{"~/.linked", "already linked"},
		{"~/.missing", "dotmgr install"},
		{"~/.untracked", "isn't tracked"},
	}
	for _, tt := range tests {
		_, err := Relink(cfg, tt.input)
		if err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("Relink(%s): err = %v, want %q", tt.input, err, tt.want)
		}
	}
}
