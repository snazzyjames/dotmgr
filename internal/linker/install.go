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
// Directories linked as a whole get one symlink for the directory instead.
// Anything already at a target that isn't the expected symlink is backed up
// (renamed with a timestamp suffix), never deleted. Safe to re-run.
func Install(cfg *config.Config, dryRun bool) (*InstallResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	filesDir := filepath.Join(cfg.Repo.Path, "files")
	dirs, err := LinkedDirs(cfg)
	if err != nil {
		return nil, err
	}
	res := &InstallResult{}

	for _, d := range dirs {
		src := filepath.Join(filesDir, d)
		if info, err := os.Stat(src); err != nil || !info.IsDir() {
			res.Failed++
			res.Errors = append(res.Errors, fmt.Sprintf("%s is listed in %s but %s isn't a directory in the repo", d, linkedDirsFile, src))
			continue
		}
		link(res, src, filepath.Join(home, d), dryRun)
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
				return fs.SkipDir // linked as a whole above
			}
			return nil
		}
		link(res, p, filepath.Join(home, rel), dryRun)
		return nil
	})

	return res, err
}

// link points target at src, backing up whatever is in the way.
func link(res *InstallResult, src, target string, dryRun bool) {
	if isCorrectSymlink(target, src) {
		res.Skipped++
		return
	}

	if exists(target) {
		if dryRun {
			res.BackedUp++
		} else {
			backup := fmt.Sprintf("%s.bak.%s", target, time.Now().Format("20060102150405"))
			if err := os.Rename(target, backup); err != nil {
				res.Failed++
				res.Errors = append(res.Errors, fmt.Sprintf("backing up %s: %v", target, err))
				return
			}
			res.BackedUp++
		}
	}

	if dryRun {
		res.Linked++
		return
	}

	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		res.Failed++
		res.Errors = append(res.Errors, fmt.Sprintf("creating directory for %s: %v", target, err))
		return
	}
	if err := os.Symlink(src, target); err != nil {
		res.Failed++
		res.Errors = append(res.Errors, fmt.Sprintf("linking %s: %v", target, err))
		return
	}
	res.Linked++
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
