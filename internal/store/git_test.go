package store

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGit runs git directly against dir for test setup/assertions, bypassing
// the package under test.
func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// configureIdentity sets a local commit identity so commits work regardless
// of the host's global git config.
func configureIdentity(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "config", "user.email", "dotmgr-test@example.com")
	runGit(t, dir, "config", "user.name", "dotmgr test")
	runGit(t, dir, "config", "commit.gpgsign", "false")
}

// newBareRemote creates an empty bare repo to use as a local "remote".
func newBareRemote(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "remote.git")
	if out, err := exec.Command("git", "init", "-q", "--bare", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %v\n%s", err, out)
	}
	return dir
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

func TestClone(t *testing.T) {
	remote := newBareRemote(t)

	// Seed the remote with a commit via a throwaway clone.
	seed := filepath.Join(t.TempDir(), "seed")
	if err := Clone(remote, seed); err != nil {
		t.Fatalf("seed clone: %v", err)
	}
	configureIdentity(t, seed)
	writeFile(t, filepath.Join(seed, "f.txt"), "hello\n")
	if err := CommitAll(seed, "seed"); err != nil {
		t.Fatalf("seed commit: %v", err)
	}
	if err := Push(seed); err != nil {
		t.Fatalf("seed push: %v", err)
	}

	dest := filepath.Join(t.TempDir(), "dest")
	if err := Clone(remote, dest); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dest, "f.txt"))
	if err != nil {
		t.Fatalf("reading cloned file: %v", err)
	}
	if string(got) != "hello\n" {
		t.Errorf("cloned file content = %q, want %q", got, "hello\n")
	}
}

func TestClone_EmptyRemoteSucceeds(t *testing.T) {
	remote := newBareRemote(t)
	dest := filepath.Join(t.TempDir(), "dest")
	if err := Clone(remote, dest); err != nil {
		t.Fatalf("Clone of empty remote should succeed, got: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dest, ".git")); err != nil {
		t.Errorf("expected %s/.git to exist: %v", dest, err)
	}
}

func TestClone_InvalidRemote(t *testing.T) {
	dest := filepath.Join(t.TempDir(), "dest")
	err := Clone(filepath.Join(t.TempDir(), "does-not-exist"), dest)
	if err == nil {
		t.Fatal("expected error cloning a nonexistent remote, got nil")
	}
	if !strings.Contains(err.Error(), "git clone") {
		t.Errorf("error = %q, want it to mention the failing git command", err)
	}
}

func TestDirty(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", dir)
	configureIdentity(t, dir)

	dirty, err := Dirty(dir)
	if err != nil {
		t.Fatalf("Dirty on fresh repo: %v", err)
	}
	if dirty {
		t.Error("fresh repo should not be dirty")
	}

	writeFile(t, filepath.Join(dir, "f.txt"), "content\n")
	dirty, err = Dirty(dir)
	if err != nil {
		t.Fatalf("Dirty with untracked file: %v", err)
	}
	if !dirty {
		t.Error("repo with an untracked file should be dirty")
	}

	if err := CommitAll(dir, "add f.txt"); err != nil {
		t.Fatalf("CommitAll: %v", err)
	}
	dirty, err = Dirty(dir)
	if err != nil {
		t.Fatalf("Dirty after commit: %v", err)
	}
	if dirty {
		t.Error("repo should be clean immediately after committing everything")
	}
}

func TestDirty_InvalidRepo(t *testing.T) {
	_, err := Dirty(t.TempDir())
	if err == nil {
		t.Fatal("expected error running status against a non-git directory, got nil")
	}
}

func TestCommitAll_CreatesCommit(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", dir)
	configureIdentity(t, dir)
	writeFile(t, filepath.Join(dir, "f.txt"), "content\n")

	if err := CommitAll(dir, "my message"); err != nil {
		t.Fatalf("CommitAll: %v", err)
	}

	count := strings.TrimSpace(runGit(t, dir, "rev-list", "--count", "HEAD"))
	if count != "1" {
		t.Errorf("commit count = %s, want 1", count)
	}
	subject := strings.TrimSpace(runGit(t, dir, "log", "-1", "--format=%s"))
	if subject != "my message" {
		t.Errorf("commit subject = %q, want %q", subject, "my message")
	}
}

func TestCommitAll_CleanTreeIsNoop(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", dir)
	configureIdentity(t, dir)
	writeFile(t, filepath.Join(dir, "f.txt"), "content\n")
	if err := CommitAll(dir, "first"); err != nil {
		t.Fatalf("first CommitAll: %v", err)
	}

	if err := CommitAll(dir, "second"); err != nil {
		t.Fatalf("CommitAll on clean tree should not error, got: %v", err)
	}

	count := strings.TrimSpace(runGit(t, dir, "rev-list", "--count", "HEAD"))
	if count != "1" {
		t.Errorf("commit count = %s, want 1 (no-op should not add a commit)", count)
	}
}

func TestPush(t *testing.T) {
	remote := newBareRemote(t)
	local := filepath.Join(t.TempDir(), "local")
	if err := Clone(remote, local); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	configureIdentity(t, local)
	writeFile(t, filepath.Join(local, "f.txt"), "pushed\n")

	if err := Push(local); err != nil {
		t.Fatalf("Push: %v", err)
	}

	// Verify the commit landed on the remote by cloning it fresh.
	check := filepath.Join(t.TempDir(), "check")
	runGit(t, filepath.Dir(check), "clone", "-q", remote, check)
	got, err := os.ReadFile(filepath.Join(check, "f.txt"))
	if err != nil {
		t.Fatalf("reading pushed file: %v", err)
	}
	if string(got) != "pushed\n" {
		t.Errorf("pushed file content = %q, want %q", got, "pushed\n")
	}
}

func TestPush_CommitsPendingChangesFirst(t *testing.T) {
	remote := newBareRemote(t)
	local := filepath.Join(t.TempDir(), "local")
	if err := Clone(remote, local); err != nil {
		t.Fatalf("Clone: %v", err)
	}
	configureIdentity(t, local)
	writeFile(t, filepath.Join(local, "f.txt"), "uncommitted\n")

	dirty, err := Dirty(local)
	if err != nil || !dirty {
		t.Fatalf("expected local repo to be dirty before Push, dirty=%v err=%v", dirty, err)
	}

	if err := Push(local); err != nil {
		t.Fatalf("Push: %v", err)
	}

	dirty, err = Dirty(local)
	if err != nil {
		t.Fatalf("Dirty after Push: %v", err)
	}
	if dirty {
		t.Error("Push should have committed pending changes, leaving the tree clean")
	}
}

func TestPush_NoRemote(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", dir)
	configureIdentity(t, dir)
	writeFile(t, filepath.Join(dir, "f.txt"), "content\n")

	err := Push(dir)
	if err == nil {
		t.Fatal("expected error pushing a repo with no remote configured, got nil")
	}
}

func TestPull(t *testing.T) {
	remote := newBareRemote(t)

	writer := filepath.Join(t.TempDir(), "writer")
	if err := Clone(remote, writer); err != nil {
		t.Fatalf("Clone writer: %v", err)
	}
	configureIdentity(t, writer)
	writeFile(t, filepath.Join(writer, "f.txt"), "v1\n")
	if err := Push(writer); err != nil {
		t.Fatalf("Push from writer: %v", err)
	}

	reader := filepath.Join(t.TempDir(), "reader")
	if err := Clone(remote, reader); err != nil {
		t.Fatalf("Clone reader: %v", err)
	}

	// Publish a second change from writer, then make sure Pull picks it up.
	writeFile(t, filepath.Join(writer, "f.txt"), "v2\n")
	if err := Push(writer); err != nil {
		t.Fatalf("second Push from writer: %v", err)
	}

	if err := Pull(reader); err != nil {
		t.Fatalf("Pull: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(reader, "f.txt"))
	if err != nil {
		t.Fatalf("reading pulled file: %v", err)
	}
	if string(got) != "v2\n" {
		t.Errorf("pulled file content = %q, want %q", got, "v2\n")
	}
}

func TestPull_NoRemote(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init", "-q", dir)

	err := Pull(dir)
	if err == nil {
		t.Fatal("expected error pulling in a repo with no remote configured, got nil")
	}
}

func TestUnpushed(t *testing.T) {
	remote := newBareRemote(t)
	repo := filepath.Join(t.TempDir(), "repo")
	if err := Clone(remote, repo); err != nil {
		t.Fatal(err)
	}
	configureIdentity(t, repo)

	if got, err := Unpushed(repo); err != nil || got {
		t.Errorf("empty clone: Unpushed = %v, %v; want false, nil", got, err)
	}

	writeFile(t, filepath.Join(repo, "f.txt"), "hello\n")
	if err := CommitAll(repo, "first"); err != nil {
		t.Fatal(err)
	}
	if got, err := Unpushed(repo); err != nil || !got {
		t.Errorf("after commit: Unpushed = %v, %v; want true, nil", got, err)
	}

	if err := Push(repo); err != nil {
		t.Fatal(err)
	}
	if got, err := Unpushed(repo); err != nil || got {
		t.Errorf("after push: Unpushed = %v, %v; want false, nil", got, err)
	}
}

func TestUnpushed_NoUpstream(t *testing.T) {
	repo := t.TempDir()
	runGit(t, repo, "init", "-q")
	configureIdentity(t, repo)
	writeFile(t, filepath.Join(repo, "f.txt"), "hello\n")
	if err := CommitAll(repo, "first"); err != nil {
		t.Fatal(err)
	}
	if got, err := Unpushed(repo); err != nil || !got {
		t.Errorf("Unpushed = %v, %v; want true, nil for a never-pushed repo", got, err)
	}
}
