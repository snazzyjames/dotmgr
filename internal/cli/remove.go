package cli

import (
	"flag"
	"fmt"

	"github.com/snazzyjames/dotmgr/internal/linker"
)

func cmdRemove(args []string) int {
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	fs.Usage = func() { fmt.Println("Usage: dotmgr remove <file-or-directory>") }
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

	if linker.IsDir(fs.Arg(0)) {
		res, err := linker.RemoveDir(cfg, fs.Arg(0))
		if err != nil {
			errf("%v", err)
			return 1
		}
		return printDirResult(res, "Untracked", "not tracked", "Don't forget: dotmgr push")
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

func cmdRelink(args []string) int {
	fs := flag.NewFlagSet("relink", flag.ContinueOnError)
	fs.Usage = func() { fmt.Println("Usage: dotmgr relink <path>") }
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

	rel, err := linker.Relink(cfg, fs.Arg(0))
	if err != nil {
		errf("%v", err)
		return 1
	}

	step("Relinked ~/%s", rel)
	info("this machine's version is now the repo's copy")
	fmt.Println()
	fmt.Println("Don't forget: dotmgr push")
	return 0
}
