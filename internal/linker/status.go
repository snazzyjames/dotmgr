package linker

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/snazzyjames/dotmgr/internal/config"
)

// Link states reported by Status.
const (
	StateOK       = "ok"       // the expected symlink is in place
	StateMissing  = "missing"  // nothing at the target
	StateUnlinked = "unlinked" // a regular file with the repo's exact content
	StateDiverged = "diverged" // a regular file with different content
	StateConflict = "conflict" // something else: a directory, another symlink
)

type TrackedFile struct {
	Rel   string
	Dir   bool // linked as a whole directory
	State string
}

// Status reports the link health of every file tracked under files/ (and
// every directory linked as a whole), without changing anything.
//
// A regular file where a symlink should be usually means a program saved
// over the link by writing a new file and renaming it into place, so its
// changes no longer reach the repo. StateUnlinked and StateDiverged tell
// apart whether anything was lost from the repo's point of view.
func Status(cfg *config.Config) ([]TrackedFile, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	filesDir := filepath.Join(cfg.Repo.Path, "files")
	dirs, err := LinkedDirs(cfg)
	if err != nil {
		return nil, err
	}

	var out []TrackedFile
	for _, d := range dirs {
		target := filepath.Join(home, d)
		state := StateConflict
		switch {
		case isCorrectSymlink(target, filepath.Join(filesDir, d)):
			state = StateOK
		case !exists(target):
			state = StateMissing
		}
		out = append(out, TrackedFile{Rel: d, Dir: true, State: state})
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
				return fs.SkipDir
			}
			return nil
		}
		out = append(out, TrackedFile{Rel: rel, State: fileState(filepath.Join(home, rel), p)})
		return nil
	})
	return out, err
}

func fileState(target, src string) string {
	if isCorrectSymlink(target, src) {
		return StateOK
	}
	info, err := os.Lstat(target)
	if err != nil {
		return StateMissing
	}
	if !info.Mode().IsRegular() {
		return StateConflict
	}
	if sameContent(target, src) {
		return StateUnlinked
	}
	return StateDiverged
}

func sameContent(a, b string) bool {
	ab, err := os.ReadFile(a)
	if err != nil {
		return false
	}
	bb, err := os.ReadFile(b)
	if err != nil {
		return false
	}
	return bytes.Equal(ab, bb)
}
