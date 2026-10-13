package selfupdate

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
)

func writeExe(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func TestReplace(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "dotmgr")
	writeExe(t, exe, "old")
	newBin := filepath.Join(t.TempDir(), "dotmgr")
	if err := os.WriteFile(newBin, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := Replace(newBin, exe); err != nil {
		t.Fatalf("Replace: %v", err)
	}
	got, _ := os.ReadFile(exe)
	if string(got) != "new" {
		t.Errorf("content = %q, want new", got)
	}
	info, _ := os.Stat(exe)
	if info.Mode().Perm() != 0o755 {
		t.Errorf("mode = %v, want 0755", info.Mode().Perm())
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) != 1 {
		t.Errorf("leftover temp files: %v", entries)
	}
}

func TestReplace_NoPermission(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write anywhere")
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "dotmgr")
	writeExe(t, exe, "old")
	newBin := filepath.Join(t.TempDir(), "dotmgr")
	writeExe(t, newBin, "new")
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	err := Replace(newBin, exe)
	if !errors.Is(err, ErrPermission) {
		t.Fatalf("err = %v, want ErrPermission", err)
	}
	if got, _ := os.ReadFile(exe); string(got) != "old" {
		t.Errorf("exe changed despite the error: %q", got)
	}
}

func TestBuild_NoGo(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	_, err := Build(Source{Version: "latest"}, t.TempDir(), io.Discard)
	if !errors.Is(err, ErrNoGo) {
		t.Errorf("err = %v, want ErrNoGo", err)
	}
}

func TestBuild_FromRejectsOtherModules(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	other := t.TempDir()
	if err := os.WriteFile(filepath.Join(other, "go.mod"), []byte("module example.com/other\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, dir := range []string{other, t.TempDir()} {
		_, err := Build(Source{Dir: dir}, t.TempDir(), io.Discard)
		if err == nil || !strings.Contains(err.Error(), "isn't a dotmgr checkout") {
			t.Errorf("Build(%s): err = %v, want a not-a-checkout error", dir, err)
		}
	}
}

// TestBuild_FromCheckout compiles this repo for real.
func TestBuild_FromCheckout(t *testing.T) {
	if testing.Short() {
		t.Skip("compiles dotmgr")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go not on PATH")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	bin, err := Build(Source{Dir: root}, t.TempDir(), io.Discard)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	out, err := exec.Command(bin, "version").Output()
	if err != nil {
		t.Fatalf("running built binary: %v", err)
	}
	if !strings.HasPrefix(string(out), "dotmgr ") {
		t.Errorf("version output = %q", out)
	}
}

func TestDescribe(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		settings map[string]string
		want     string
	}{
		{"go install @version", "v1.2.0", nil, "v1.2.0"},
		{"checkout build", "(devel)", map[string]string{"vcs.revision": "abcdef0123456789"}, "dev (abcdef012345)"},
		{"dirty checkout", "(devel)", map[string]string{"vcs.revision": "abc", "vcs.modified": "true"}, "dev (abc-dirty)"},
		{"no vcs info", "(devel)", nil, "dev"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &debug.BuildInfo{Main: debug.Module{Version: tt.version}}
			for k, v := range tt.settings {
				info.Settings = append(info.Settings, debug.BuildSetting{Key: k, Value: v})
			}
			if got := describe(info); got != tt.want {
				t.Errorf("describe = %q, want %q", got, tt.want)
			}
		})
	}
}
