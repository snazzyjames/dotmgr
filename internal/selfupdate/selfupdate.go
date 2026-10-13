// Package selfupdate rebuilds dotmgr with the Go toolchain and swaps the new
// binary in for the one that's running. It builds from source rather than
// downloading release binaries, matching how dotmgr is installed.
package selfupdate

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"
)

// Module is dotmgr's module path; Package is its main package.
const (
	Module  = "github.com/snazzyjames/dotmgr"
	Package = Module + "/cmd/dotmgr"
)

// ErrNoGo means the go command isn't on $PATH.
var ErrNoGo = errors.New("dotmgr update needs the Go toolchain, but go isn't on $PATH; install it from https://go.dev/dl")

// ErrPermission wraps a failure to replace the installed binary because its
// directory isn't writable (e.g. /usr/local/bin without sudo).
var ErrPermission = errors.New("no permission to replace the installed binary")

// Source says what to build: a published version of the module, or a local
// checkout of it.
type Source struct {
	Version string // e.g. "latest" or "v1.2.0"; used when Dir is empty
	Dir     string // a local checkout of dotmgr to build instead
}

func (s Source) String() string {
	if s.Dir != "" {
		return s.Dir
	}
	return Package + "@" + s.Version
}

// Build compiles dotmgr from src into a new binary inside dir and returns
// its path. Build output (compiler errors, download progress) goes to log.
func Build(src Source, dir string, log io.Writer) (string, error) {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return "", ErrNoGo
	}
	out := filepath.Join(dir, "dotmgr")

	var cmd *exec.Cmd
	if src.Dir != "" {
		if err := checkCheckout(src.Dir); err != nil {
			return "", err
		}
		cmd = exec.Command(goBin, "build", "-o", out, "./cmd/dotmgr")
		cmd.Dir = src.Dir
	} else {
		version := src.Version
		if version == "" {
			version = "latest"
		}
		// go install pkg@version ignores any surrounding module, so run it
		// from the empty build dir and point GOBIN there.
		cmd = exec.Command(goBin, "install", Package+"@"+version)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "GOBIN="+dir)
	}
	cmd.Stdout = log
	cmd.Stderr = log
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("building %s: %w", src, err)
	}
	if _, err := os.Stat(out); err != nil {
		return "", fmt.Errorf("build finished but %s is missing: %w", out, err)
	}
	return out, nil
}

// checkCheckout makes sure dir is a checkout of dotmgr, so --from can't
// build and install some other program over dotmgr.
func checkCheckout(dir string) error {
	f, err := os.Open(filepath.Join(dir, "go.mod"))
	if err != nil {
		return fmt.Errorf("%s isn't a dotmgr checkout (no go.mod)", dir)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			if strings.TrimSpace(strings.TrimPrefix(line, "module ")) == Module {
				return nil
			}
			break
		}
	}
	return fmt.Errorf("%s isn't a dotmgr checkout (its go.mod isn't module %s)", dir, Module)
}

// Installed returns the path of the running dotmgr binary, following
// symlinks so the real file is replaced rather than a link to it.
func Installed() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(exe)
}

// Replace swaps newBin in for the binary at exe. The new binary is copied
// next to exe first and renamed over it, so exe is never left half-written;
// replacing a running binary this way is safe on macOS and Linux. If exe's
// directory isn't writable, the error wraps ErrPermission.
func Replace(newBin, exe string) error {
	data, err := os.ReadFile(newBin)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(exe), ".dotmgr-update-*")
	if err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return fmt.Errorf("%w: %s", ErrPermission, filepath.Dir(exe))
		}
		return err
	}
	defer os.Remove(tmp.Name()) // no-op once renamed into place
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	if err := os.Rename(tmp.Name(), exe); err != nil {
		if errors.Is(err, fs.ErrPermission) {
			return fmt.Errorf("%w: %s", ErrPermission, exe)
		}
		return err
	}
	return nil
}

// Version describes the running binary from the build info Go embeds: the
// module version for a go install @version build, or the git commit for a
// build from a checkout.
func Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	return describe(info)
}

func describe(info *debug.BuildInfo) string {
	var rev, modified string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
		case "vcs.modified":
			modified = s.Value
		}
	}
	v := info.Main.Version
	if v != "" && v != "(devel)" {
		return v
	}
	if rev == "" {
		return "dev"
	}
	if len(rev) > 12 {
		rev = rev[:12]
	}
	if modified == "true" {
		rev += "-dirty"
	}
	return "dev (" + rev + ")"
}
