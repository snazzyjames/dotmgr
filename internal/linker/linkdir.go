package linker

import (
	"bufio"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/snazzyjames/dotmgr/internal/config"
	"github.com/snazzyjames/dotmgr/internal/ignore"
	"github.com/snazzyjames/dotmgr/internal/secrets"
)

// linkedDirsFile lists, one $HOME-relative path per line, the directories
// tracked as a whole: a single symlink to the directory instead of one per
// file. It lives in the repo so every machine links them the same way.
const linkedDirsFile = ".dotmgrdirs"

// neverLink are directories too broad to link as a whole: linking one would
// sweep unrelated programs' state, caches, or keys into the repo.
var neverLink = []string{
	".", ".config", ".local", ".local/share", ".local/state", ".cache",
	".ssh", ".gnupg", "Library", "Library/Application Support",
	"Library/Preferences", "Library/Caches",
}

// LinkedDirs returns the $HOME-relative directories tracked as a whole.
func LinkedDirs(cfg *config.Config) ([]string, error) {
	f, err := os.Open(filepath.Join(cfg.Repo.Path, linkedDirsFile))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var dirs []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		dirs = append(dirs, filepath.Clean(filepath.FromSlash(line)))
	}
	return dirs, scanner.Err()
}

func saveLinkedDirs(cfg *config.Config, dirs []string) error {
	path := filepath.Join(cfg.Repo.Path, linkedDirsFile)
	if len(dirs) == 0 {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	sort.Strings(dirs)
	var b strings.Builder
	b.WriteString("# Directories dotmgr links as a whole (dotmgr add --link-dir).\n")
	b.WriteString("# Managed by dotmgr: use add --link-dir / remove rather than editing.\n")
	for _, d := range dirs {
		b.WriteString(filepath.ToSlash(d) + "\n")
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// linkedDirFor returns the linked directory that rel is, or is inside of.
func linkedDirFor(dirs []string, rel string) (string, bool) {
	for _, d := range dirs {
		if under(rel, d) {
			return d, true
		}
	}
	return "", false
}

// insideLinkedDir returns an error if abs is inside a directory that's
// linked as a whole: it's already tracked along with the directory.
func insideLinkedDir(cfg *config.Config, abs string) error {
	rel, err := relToHome(abs)
	if err != nil {
		return nil
	}
	dirs, err := LinkedDirs(cfg)
	if err != nil {
		return err
	}
	if d, ok := linkedDirFor(dirs, rel); ok && d != rel {
		return fmt.Errorf("~/%s is inside ~/%s, which is linked as a whole: it's already tracked, and to stop tracking it, remove ~/%s", filepath.ToSlash(rel), filepath.ToSlash(d), filepath.ToSlash(d))
	}
	return nil
}

type LinkDirResult struct {
	Rel       string   // the directory, relative to $HOME
	Converted []string // files that were tracked one by one before
	Ignored   []string // files kept out of git by .dotmgrignore
}

// LinkDir moves the directory at input into the repo's files/ tree and
// replaces it with a single symlink, so everything in it (including files
// created later) is tracked. Files already tracked one by one inside it are
// converted. It refuses broad directories, the repo itself, overlap with
// another linked directory, and any directory holding a denylisted secret.
func LinkDir(cfg *config.Config, input string) (*LinkDirResult, error) {
	abs, err := ResolveInput(input)
	if err != nil {
		return nil, err
	}
	info, err := os.Lstat(abs)
	if err != nil {
		return nil, fmt.Errorf("no such directory: %s", abs)
	}
	filesDir := filepath.Join(cfg.Repo.Path, "files")
	if info.Mode()&os.ModeSymlink != 0 {
		if target, err := resolveSymlinkTarget(abs); err == nil && under(target, filesDir) {
			return nil, fmt.Errorf("%s is already tracked", input)
		}
		return nil, fmt.Errorf("%s is a symlink: only real directories can be linked", input)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory: use dotmgr add without --link-dir for a file", input)
	}

	rel, err := relToHome(abs)
	if err != nil {
		return nil, err
	}
	for _, n := range neverLink {
		if rel == filepath.FromSlash(n) {
			return nil, fmt.Errorf("~/%s is too broad to link as a whole; link the specific program's directory inside it instead", filepath.ToSlash(rel))
		}
	}
	if under(abs, cfg.Repo.Path) || under(cfg.Repo.Path, abs) {
		return nil, fmt.Errorf("%s overlaps the dotfiles repo itself", input)
	}
	if cfgPath, err := config.DefaultPath(); err == nil && under(cfgPath, abs) {
		return nil, fmt.Errorf("%s contains dotmgr's own config", input)
	}

	dirs, err := LinkedDirs(cfg)
	if err != nil {
		return nil, err
	}
	for _, d := range dirs {
		if under(rel, d) || under(d, rel) {
			return nil, fmt.Errorf("%s overlaps ~/%s, which is already linked as a whole", input, filepath.ToSlash(d))
		}
	}

	pats, err := ignore.Load(filepath.Join(cfg.Repo.Path, ".dotmgrignore"))
	if err != nil {
		return nil, err
	}
	res := &LinkDirResult{Rel: rel}
	var denied, perFile []string
	err = filepath.WalkDir(abs, func(p string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		frel, err := relToHome(p)
		if err != nil {
			return err
		}
		if d.Type()&fs.ModeSymlink != 0 {
			if target, err := resolveSymlinkTarget(p); err == nil && under(target, filesDir) {
				perFile = append(perFile, p)
			}
			return nil
		}
		switch {
		case secrets.IsDenied(frel):
			denied = append(denied, frel)
		case pats.Match(filepath.ToSlash(frel)):
			res.Ignored = append(res.Ignored, frel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(denied) > 0 {
		return nil, fmt.Errorf("%s contains files on the secrets denylist, so it can't be linked as a whole: %s", input, strings.Join(denied, ", "))
	}

	// Turn per-file links back into regular files so the directory moves
	// into the repo with their content in place.
	for _, p := range perFile {
		frel, err := Remove(cfg, p)
		if err != nil {
			return nil, fmt.Errorf("converting %s: %w", p, err)
		}
		res.Converted = append(res.Converted, frel)
	}

	dest := filepath.Join(filesDir, rel)
	removeEmptyDirs(dest)
	if exists(dest) {
		return nil, fmt.Errorf("the repo already has files under %s that aren't on this machine; run dotmgr install first, then try again", filepath.Join("files", rel))
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return nil, err
	}
	if err := os.Rename(abs, dest); err != nil {
		return nil, fmt.Errorf("moving %s into the repo: %w", input, err)
	}
	if err := os.Symlink(dest, abs); err != nil {
		os.Rename(dest, abs)
		return nil, fmt.Errorf("linking %s: %w", input, err)
	}

	if err := saveLinkedDirs(cfg, append(dirs, rel)); err != nil {
		return nil, err
	}
	if err := SyncGitignore(cfg); err != nil {
		return nil, err
	}
	return res, nil
}

// removeEmptyDirs deletes dir and any directories under it that contain
// nothing but other empty directories.
func removeEmptyDirs(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			removeEmptyDirs(filepath.Join(dir, e.Name()))
		}
	}
	os.Remove(dir) // fails, harmlessly, unless now empty
}

const (
	gitignoreBegin = "# BEGIN dotmgr: generated from .dotmgrignore and the secrets denylist for linked directories; edits here are overwritten"
	gitignoreEnd   = "# END dotmgr"
)

// SyncGitignore rewrites the dotmgr-managed block of the repo's .gitignore
// so files that appear inside linked directories later are still kept out
// of git if they match .dotmgrignore or the secrets denylist. Per-file
// tracking doesn't need this: add checks each file up front.
func SyncGitignore(cfg *config.Config) error {
	dirs, err := LinkedDirs(cfg)
	if err != nil {
		return err
	}
	pats, err := ignore.Load(filepath.Join(cfg.Repo.Path, ".dotmgrignore"))
	if err != nil {
		return err
	}
	all := append(append(ignore.Patterns{}, secrets.Denylist...), pats...)

	var lines []string
	for _, d := range dirs {
		ds := filepath.ToSlash(d)
		for _, p := range all {
			p = filepath.ToSlash(p)
			switch {
			case strings.HasPrefix(p, "**/"):
				lines = append(lines, "/files/"+ds+"/"+p)
			case strings.Contains(p, "/"):
				if strings.HasPrefix(p, ds+"/") {
					lines = append(lines, "/files/"+p)
				}
			default:
				lines = append(lines, "/files/"+ds+"/**/"+p)
			}
		}
	}

	path := filepath.Join(cfg.Repo.Path, ".gitignore")
	existing, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	kept := stripManagedBlock(string(existing))

	var b strings.Builder
	b.WriteString(kept)
	if len(lines) > 0 {
		if kept != "" && !strings.HasSuffix(kept, "\n\n") {
			b.WriteString("\n")
		}
		b.WriteString(gitignoreBegin + "\n")
		for _, l := range lines {
			b.WriteString(l + "\n")
		}
		b.WriteString(gitignoreEnd + "\n")
	}

	out := b.String()
	if out == string(existing) {
		return nil
	}
	if strings.TrimSpace(out) == "" {
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	return os.WriteFile(path, []byte(out), 0o644)
}

// stripManagedBlock returns s without the dotmgr-managed block (and the
// blank line before it), leaving the user's own lines untouched.
func stripManagedBlock(s string) string {
	start := strings.Index(s, gitignoreBegin)
	if start < 0 {
		return s
	}
	end := strings.Index(s[start:], gitignoreEnd)
	if end < 0 {
		return s
	}
	end = start + end + len(gitignoreEnd)
	if end < len(s) && s[end] == '\n' {
		end++
	}
	before := strings.TrimRight(s[:start], "\n")
	if before != "" {
		before += "\n"
	}
	return before + s[end:]
}
