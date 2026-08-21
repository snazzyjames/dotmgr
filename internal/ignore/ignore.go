// Package ignore matches paths against gitignore-style patterns. It covers
// the subset dotmgr's own default patterns use — a literal path, a
// single-segment glob (id_*, *.pem), and a "**/name" prefix meaning "at any
// depth" — not the full gitignore spec.
package ignore

import (
	"bufio"
	"os"
	"path"
	"strings"
)

type Patterns []string

// Load reads a pattern file, skipping blank lines and "#" comments. A
// missing file yields an empty, harmless Patterns value.
func Load(file string) (Patterns, error) {
	f, err := os.Open(file)
	if os.IsNotExist(err) {
		return Patterns{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pats Patterns
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		pats = append(pats, line)
	}
	return pats, scanner.Err()
}

// Match reports whether rel (a slash-separated path relative to the
// dotfiles repo's files/ root) is excluded by any pattern.
func (pats Patterns) Match(rel string) bool {
	rel = path.Clean(filepathToSlash(rel))
	base := path.Base(rel)

	for _, pat := range pats {
		pat = filepathToSlash(pat)
		switch {
		case strings.HasPrefix(pat, "**/"):
			suffix := strings.TrimPrefix(pat, "**/")
			for _, seg := range strings.Split(rel, "/") {
				if ok, _ := path.Match(suffix, seg); ok {
					return true
				}
			}
		case strings.Contains(pat, "/"):
			if ok, _ := path.Match(pat, rel); ok {
				return true
			}
		default:
			if ok, _ := path.Match(pat, base); ok {
				return true
			}
		}
	}
	return false
}

func filepathToSlash(s string) string {
	return strings.ReplaceAll(s, "\\", "/")
}
