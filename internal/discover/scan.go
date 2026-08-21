package discover

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/snazzyjames/dotmgr/internal/config"
	"github.com/snazzyjames/dotmgr/internal/ignore"
	"github.com/snazzyjames/dotmgr/internal/secrets"
)

// Candidates walks the registry entries for the current OS and returns
// $HOME-relative paths that exist on disk, aren't already tracked, and
// aren't excluded by the repo's .dotmgrignore or the secrets denylist.
func Candidates(cfg *config.Config) ([]string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	filesDir := filepath.Join(cfg.Repo.Path, "files")
	ignorePats, err := ignore.Load(filepath.Join(cfg.Repo.Path, ".dotmgrignore"))
	if err != nil {
		return nil, err
	}

	seen := map[string]bool{}
	var out []string

	for _, entry := range Registry {
		for _, p := range entry.Paths[runtime.GOOS] {
			root := expandHome(p, home)
			if _, err := os.Lstat(root); err != nil {
				continue
			}
			_ = filepath.WalkDir(root, func(fp string, d fs.DirEntry, walkErr error) error {
				if walkErr != nil || d.IsDir() {
					return nil
				}
				rel, err := filepath.Rel(home, fp)
				if err != nil || seen[rel] {
					return nil
				}
				if ignorePats.Match(rel) || secrets.IsDenied(rel) {
					return nil
				}
				if alreadyTracked(fp, filesDir) {
					return nil
				}
				if _, err := os.Lstat(filepath.Join(filesDir, rel)); err == nil {
					return nil
				}
				seen[rel] = true
				out = append(out, rel)
				return nil
			})
		}
	}

	sort.Strings(out)
	return out, nil
}

func alreadyTracked(fp, filesDir string) bool {
	info, err := os.Lstat(fp)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(fp)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(fp), target)
	}
	target = filepath.Clean(target)
	filesDir = filepath.Clean(filesDir)
	return target == filesDir || strings.HasPrefix(target, filesDir+string(filepath.Separator))
}

func expandHome(p, home string) string {
	switch {
	case p == "~":
		return home
	case strings.HasPrefix(p, "~/"):
		return filepath.Join(home, strings.TrimPrefix(p, "~/"))
	default:
		return p
	}
}
