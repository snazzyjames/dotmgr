package cli

import (
	"github.com/snazzyjames/dotmgr/internal/linker"
	"github.com/snazzyjames/dotmgr/internal/store"
)

func cmdPush(args []string) int {
	cfg, err := loadConfig()
	if err != nil {
		errf("%v", err)
		return 1
	}
	step("Pushing dotfiles")
	// .dotmgrignore may have changed since a directory was linked.
	if err := linker.SyncGitignore(cfg); err != nil {
		errf("%v", err)
		return 1
	}
	if err := store.Push(cfg.Repo.Path); err != nil {
		errf("%v", err)
		return 1
	}
	info("done")
	return 0
}

func cmdPull(args []string) int {
	cfg, err := loadConfig()
	if err != nil {
		errf("%v", err)
		return 1
	}
	step("Pulling dotfiles")
	if err := store.Pull(cfg.Repo.Path); err != nil {
		errf("%v", err)
		return 1
	}
	info("done")
	return 0
}
