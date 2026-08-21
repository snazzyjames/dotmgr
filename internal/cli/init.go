package cli

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/snazzyjames/dotmgr/internal/config"
	"github.com/snazzyjames/dotmgr/internal/store"
)

const defaultIgnore = `# Default .dotmgrignore — edit freely, this is a starting point.
# Gitignore-style patterns. Checked by "dotmgr scan" (won't suggest a match)
# and "dotmgr add" (refuses a match unless --force).

# private key material — never the config that references it
.ssh/id_*
.ssh/*.pem
.gnupg/private-keys-*
.aws/credentials

# per-install state, not portable config
**/fish_variables
**/*.bak.*

# noise
**/.DS_Store
**/node_modules
`

const defaultRepoReadme = `# dotfiles

Managed with [dotmgr](https://github.com/snazzyjames/dotmgr).

Layout mirrors ` + "`$HOME`" + ` under ` + "`files/`" + `: ` + "`files/.config/fish/config.fish`" + ` links to
` + "`~/.config/fish/config.fish`" + `.

    dotmgr add ~/.gitconfig     # track a file
    dotmgr install              # symlink everything here into place
    dotmgr scan                 # suggest untracked files a known tool owns
`

func cmdInit(args []string) int {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.Usage = func() { fmt.Println("Usage: dotmgr init <remote-url>") }
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		errf("expected exactly one argument: the git remote URL")
		fs.Usage()
		return 1
	}
	remote := fs.Arg(0)

	cfgPath, err := config.DefaultPath()
	if err != nil {
		errf("%v", err)
		return 1
	}
	if _, err := os.Stat(cfgPath); err == nil {
		errf("dotmgr is already initialized (%s exists)", cfgPath)
		return 1
	}

	home, err := os.UserHomeDir()
	if err != nil {
		errf("%v", err)
		return 1
	}
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(home, ".local", "share")
	}
	repoPath := filepath.Join(dataHome, "dotmgr", "repo")

	step("Setting up dotmgr")
	info("remote: %s", remote)
	info("local clone: %s", repoPath)

	if err := os.MkdirAll(filepath.Dir(repoPath), 0o755); err != nil {
		errf("%v", err)
		return 1
	}

	if err := store.Clone(remote, repoPath); err != nil {
		info("clone failed (likely an empty remote) — initializing a new repo instead")
		if err := store.InitNew(repoPath, remote); err != nil {
			errf("%v", err)
			return 1
		}
	} else {
		info("cloned existing repo")
	}

	filesDir := filepath.Join(repoPath, "files")
	if err := os.MkdirAll(filesDir, 0o755); err != nil {
		errf("%v", err)
		return 1
	}

	ignorePath := filepath.Join(repoPath, ".dotmgrignore")
	if _, err := os.Stat(ignorePath); os.IsNotExist(err) {
		if err := os.WriteFile(ignorePath, []byte(defaultIgnore), 0o644); err != nil {
			errf("%v", err)
			return 1
		}
		info("wrote default .dotmgrignore")
	}

	readmePath := filepath.Join(repoPath, "README.md")
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		if err := os.WriteFile(readmePath, []byte(defaultRepoReadme), 0o644); err != nil {
			errf("%v", err)
			return 1
		}
	}

	cfg := &config.Config{}
	cfg.Repo.Remote = remote
	cfg.Repo.Path = repoPath
	cfg.Machine.Name, _ = os.Hostname()

	if err := config.Save(cfgPath, cfg); err != nil {
		errf("%v", err)
		return 1
	}
	info("wrote config: %s", cfgPath)

	fmt.Println()
	fmt.Println("Done. Track your first file with: dotmgr add ~/.gitconfig")
	return 0
}
