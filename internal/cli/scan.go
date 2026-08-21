package cli

import (
	"fmt"

	"github.com/snazzyjames/dotmgr/internal/discover"
)

func cmdScan(args []string) int {
	cfg, err := loadConfig()
	if err != nil {
		errf("%v", err)
		return 1
	}

	candidates, err := discover.Candidates(cfg)
	if err != nil {
		errf("%v", err)
		return 1
	}
	if len(candidates) == 0 {
		info("nothing new found")
		return 0
	}

	step("Untracked files a known tool owns")
	for _, c := range candidates {
		fmt.Println(" ", c)
	}
	info("track one with: dotmgr add <path>")
	return 0
}
