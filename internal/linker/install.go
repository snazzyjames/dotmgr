package linker

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/snazzyjames/dotmgr/internal/config"
)

type InstallResult struct {
	Linked   int
	Skipped  int
	BackedUp int
	Failed   int
	Errors   []string
}

// Install symlinks every file tracked under the repo's files/ tree into its
// matching path under $HOME, creating parent directories as needed.
// Anything already at a target that isn't the expected symlink is backed up
// (renamed with a timestamp suffix), never deleted. Safe to re-run.
func Install(cfg *config.Config, dryRun bool) (*InstallResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	filesDir := filepath.Join(cfg.Repo.Path, "files")
	res := &InstallResult{}

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

		if isCorrectSymlink(target, p) {
			res.Skipped++
			return nil
		}

		if exists(target) {
			if dryRun {
				res.BackedUp++
			} else {
				backup := fmt.Sprintf("%s.bak.%s", target, time.Now().Format("20060102150405"))
				if err := os.Rename(target, backup); err != nil {
					res.Failed++
					res.Errors = append(res.Errors, fmt.Sprintf("backing up %s: %v", target, err))
					return nil
				}
				res.BackedUp++
			}
		}

		if dryRun {
			res.Linked++
			return nil
		}

		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			res.Failed++
			res.Errors = append(res.Errors, fmt.Sprintf("creating directory for %s: %v", target, err))
			return nil
		}
		if err := os.Symlink(p, target); err != nil {
			res.Failed++
			res.Errors = append(res.Errors, fmt.Sprintf("linking %s: %v", target, err))
			return nil
		}
		res.Linked++
		return nil
	})

	return res, err
}

func isCorrectSymlink(target, expected string) bool {
	info, err := os.Lstat(target)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	got, err := resolveSymlinkTarget(target)
	if err != nil {
		return false
	}
	return got == filepath.Clean(expected)
}
