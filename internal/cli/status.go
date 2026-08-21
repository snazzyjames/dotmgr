package cli

import (
	"fmt"

	"github.com/snazzyjames/dotmgr/internal/discover"
	"github.com/snazzyjames/dotmgr/internal/linker"
)

func cmdStatus(args []string) int {
	cfg, err := loadConfig()
	if err != nil {
		errf("%v", err)
		return 1
	}

	step("Tracked files")
	tracked, err := linker.Status(cfg)
	if err != nil {
		errf("%v", err)
		return 1
	}
	if len(tracked) == 0 {
		info("nothing tracked yet — see: dotmgr add <path>")
	}
	for _, t := range tracked {
		fmt.Printf("  %-8s %s\n", t.State, t.Rel)
	}

	candidates, err := discover.Candidates(cfg)
	if err != nil {
		errf("%v", err)
		return 1
	}
	if len(candidates) > 0 {
		step("Untracked files a known tool owns")
		for _, c := range candidates {
			fmt.Println(" ", c)
		}
		info("track one with: dotmgr add <path>")
	}

	return 0
}
