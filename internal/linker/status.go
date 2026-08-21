package linker

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/snazzyjames/dotmgr/internal/config"
)

type TrackedFile struct {
	Rel   string
	State string // "ok", "missing", "conflict"
}

// Status reports the link health of every file tracked under files/,
// without changing anything.
func Status(cfg *config.Config) ([]TrackedFile, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	filesDir := filepath.Join(cfg.Repo.Path, "files")

	var out []TrackedFile
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

		state := "conflict"
		switch {
		case isCorrectSymlink(target, p):
			state = "ok"
		case !exists(target):
			state = "missing"
		}
		out = append(out, TrackedFile{Rel: rel, State: state})
		return nil
	})
	return out, err
}
