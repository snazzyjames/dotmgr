package cli

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/snazzyjames/dotmgr/internal/selfupdate"
)

// fakeUpdate points the update command at a stand-in installed binary and
// a stand-in build that produces a script reporting newVersion. It returns
// the installed path and a pointer to the Source the build was asked for.
func fakeUpdate(t *testing.T, buildErr error) (exe string, gotSrc *selfupdate.Source) {
	t.Helper()
	exe = filepath.Join(t.TempDir(), "dotmgr")
	if err := os.WriteFile(exe, []byte("#!/bin/sh\necho 'dotmgr old'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	gotSrc = &selfupdate.Source{}

	oldBuild, oldInstalled := buildBinary, installedPath
	t.Cleanup(func() { buildBinary, installedPath = oldBuild, oldInstalled })
	installedPath = func() (string, error) { return exe, nil }
	buildBinary = func(src selfupdate.Source, dir string, _ io.Writer) (string, error) {
		*gotSrc = src
		if buildErr != nil {
			return "", buildErr
		}
		out := filepath.Join(dir, "dotmgr")
		return out, os.WriteFile(out, []byte("#!/bin/sh\necho 'dotmgr v9.9.9'\n"), 0o755)
	}
	return exe, gotSrc
}

func TestCmdUpdate_Latest(t *testing.T) {
	exe, src := fakeUpdate(t, nil)

	stdout, stderr, code := captureOutput(t, func() int { return cmdUpdate(nil) })
	if code != 0 {
		t.Fatalf("code = %d\nstdout: %s\nstderr: %s", code, stdout, stderr)
	}
	if src.Version != "latest" || src.Dir != "" {
		t.Errorf("built %+v, want latest", *src)
	}
	if got := mustReadFile(t, exe); !strings.Contains(got, "v9.9.9") {
		t.Errorf("installed binary wasn't replaced: %q", got)
	}
	if !strings.Contains(stdout, "now: v9.9.9") {
		t.Errorf("stdout = %q, want the new version", stdout)
	}
}

func TestCmdUpdate_VersionAndFrom(t *testing.T) {
	_, src := fakeUpdate(t, nil)
	if code := runQuiet(t, func() int { return cmdUpdate([]string{"--version", "v1.2.0"}) }); code != 0 {
		t.Fatalf("code = %d", code)
	}
	if src.Version != "v1.2.0" {
		t.Errorf("built %+v, want v1.2.0", *src)
	}

	checkout := t.TempDir()
	if code := runQuiet(t, func() int { return cmdUpdate([]string{"--from", checkout}) }); code != 0 {
		t.Fatalf("code = %d", code)
	}
	if src.Dir != checkout {
		t.Errorf("built %+v, want --from %s", *src, checkout)
	}

	_, stderr, code := captureOutput(t, func() int { return cmdUpdate([]string{"--version", "v1", "--from", checkout}) })
	if code != 1 || !strings.Contains(stderr, "not both") {
		t.Errorf("code = %d, stderr = %q, want a conflict error", code, stderr)
	}
}

func TestCmdUpdate_BuildFailureLeavesBinary(t *testing.T) {
	exe, _ := fakeUpdate(t, errors.New("compile error"))

	stdout, stderr, code := captureOutput(t, func() int { return cmdUpdate(nil) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stderr, "compile error") || !strings.Contains(stdout, "wasn't changed") {
		t.Errorf("stdout = %q, stderr = %q", stdout, stderr)
	}
	if got := mustReadFile(t, exe); !strings.Contains(got, "old") {
		t.Errorf("binary changed after a failed build: %q", got)
	}
}

func TestCmdUpdate_NoPermissionPrintsSudoCommand(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write anywhere")
	}
	exe, _ := fakeUpdate(t, nil)
	dir := filepath.Dir(exe)
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	stdout, _, code := captureOutput(t, func() int { return cmdUpdate(nil) })
	if code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
	if !strings.Contains(stdout, "sudo install -m 0755 ") || !strings.Contains(stdout, exe) {
		t.Fatalf("stdout = %q, want a sudo install command", stdout)
	}
	// The built binary must survive for that command to work.
	built := strings.Fields(stdout[strings.Index(stdout, "sudo install"):])[4]
	if !lexists(built) {
		t.Errorf("built binary %s was cleaned up", built)
	}
	os.RemoveAll(filepath.Dir(built))
}

func TestCmdUpdate_RejectsArgs(t *testing.T) {
	if code := runQuiet(t, func() int { return cmdUpdate([]string{"extra"}) }); code != 1 {
		t.Errorf("code = %d, want 1", code)
	}
}

func TestCmdVersion(t *testing.T) {
	stdout, _, code := captureOutput(t, func() int { return cmdVersion(nil) })
	if code != 0 || !strings.HasPrefix(stdout, "dotmgr ") {
		t.Errorf("code = %d, stdout = %q", code, stdout)
	}
}
