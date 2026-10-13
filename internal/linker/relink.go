package linker

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/snazzyjames/dotmgr/internal/config"
)

// Relink restores the symlink for a tracked file that a regular file has
// replaced (see StateUnlinked and StateDiverged), keeping this machine's
// version: the file at input becomes the repo's copy, and the link goes back
// in its place. To keep the repo's version instead, run Install, which backs
// up the file in the way. It returns the path relative to $HOME.
func Relink(cfg *config.Config, input string) (string, error) {
	abs, err := ResolveInput(input)
	if err != nil {
		return "", err
	}
	rel, err := relToHome(abs)
	if err != nil {
		return "", err
	}
	if err := insideLinkedDir(cfg, abs); err != nil {
		return "", err
	}

	src := filepath.Join(cfg.Repo.Path, "files", rel)
	srcInfo, err := os.Lstat(src)
	if err != nil || !srcInfo.Mode().IsRegular() {
		return "", fmt.Errorf("~/%s isn't tracked", filepath.ToSlash(rel))
	}

	switch fileState(abs, src) {
	case StateOK:
		return "", fmt.Errorf("~/%s is already linked", filepath.ToSlash(rel))
	case StateMissing:
		return "", fmt.Errorf("~/%s doesn't exist on this machine; run dotmgr install to link it", filepath.ToSlash(rel))
	case StateConflict:
		return "", fmt.Errorf("~/%s isn't a regular file; move it out of the way and run dotmgr install", filepath.ToSlash(rel))
	}

	if err := os.Rename(abs, src); err != nil {
		return "", fmt.Errorf("moving %s into the repo: %w", input, err)
	}
	if err := os.Symlink(src, abs); err != nil {
		return "", fmt.Errorf("moved to %s but failed to symlink back (%w); restore with: mv %q %q", src, err, src, abs)
	}
	return rel, nil
}
