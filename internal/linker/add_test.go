package linker

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
)

func TestAdd_HappyPath(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)

	src := filepath.Join(home, ".gitconfig")
	writeFile(t, src, "[user]\nname = test\n")

	rel, err := Add(cfg, "~/.gitconfig", false)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if rel != ".gitconfig" {
		t.Errorf("rel = %q, want %q", rel, ".gitconfig")
	}

	dest := filepath.Join(cfg.Repo.Path, "files", ".gitconfig")
	if got := mustReadFile(t, dest); got != "[user]\nname = test\n" {
		t.Errorf("repo content = %q, want original content", got)
	}

	info, err := os.Lstat(src)
	if err != nil {
		t.Fatalf("stat original path: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("original path is no longer a symlink after Add")
	}
	got, err := resolveSymlinkTarget(src)
	if err != nil {
		t.Fatalf("resolveSymlinkTarget: %v", err)
	}
	if got != dest {
		t.Errorf("symlink target = %q, want %q", got, dest)
	}
}

func TestAdd_NestedPath(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)

	src := filepath.Join(home, ".config", "fish", "config.fish")
	writeFile(t, src, "set -x FOO bar\n")

	rel, err := Add(cfg, "~/.config/fish/config.fish", false)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	wantRel := filepath.Join(".config", "fish", "config.fish")
	if rel != wantRel {
		t.Errorf("rel = %q, want %q", rel, wantRel)
	}

	dest := filepath.Join(cfg.Repo.Path, "files", wantRel)
	if got := mustReadFile(t, dest); got != "set -x FOO bar\n" {
		t.Errorf("repo content = %q, want original content", got)
	}
}

func TestAdd_BareRelativeInput(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(home, ".gitconfig"), "content")

	rel, err := Add(cfg, ".gitconfig", false)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if rel != ".gitconfig" {
		t.Errorf("rel = %q, want %q", rel, ".gitconfig")
	}
}

func TestAdd_NoSuchFile(t *testing.T) {
	newTestHome(t)
	cfg := newTestConfig(t)

	_, err := Add(cfg, "~/does-not-exist", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "no such file") {
		t.Errorf("error = %q, want it to mention 'no such file'", err)
	}
}

func TestAdd_Directory(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	dir := filepath.Join(home, ".config", "nvim")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Add(cfg, "~/.config/nvim", false)
	if err == nil {
		t.Fatal("expected error adding a directory, got nil")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("error = %q, want it to mention 'directory'", err)
	}
}

func TestAdd_AlreadyTrackedSymlink(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(home, ".gitconfig"), "content")

	if _, err := Add(cfg, "~/.gitconfig", false); err != nil {
		t.Fatalf("first Add: %v", err)
	}

	_, err := Add(cfg, "~/.gitconfig", false)
	if err == nil {
		t.Fatal("expected error re-adding an already-tracked file, got nil")
	}
	if !strings.Contains(err.Error(), "already tracked") {
		t.Errorf("error = %q, want it to mention 'already tracked'", err)
	}
}

func TestAdd_ForeignSymlink(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)

	elsewhere := filepath.Join(t.TempDir(), "elsewhere.txt")
	writeFile(t, elsewhere, "content")
	link := filepath.Join(home, ".gitconfig")
	if err := os.Symlink(elsewhere, link); err != nil {
		t.Fatal(err)
	}

	_, err := Add(cfg, "~/.gitconfig", false)
	if err == nil {
		t.Fatal("expected error adding a symlink that points outside the repo, got nil")
	}
	if !strings.Contains(err.Error(), "symlink") {
		t.Errorf("error = %q, want it to mention 'symlink'", err)
	}
}

func TestAdd_DestAlreadyExists(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)

	// Pre-seed the repo's files/ tree without going through Add, then try
	// to Add a same-named file from $HOME.
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", ".gitconfig"), "repo copy")
	writeFile(t, filepath.Join(home, ".gitconfig"), "home copy")

	_, err := Add(cfg, "~/.gitconfig", false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "already tracked") {
		t.Errorf("error = %q, want it to mention 'already tracked'", err)
	}
}

func TestAdd_SecretsDenylist(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(home, ".ssh", "id_rsa"), "private key material")

	_, err := Add(cfg, "~/.ssh/id_rsa", false)
	if err == nil {
		t.Fatal("expected error adding a denylisted file without --force, got nil")
	}
	if !strings.Contains(err.Error(), "denylist") {
		t.Errorf("error = %q, want it to mention the denylist", err)
	}

	rel, err := Add(cfg, "~/.ssh/id_rsa", true)
	if err != nil {
		t.Fatalf("Add with force: %v", err)
	}
	if rel != filepath.Join(".ssh", "id_rsa") {
		t.Errorf("rel = %q, want %q", rel, filepath.Join(".ssh", "id_rsa"))
	}
}

func TestAdd_NotRegularFile(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)

	fifo := filepath.Join(home, ".fifo")
	if err := syscall.Mkfifo(fifo, 0o600); err != nil {
		t.Skipf("cannot create a FIFO on this system: %v", err)
	}

	_, err := Add(cfg, "~/.fifo", false)
	if err == nil {
		t.Fatal("expected error adding a non-regular file, got nil")
	}
	if !strings.Contains(err.Error(), "not a regular file") {
		t.Errorf("error = %q, want it to mention 'not a regular file'", err)
	}
}
