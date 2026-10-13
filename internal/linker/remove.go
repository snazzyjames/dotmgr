package linker

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/snazzyjames/dotmgr/internal/config"
)

// Remove reverses Add (or LinkDir): it deletes the symlink at input and
// moves the real file or directory from the repo back into place. It
// returns the path relative to $HOME on success.
func Remove(cfg *config.Config, input string) (string, error) {
	abs, err := ResolveInput(input)
	if err != nil {
		return "", err
	}

	info, err := os.Lstat(abs)
	if err != nil {
		return "", fmt.Errorf("no such file: %s", abs)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		if err := insideLinkedDir(cfg, abs); err != nil {
			return "", err
		}
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

	// A directory linked as a whole is no longer one.
	dirs, err := LinkedDirs(cfg)
	if err != nil {
		return "", err
	}
	for i, d := range dirs {
		if d == rel {
			if err := saveLinkedDirs(cfg, append(dirs[:i:i], dirs[i+1:]...)); err != nil {
				return "", err
			}
			if err := SyncGitignore(cfg); err != nil {
				return "", err
			}
			break
		}
	}

	return rel, nil
}
