package cli

import "github.com/snazzyjames/dotmgr/internal/store"

func cmdPush(args []string) int {
	cfg, err := loadConfig()
	if err != nil {
		errf("%v", err)
		return 1
	}
	step("Pushing dotfiles")
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
