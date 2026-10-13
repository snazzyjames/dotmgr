package linker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStatus(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)

	for _, name := range []string{"ok.txt", "missing.txt", "unlinked.txt", "diverged.txt", "conflict.txt"} {
		writeFile(t, filepath.Join(cfg.Repo.Path, "files", name), "repo content")
	}

	// "ok.txt": correctly linked.
	if err := os.Symlink(
		filepath.Join(cfg.Repo.Path, "files", "ok.txt"),
		filepath.Join(home, "ok.txt"),
	); err != nil {
		t.Fatal(err)
	}

	// "missing.txt": nothing at the target, left alone.

	// "unlinked.txt": a plain file with the same content, e.g. a program
	// rewrote the file without changing it.
	writeFile(t, filepath.Join(home, "unlinked.txt"), "repo content")

	// "diverged.txt": a plain file with different content, e.g. a program
	// saved over the link.
	writeFile(t, filepath.Join(home, "diverged.txt"), "edited on this machine")

	// "conflict.txt": something that isn't a file at all.
	if err := os.MkdirAll(filepath.Join(home, "conflict.txt"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := Status(cfg)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	states := map[string]string{}
	for _, tf := range got {
		states[tf.Rel] = tf.State
	}
	if len(states) != 5 {
		t.Fatalf("Status returned %d entries, want 5: %+v", len(states), got)
	}

	want := map[string]string{
		"ok.txt":       StateOK,
		"missing.txt":  StateMissing,
		"unlinked.txt": StateUnlinked,
		"diverged.txt": StateDiverged,
		"conflict.txt": StateConflict,
	}
	for rel, wantState := range want {
		if states[rel] != wantState {
			t.Errorf("state[%q] = %q, want %q", rel, states[rel], wantState)
		}
	}
}

func TestStatus_NoTrackedFiles(t *testing.T) {
	newTestHome(t)
	cfg := newTestConfig(t)
	if err := os.MkdirAll(filepath.Join(cfg.Repo.Path, "files"), 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := Status(cfg)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("Status = %+v, want empty", got)
	}
}
