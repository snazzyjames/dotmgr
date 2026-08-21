package cli

import (
	"flag"
	"fmt"

	"github.com/snazzyjames/dotmgr/internal/linker"
)

func cmdRemove(args []string) int {
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	fs.Usage = func() { fmt.Println("Usage: dotmgr remove <path>") }
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

	rel, err := linker.Remove(cfg, fs.Arg(0))
	if err != nil {
		errf("%v", err)
		return 1
	}

	step("Untracked %s", rel)
	info("moved back to ~/%s", rel)
	fmt.Println()
	fmt.Println("Don't forget: dotmgr push")
	return 0
}
