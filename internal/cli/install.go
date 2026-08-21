package cli

import (
	"flag"
	"fmt"

	"github.com/snazzyjames/dotmgr/internal/linker"
)

func cmdInstall(args []string) int {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "show what would happen without changing anything")
	fs.BoolVar(dryRun, "n", false, "shorthand for --dry-run")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfg, err := loadConfig()
	if err != nil {
		errf("%v", err)
		return 1
	}

	step("Installing dotfiles")
	if *dryRun {
		info("dry run — no changes will be made")
	}

	res, err := linker.Install(cfg, *dryRun)
	if err != nil {
		errf("%v", err)
		return 1
	}

	step("Summary")
	fmt.Printf("  Linked  : %d\n", res.Linked)
	fmt.Printf("  Skipped : %d (already linked)\n", res.Skipped)
	if res.BackedUp > 0 {
		fmt.Printf("  Backed up existing files: %d\n", res.BackedUp)
	}
	if res.Failed > 0 {
		fmt.Printf("  Failed  : %d\n", res.Failed)
		for _, e := range res.Errors {
			fmt.Println("    -", e)
		}
		return 1
	}

	fmt.Println()
	if *dryRun {
		fmt.Println("Dry run complete — no changes made.")
	} else {
		fmt.Println("Done.")
	}
	return 0
}
