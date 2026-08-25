package discover

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/config"
)

func newTestHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func newTestConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg := &config.Config{}
	cfg.Repo.Path = t.TempDir()
	return cfg
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCandidates_FindsUntrackedKnownFiles(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(home, ".gitconfig"), "[user]\n")
	writeFile(t, filepath.Join(home, ".zshrc"), "export PATH=$PATH\n")

	got, err := Candidates(cfg)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	want := []string{".gitconfig", ".zshrc"} // sorted
	if len(got) != len(want) {
		t.Fatalf("Candidates() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCandidates_DirectoryEntryWalksNestedFiles(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(home, ".config", "fish", "config.fish"), "set -x FOO bar\n")
	writeFile(t, filepath.Join(home, ".config", "fish", "functions", "f.fish"), "function f\nend\n")

	got, err := Candidates(cfg)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	want := []string{
		filepath.Join(".config", "fish", "config.fish"),
		filepath.Join(".config", "fish", "functions", "f.fish"),
	}
	if len(got) != len(want) {
		t.Fatalf("Candidates() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestCandidates_SkipsAlreadyTrackedSymlink(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	filesDir := filepath.Join(cfg.Repo.Path, "files")
	writeFile(t, filepath.Join(filesDir, ".gitconfig"), "[user]\n")

	if err := os.Symlink(filepath.Join(filesDir, ".gitconfig"), filepath.Join(home, ".gitconfig")); err != nil {
		t.Fatal(err)
	}

	got, err := Candidates(cfg)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Candidates() = %v, want empty (file is already tracked)", got)
	}
}

func TestCandidates_SkipsWhenRepoCopyAlreadyExists(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	// A plain (non-symlink) file at the candidate path, but the repo
	// already has a same-named file under files/ (tracked without a
	// symlink back yet, e.g. mid-migration).
	writeFile(t, filepath.Join(home, ".gitconfig"), "[user]\n")
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", ".gitconfig"), "[user]\n")

	got, err := Candidates(cfg)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Candidates() = %v, want empty", got)
	}
}

func TestCandidates_SkipsDotmgrignorePatterns(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	writeFile(t, filepath.Join(home, ".gitconfig"), "[user]\n")
	writeFile(t, filepath.Join(cfg.Repo.Path, ".dotmgrignore"), ".gitconfig\n")

	got, err := Candidates(cfg)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Candidates() = %v, want empty (excluded by .dotmgrignore)", got)
	}
}

func TestCandidates_SkipsSecretsDenylist(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)
	// Lives under the "neovim" registry entry's directory but matches the
	// built-in secrets denylist ("*_rsa").
	writeFile(t, filepath.Join(home, ".config", "nvim", "deploy_rsa"), "private key material")

	got, err := Candidates(cfg)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Candidates() = %v, want empty (denylisted)", got)
	}
}

func TestCandidates_IgnoresNonexistentRegistryRoots(t *testing.T) {
	newTestHome(t) // empty $HOME: none of the registry's paths exist
	cfg := newTestConfig(t)

	got, err := Candidates(cfg)
	if err != nil {
		t.Fatalf("Candidates: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Candidates() = %v, want empty", got)
	}
}

func TestAlreadyTracked(t *testing.T) {
	dir := t.TempDir()
	filesDir := filepath.Join(dir, "files")
	writeFile(t, filepath.Join(filesDir, "f.txt"), "x")

	tracked := filepath.Join(dir, "tracked-link")
	if err := os.Symlink(filepath.Join(filesDir, "f.txt"), tracked); err != nil {
		t.Fatal(err)
	}
	if !alreadyTracked(tracked, filesDir) {
		t.Error("alreadyTracked(symlink into filesDir) = false, want true")
	}

	elsewhere := filepath.Join(dir, "elsewhere.txt")
	writeFile(t, elsewhere, "x")
	foreignLink := filepath.Join(dir, "foreign-link")
	if err := os.Symlink(elsewhere, foreignLink); err != nil {
		t.Fatal(err)
	}
	if alreadyTracked(foreignLink, filesDir) {
		t.Error("alreadyTracked(symlink outside filesDir) = true, want false")
	}

	plain := filepath.Join(dir, "plain.txt")
	writeFile(t, plain, "x")
	if alreadyTracked(plain, filesDir) {
		t.Error("alreadyTracked(regular file) = true, want false")
	}

	if alreadyTracked(filepath.Join(dir, "missing"), filesDir) {
		t.Error("alreadyTracked(missing path) = true, want false")
	}
}

func TestExpandHome(t *testing.T) {
	home := "/home/test"
	tests := []struct {
		in, want string
	}{
		{"~", home},
		{"~/.gitconfig", filepath.Join(home, ".gitconfig")},
		{"/etc/passwd", "/etc/passwd"},
	}
	for _, tt := range tests {
		if got := expandHome(tt.in, home); got != tt.want {
			t.Errorf("expandHome(%q, %q) = %q, want %q", tt.in, home, got, tt.want)
		}
	}
}
