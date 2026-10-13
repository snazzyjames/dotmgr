package linker

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/snazzyjames/dotmgr/internal/config"
)

type UninstallResult struct {
	Restored        []string // replaced a dotmgr symlink with a regular copy
	Skipped         []string // not linked by dotmgr on this machine, left alone
	BackupsRestored []string // a .bak.<timestamp> moved back over its original path
	Backups         []string // .bak.<timestamp> files left in place
	Failed          int
	Errors          []string
}

// Uninstall replaces every dotmgr symlink under $HOME with a regular copy of
// the file (or, for a directory linked as a whole, the directory) it points
// to, so nothing depends on the repo's local clone afterward. Content is
// copied, not moved, so the clone is untouched and a partial failure loses
// nothing.
//
// If restoreBackups is set, the oldest backup Install made for a path (what
// was there before dotmgr) is moved back into place instead. Only paths that
// were a dotmgr symlink or are now empty are restored over; anything else is
// left alone and its backups reported.
func Uninstall(cfg *config.Config, restoreBackups, dryRun bool) (*UninstallResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	filesDir := filepath.Join(cfg.Repo.Path, "files")
	dirs, err := LinkedDirs(cfg)
	if err != nil {
		return nil, err
	}
	res := &UninstallResult{}

	for _, d := range dirs {
		if err := uninstallOne(res, filepath.Join(filesDir, d), filepath.Join(home, d), d, true, restoreBackups, dryRun); err != nil {
			return res, err
		}
	}

	err = filepath.WalkDir(filesDir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(filesDir, p)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if _, ok := linkedDirFor(dirs, rel); ok {
				return fs.SkipDir // handled above
			}
			return nil
		}
		return uninstallOne(res, p, filepath.Join(home, rel), rel, false, restoreBackups, dryRun)
	})

	return res, err
}

func uninstallOne(res *UninstallResult, src, target, rel string, isDir, restoreBackups, dryRun bool) error {
	linked := isCorrectSymlink(target, src)
	backups, err := findBackups(target)
	if err != nil {
		return err
	}
	restore := restoreBackups && len(backups) > 0 && (linked || !exists(target))

	if linked {
		if !dryRun && !restore {
			copyFn := replaceWithCopy
			if isDir {
				copyFn = replaceDirWithCopy
			}
			if err := copyFn(target, src); err != nil {
				res.Failed++
				res.Errors = append(res.Errors, fmt.Sprintf("restoring %s: %v", target, err))
				res.Backups = append(res.Backups, backups...)
				return nil
			}
		}
		res.Restored = append(res.Restored, rel)
	} else {
		res.Skipped = append(res.Skipped, rel)
	}

	if restore {
		oldest := backups[0]
		if !dryRun {
			if linked {
				if err := os.Remove(target); err != nil {
					res.Failed++
					res.Errors = append(res.Errors, fmt.Sprintf("removing symlink %s: %v", target, err))
					res.Backups = append(res.Backups, backups...)
					return nil
				}
			}
			if err := os.Rename(oldest, target); err != nil {
				res.Failed++
				res.Errors = append(res.Errors, fmt.Sprintf("restoring backup %s: %v", oldest, err))
				res.Backups = append(res.Backups, backups...)
				return nil
			}
		}
		res.BackupsRestored = append(res.BackupsRestored, oldest)
		backups = backups[1:]
	}
	res.Backups = append(res.Backups, backups...)
	return nil
}

// replaceDirWithCopy swaps the symlink at target for a regular directory
// holding a copy of src. The copy is built beside target first, so a failed
// copy leaves the symlink in place.
func replaceDirWithCopy(target, src string) error {
	tmp, err := os.MkdirTemp(filepath.Dir(target), "."+filepath.Base(target)+".dotmgr-*")
	if err != nil {
		return err
	}
	if err := copyTree(src, tmp); err != nil {
		os.RemoveAll(tmp)
		return err
	}
	if err := os.Remove(target); err != nil {
		os.RemoveAll(tmp)
		return err
	}
	return os.Rename(tmp, target)
}

// copyTree copies the contents of directory src into the existing directory
// dst, keeping permissions and recreating symlinks as symlinks.
func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		info, err := d.Info()
		if err != nil {
			return err
		}
		switch {
		case d.IsDir():
			if rel == "." {
				return os.Chmod(dst, info.Mode().Perm())
			}
			return os.Mkdir(out, info.Mode().Perm())
		case d.Type()&fs.ModeSymlink != 0:
			link, err := os.Readlink(p)
			if err != nil {
				return err
			}
			return os.Symlink(link, out)
		case d.Type().IsRegular():
			data, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			return os.WriteFile(out, data, info.Mode().Perm())
		}
		return nil // sockets, devices: nothing sensible to copy
	})
}

// replaceWithCopy atomically swaps the symlink at target for a regular file
// holding src's contents and permissions.
func replaceWithCopy(target, src string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".dotmgr-*")
	if err != nil {
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
	if err := os.Chmod(tmp.Name(), info.Mode().Perm()); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), target)
}

// findBackups returns the backups Install made for target
// (<target>.bak.<YYYYMMDDHHMMSS>), oldest first.
func findBackups(target string) ([]string, error) {
	matches, err := filepath.Glob(target + ".bak.[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]")
	if err != nil {
		return nil, err
	}
	// The timestamp format sorts lexically in time order.
	sort.Strings(matches)
	return matches, nil
}
