package linker

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/snazzyjames/dotmgr/internal/config"
	"github.com/snazzyjames/dotmgr/internal/ignore"
	"github.com/snazzyjames/dotmgr/internal/secrets"
)

type DirResult struct {
	Done    []string // $HOME-relative paths added (or removed)
	Already []string // already tracked, nothing to do (AddDir only)
	Ignored []string // matched .dotmgrignore
	Denied  []string // matched the secrets denylist
	Errors  []string
}

// AddDir tracks every regular file under the directory at input, one file
// at a time, exactly as Add would. Files that are already tracked are
// skipped, so re-running it picks up files created since. Files matching
// .dotmgrignore or the secrets denylist are skipped and reported; --force
// deliberately doesn't apply here, so a whole tree can't sweep secrets in.
func AddDir(cfg *config.Config, input string) (*DirResult, error) {
	root, err := ResolveInput(input)
	if err != nil {
		return nil, err
	}
	if _, err := relToHome(root); err != nil {
		return nil, err
	}
	if err := insideLinkedDir(cfg, root); err != nil {
		return nil, err
	}
	filesDir := filepath.Join(cfg.Repo.Path, "files")
	if under(root, cfg.Repo.Path) {
		return nil, fmt.Errorf("%s is inside the dotfiles repo itself", input)
	}
	pats, err := ignore.Load(filepath.Join(cfg.Repo.Path, ".dotmgrignore"))
	if err != nil {
		return nil, err
	}

	res := &DirResult{}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		rel, err := relToHome(p)
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			if target, err := resolveSymlinkTarget(p); err == nil && under(target, filesDir) {
				res.Already = append(res.Already, rel)
			}
			// Other symlinks aren't ours to move; add only takes real files.
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		switch {
		case pats.Match(filepath.ToSlash(rel)):
			res.Ignored = append(res.Ignored, rel)
		case secrets.IsDenied(rel):
			res.Denied = append(res.Denied, rel)
		default:
			if _, err := Add(cfg, p, false); err != nil {
				res.Errors = append(res.Errors, err.Error())
				return nil
			}
			res.Done = append(res.Done, rel)
		}
		return nil
	})
	return res, err
}

// RemoveDir untracks every dotmgr symlink under the directory at input, one
// file at a time, exactly as Remove would. Anything that isn't a dotmgr
// symlink is left alone.
func RemoveDir(cfg *config.Config, input string) (*DirResult, error) {
	root, err := ResolveInput(input)
	if err != nil {
		return nil, err
	}
	filesDir := filepath.Join(cfg.Repo.Path, "files")

	res := &DirResult{}
	err = filepath.WalkDir(root, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || d.Type()&fs.ModeSymlink == 0 {
			return nil
		}
		target, err := resolveSymlinkTarget(p)
		if err != nil || !under(target, filesDir) {
			return nil
		}
		rel, err := Remove(cfg, p)
		if err != nil {
			res.Errors = append(res.Errors, err.Error())
			return nil
		}
		res.Done = append(res.Done, rel)
		return nil
	})
	return res, err
}

// IsDir reports whether input, resolved like any other path argument, is a
// directory (not following a final symlink).
func IsDir(input string) bool {
	abs, err := ResolveInput(input)
	if err != nil {
		return false
	}
	info, err := os.Lstat(abs)
	return err == nil && info.IsDir()
}
