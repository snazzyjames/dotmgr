package cli

import (
	"fmt"
	"path/filepath"

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
		info("nothing tracked yet, see: dotmgr add <path>")
	}
	seen := map[string]bool{}
	for _, t := range tracked {
		name := "~/" + filepath.ToSlash(t.Rel)
		if t.Dir {
			name += "/  (linked as a whole)"
		}
		fmt.Printf("  %-9s %s\n", t.State, name)
		seen[t.State] = true
	}
	printStatusHints(seen)

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

// printStatusHints explains each problem state that appeared, once.
func printStatusHints(seen map[string]bool) {
	hints := []struct{ state, text string }{
		{linker.StateMissing, "missing: not on this machine yet. Fix: dotmgr install"},
		{linker.StateUnlinked, "unlinked: a regular file with the same content replaced the link, usually a program\n" +
			"    rewriting the file. Nothing is lost. Fix: dotmgr relink <path>"},
		{linker.StateDiverged, "diverged: a regular file with different content replaced the link, usually a program\n" +
			"    saving over it, so those changes aren't in the repo. Keep this machine's version:\n" +
			"    dotmgr relink <path>. Keep the repo's version instead: dotmgr install (backs this one up)"},
		{linker.StateConflict, "conflict: something other than a file is in the way. Fix: move it aside, then dotmgr install\n" +
			"    (or just dotmgr install, which backs it up)"},
	}
	var shown bool
	for _, h := range hints {
		if !seen[h.state] {
			continue
		}
		if !shown {
			fmt.Println()
			shown = true
		}
		fmt.Println("  " + h.text)
	}
}
