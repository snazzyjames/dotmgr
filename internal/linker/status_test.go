package linker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStatus(t *testing.T) {
	home := newTestHome(t)
	cfg := newTestConfig(t)

	writeFile(t, filepath.Join(cfg.Repo.Path, "files", "ok.txt"), "a")
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", "missing.txt"), "b")
	writeFile(t, filepath.Join(cfg.Repo.Path, "files", "conflict.txt"), "c")

	// "ok.txt": correctly linked.
	if err := os.Symlink(
		filepath.Join(cfg.Repo.Path, "files", "ok.txt"),
		filepath.Join(home, "ok.txt"),
	); err != nil {
		t.Fatal(err)
	}

	// "missing.txt": nothing at the target, left alone.

	// "conflict.txt": a plain file sits where the symlink should be.
	writeFile(t, filepath.Join(home, "conflict.txt"), "unrelated content")

	got, err := Status(cfg)
	if err != nil {
		t.Fatalf("Status: %v", err)
	}

	states := map[string]string{}
	for _, tf := range got {
		states[tf.Rel] = tf.State
	}
	if len(states) != 3 {
		t.Fatalf("Status returned %d entries, want 3: %+v", len(states), got)
	}

	want := map[string]string{
		"ok.txt":       "ok",
		"missing.txt":  "missing",
		"conflict.txt": "conflict",
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
