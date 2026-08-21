// Package store wraps the git operations dotmgr needs against a dotfiles
// repo's local clone: dotmgr never talks to GitHub's (or anyone else's)
// API directly, only to whatever git remote the user configured.
package store

import (
	"fmt"
	"os/exec"
	"strings"
)

// Clone clones remote into path.
func Clone(remote, path string) error {
	_, err := run("", "clone", remote, path)
	return err
}

// InitNew git-inits an empty local repo at path and points "origin" at
// remote, for the case where remote is a freshly created, still-empty repo
// that "git clone" refuses to treat as a real clone target.
func InitNew(path, remote string) error {
	if _, err := run("", "init", "-q", path); err != nil {
		return err
	}
	if _, err := run(path, "remote", "add", "origin", remote); err != nil {
		return err
	}
	return nil
}

// Dirty reports whether repoPath has uncommitted changes.
func Dirty(repoPath string) (bool, error) {
	out, err := run(repoPath, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// CommitAll stages everything in repoPath and commits, if there's anything
// to commit. A clean tree is not an error.
func CommitAll(repoPath, message string) error {
	dirty, err := Dirty(repoPath)
	if err != nil {
		return err
	}
	if !dirty {
		return nil
	}
	if _, err := run(repoPath, "add", "-A"); err != nil {
		return err
	}
	_, err = run(repoPath, "commit", "-q", "-m", message)
	return err
}

// Push commits any pending changes, then pushes.
func Push(repoPath string) error {
	if err := CommitAll(repoPath, "dotmgr: sync"); err != nil {
		return err
	}
	_, err := run(repoPath, "push")
	return err
}

// Pull pulls the latest from origin.
func Pull(repoPath string) error {
	_, err := run(repoPath, "pull")
	return err
}

func run(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("git %s: %w\n%s", strings.Join(args, " "), err, out)
	}
	return string(out), nil
}
