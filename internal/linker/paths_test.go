package linker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveHomeInput(t *testing.T) {
	home := newTestHome(t)

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"tilde alone", "~", home},
		{"tilde slash prefix", "~/.gitconfig", filepath.Join(home, ".gitconfig")},
		{"tilde slash nested", "~/.config/fish/config.fish", filepath.Join(home, ".config/fish/config.fish")},
		{"bare relative", ".gitconfig", filepath.Join(home, ".gitconfig")},
		{"relative with dotdot cleaned", "~/a/../b", filepath.Join(home, "b")},
		{"absolute unchanged", "/etc/passwd", "/etc/passwd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveHomeInput(tt.input)
			if err != nil {
				t.Fatalf("resolveHomeInput(%q): %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("resolveHomeInput(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRelToHome(t *testing.T) {
	home := newTestHome(t)

	rel, err := relToHome(filepath.Join(home, ".gitconfig"))
	if err != nil {
		t.Fatalf("relToHome inside $HOME: %v", err)
	}
	if rel != ".gitconfig" {
		t.Errorf("rel = %q, want %q", rel, ".gitconfig")
	}

	rel, err = relToHome(filepath.Join(home, ".config", "fish", "config.fish"))
	if err != nil {
		t.Fatalf("relToHome nested: %v", err)
	}
	if rel != filepath.Join(".config", "fish", "config.fish") {
		t.Errorf("rel = %q, want %q", rel, filepath.Join(".config", "fish", "config.fish"))
	}

	if _, err := relToHome("/etc/passwd"); err == nil {
		t.Error("expected error for a path outside $HOME, got nil")
	}
}

func TestResolveSymlinkTarget(t *testing.T) {
	dir := t.TempDir()
	real := filepath.Join(dir, "real.txt")
	writeFile(t, real, "content\n")

	t.Run("relative target", func(t *testing.T) {
		link := filepath.Join(dir, "rel-link")
		if err := os.Symlink("real.txt", link); err != nil {
			t.Fatal(err)
		}
		got, err := resolveSymlinkTarget(link)
		if err != nil {
			t.Fatalf("resolveSymlinkTarget: %v", err)
		}
		if got != real {
			t.Errorf("resolved = %q, want %q", got, real)
		}
	})

	t.Run("absolute target", func(t *testing.T) {
		link := filepath.Join(dir, "abs-link")
		if err := os.Symlink(real, link); err != nil {
			t.Fatal(err)
		}
		got, err := resolveSymlinkTarget(link)
		if err != nil {
			t.Fatalf("resolveSymlinkTarget: %v", err)
		}
		if got != real {
			t.Errorf("resolved = %q, want %q", got, real)
		}
	})

	t.Run("not a symlink", func(t *testing.T) {
		if _, err := resolveSymlinkTarget(real); err == nil {
			t.Error("expected error reading a regular file as a symlink, got nil")
		}
	})
}

func TestUnder(t *testing.T) {
	tests := []struct {
		path, dir string
		want      bool
	}{
		{"/a/b", "/a/b", true},
		{"/a/b/c", "/a/b", true},
		{"/a/bc", "/a/b", false}, // shares a prefix but isn't nested
		{"/a/c", "/a/b", false},
		{"/a/b/../c", "/a/b", false}, // cleans to /a/c
	}
	for _, tt := range tests {
		got := under(tt.path, tt.dir)
		if got != tt.want {
			t.Errorf("under(%q, %q) = %v, want %v", tt.path, tt.dir, got, tt.want)
		}
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()

	real := filepath.Join(dir, "real.txt")
	writeFile(t, real, "x")
	if !exists(real) {
		t.Error("exists(real file) = false, want true")
	}

	broken := filepath.Join(dir, "broken-link")
	if err := os.Symlink(filepath.Join(dir, "nowhere"), broken); err != nil {
		t.Fatal(err)
	}
	if !exists(broken) {
		t.Error("exists(broken symlink) = false, want true (Lstat succeeds on the link itself)")
	}

	if exists(filepath.Join(dir, "missing")) {
		t.Error("exists(missing path) = true, want false")
	}
}
