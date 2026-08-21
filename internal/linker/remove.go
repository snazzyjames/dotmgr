package linker

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/snazzyjames/dotmgr/internal/config"
)

// Remove reverses Add: it deletes the symlink at input and moves the real
// file from the repo back into place. It returns the path relative to
// $HOME on success.
func Remove(cfg *config.Config, input string) (string, error) {
	abs, err := resolveHomeInput(input)
	if err != nil {
		return "", err
	}

	info, err := os.Lstat(abs)
	if err != nil {
		return "", fmt.Errorf("no such file: %s", abs)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return "", fmt.Errorf("%s is not a symlink managed by dotmgr", input)
	}

	target, err := resolveSymlinkTarget(abs)
	if err != nil {
		return "", err
	}

	filesDir := filepath.Join(cfg.Repo.Path, "files")
	if !under(target, filesDir) {
		return "", fmt.Errorf("%s does not point into the dotfiles repo (%s)", input, target)
	}

	rel, err := filepath.Rel(filesDir, target)
	if err != nil {
		return "", err
	}

	if err := os.Remove(abs); err != nil {
		return "", fmt.Errorf("removing symlink %s: %w", input, err)
	}
	if err := os.Rename(target, abs); err != nil {
		return "", fmt.Errorf("moving %s back out of the repo: %w", rel, err)
	}

	return rel, nil
}
