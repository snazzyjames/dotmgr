// Package cli implements dotmgr's command dispatch and each subcommand.
package cli

import "fmt"

func Run(args []string) int {
	if len(args) == 0 {
		usage()
		return 1
	}

	cmd, rest := args[0], args[1:]
	switch cmd {
	case "init":
		return cmdInit(rest)
	case "add":
		return cmdAdd(rest)
	case "remove", "rm":
		return cmdRemove(rest)
	case "install", "apply":
		return cmdInstall(rest)
	case "status":
		return cmdStatus(rest)
	case "scan":
		return cmdScan(rest)
	case "push":
		return cmdPush(rest)
	case "pull":
		return cmdPull(rest)
	case "-h", "--help", "help":
		usage()
		return 0
	default:
		errf("unknown command: %s", cmd)
		usage()
		return 1
	}
}

func usage() {
	fmt.Println(`dotmgr: track dotfiles in a git repo, symlink them into place anywhere.

Usage:
  dotmgr init <remote>     Clone (or create) the dotfiles repo
  dotmgr add <path>        Track a file: move it into the repo, symlink it back
  dotmgr remove <path>     Reverse of add
  dotmgr install           Symlink every tracked file into place
  dotmgr status            Show tracked files, drift, and untracked candidates
  dotmgr scan              Suggest untracked files a known tool owns
  dotmgr push              Commit and push the dotfiles repo
  dotmgr pull              Pull the latest from the dotfiles repo

Flags:
  add --force              Override the secrets denylist for one file
  install --dry-run, -n    Show what would happen without changing anything`)
}
