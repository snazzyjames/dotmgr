package linker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveHomeInput accepts an absolute path, a "~/"-prefixed path, or a path
// relative to $HOME, and returns an absolute path.
func resolveHomeInput(input string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch {
	case input == "~":
		input = home
	case strings.HasPrefix(input, "~/"):
		input = filepath.Join(home, strings.TrimPrefix(input, "~/"))
	}
	if !filepath.IsAbs(input) {
		input = filepath.Join(home, input)
	}
	return filepath.Clean(input), nil
}

// relToHome returns abs relative to $HOME, erroring if abs isn't inside it.
func relToHome(abs string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(home, abs)
	if err != nil {
		return "", err
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s is not inside $HOME (%s): only home-directory dotfiles are supported", abs, home)
	}
	return rel, nil
}

// resolveSymlink follows a symlink at abs one hop and returns an absolute
// target path.
func resolveSymlinkTarget(abs string) (string, error) {
	target, err := os.Readlink(abs)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(abs), target)
	}
	return filepath.Clean(target), nil
}

func under(path, dir string) bool {
	dir = filepath.Clean(dir)
	path = filepath.Clean(path)
	return path == dir || strings.HasPrefix(path, dir+string(filepath.Separator))
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
