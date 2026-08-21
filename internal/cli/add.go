package cli

import (
	"flag"
	"fmt"
	"path/filepath"

	"github.com/snazzyjames/dotmgr/internal/linker"
)

func cmdAdd(args []string) int {
	fs := flag.NewFlagSet("add", flag.ContinueOnError)
	force := fs.Bool("force", false, "override the secrets denylist")
	fs.Usage = func() { fmt.Println("Usage: dotmgr add [--force] <path>") }
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		errf("expected exactly one path")
		fs.Usage()
		return 1
	}

	cfg, err := loadConfig()
	if err != nil {
		errf("%v", err)
		return 1
	}

	rel, err := linker.Add(cfg, fs.Arg(0), *force)
	if err != nil {
		errf("%v", err)
		return 1
	}

	step("Tracked %s", rel)
	info("moved into %s", filepath.Join(cfg.Repo.Path, "files", rel))
	info("linked back to ~/%s", rel)
	fmt.Println()
	fmt.Println("Don't forget: dotmgr push")
	return 0
}
