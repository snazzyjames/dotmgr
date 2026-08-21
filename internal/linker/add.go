package linker

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/snazzyjames/dotmgr/internal/config"
	"github.com/snazzyjames/dotmgr/internal/secrets"
)

// Add moves the file at input into the repo's files/ tree (mirroring its
// path under $HOME) and replaces it with a symlink back. It returns the
// path relative to $HOME on success.
func Add(cfg *config.Config, input string, force bool) (string, error) {
	abs, err := resolveHomeInput(input)
	if err != nil {
		return "", err
	}

	info, err := os.Lstat(abs)
	if err != nil {
		return "", fmt.Errorf("no such file: %s", abs)
	}

	filesDir := filepath.Join(cfg.Repo.Path, "files")

	if info.Mode()&os.ModeSymlink != 0 {
		target, terr := resolveSymlinkTarget(abs)
		if terr == nil && under(target, filesDir) {
			return "", fmt.Errorf("%s is already tracked (symlinked to %s)", input, target)
		}
		return "", fmt.Errorf("%s is a symlink: only real files are supported by add", input)
	}

	if info.IsDir() {
		return "", fmt.Errorf("%s is a directory: add the files inside it individually", input)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("%s is not a regular file", input)
	}

	rel, err := relToHome(abs)
	if err != nil {
		return "", err
	}

	if !force && secrets.IsDenied(rel) {
		return "", fmt.Errorf("%s matches the built-in secrets denylist, re-run with --force if you're sure", rel)
	}

	dest := filepath.Join(filesDir, rel)
	if exists(dest) {
		return "", fmt.Errorf("%s is already tracked at %s", rel, dest)
	}

	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return "", fmt.Errorf("creating %s: %w", filepath.Dir(dest), err)
	}
	if err := os.Rename(abs, dest); err != nil {
		return "", fmt.Errorf("moving %s into the repo: %w", input, err)
	}
	if err := os.Symlink(dest, abs); err != nil {
		return "", fmt.Errorf("moved to %s but failed to symlink back (%w); restore with: mv %q %q", dest, err, dest, abs)
	}

	return rel, nil
}
