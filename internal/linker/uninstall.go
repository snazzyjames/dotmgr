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
// the file it points to, so nothing depends on the repo's local clone
// afterward. Files are copied, not moved, so the clone is untouched and a
// partial failure loses nothing.
//
// If restoreBackups is set, the oldest backup Install made for a file (what
// was there before dotmgr) is moved back into place. Only paths that were a
// dotmgr symlink or are now empty are restored over; anything else is left
// alone and its backups reported.
func Uninstall(cfg *config.Config, restoreBackups, dryRun bool) (*UninstallResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	filesDir := filepath.Join(cfg.Repo.Path, "files")
	res := &UninstallResult{}

	err = filepath.WalkDir(filesDir, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(filesDir, p)
		if err != nil {
			return err
		}
		target := filepath.Join(home, rel)

		linked := isCorrectSymlink(target, p)
		if linked {
			if !dryRun {
				if err := replaceWithCopy(target, p); err != nil {
					res.Failed++
					res.Errors = append(res.Errors, fmt.Sprintf("restoring %s: %v", target, err))
					return nil
				}
			}
			res.Restored = append(res.Restored, rel)
		} else {
			res.Skipped = append(res.Skipped, rel)
		}

		backups, err := findBackups(target)
		if err != nil {
			return err
		}
		if restoreBackups && len(backups) > 0 && (linked || !exists(target)) {
			oldest := backups[0]
			if !dryRun {
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
	})

	return res, err
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
