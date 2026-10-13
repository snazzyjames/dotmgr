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
	linkDir := fs.Bool("link-dir", false, "track a directory as a whole with one symlink")
	fs.Usage = func() {
		fmt.Println("Usage: dotmgr add [--force] <file-or-directory>\n       dotmgr add --link-dir <directory>")
	}
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

	if *linkDir {
		if *force {
			errf("--force can't be combined with --link-dir")
			return 1
		}
		res, err := linker.LinkDir(cfg, fs.Arg(0))
		if err != nil {
			errf("%v", err)
			return 1
		}
		step("Linked ~/%s as a whole", filepath.ToSlash(res.Rel))
		info("moved into %s", tilde(filepath.Join(cfg.Repo.Path, "files", res.Rel)))
		info("linked back to ~/%s", filepath.ToSlash(res.Rel))
		if len(res.Converted) > 0 {
			info("%d file(s) inside were tracked one by one; they're now part of the directory", len(res.Converted))
		}
		if len(res.Ignored) > 0 {
			info("kept out of git (matching .dotmgrignore):")
			for _, rel := range res.Ignored {
				fmt.Println("      ~/" + filepath.ToSlash(rel))
			}
		}
		fmt.Println()
		fmt.Println("New files in this directory are tracked automatically. Don't forget: dotmgr push")
		return 0
	}

	if linker.IsDir(fs.Arg(0)) {
		if *force {
			errf("--force only works on a single file; add a denylisted file inside a directory by its own path")
			return 1
		}
		res, err := linker.AddDir(cfg, fs.Arg(0))
		if err != nil {
			errf("%v", err)
			return 1
		}
		return printDirResult(res, "Tracked", "already tracked", "Don't forget: dotmgr push")
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

// printDirResult reports what AddDir or RemoveDir did, one line per file.
func printDirResult(res *linker.DirResult, verb, alreadyLabel, reminder string) int {
	if len(res.Done) > 0 {
		step("%s %d file(s)", verb, len(res.Done))
		for _, rel := range res.Done {
			fmt.Println("    ~/" + rel)
		}
	}
	if len(res.Already) > 0 {
		step("Skipped %d file(s): %s", len(res.Already), alreadyLabel)
	}
	if len(res.Ignored) > 0 {
		step("Skipped %d file(s) matching .dotmgrignore", len(res.Ignored))
		for _, rel := range res.Ignored {
			fmt.Println("    ~/" + rel)
		}
	}
	if len(res.Denied) > 0 {
		step("Skipped %d file(s) matching the secrets denylist", len(res.Denied))
		for _, rel := range res.Denied {
			fmt.Println("    ~/" + rel)
		}
		info("to track one anyway: dotmgr add --force <that file>")
	}
	if len(res.Errors) > 0 {
		step("Failed")
		for _, e := range res.Errors {
			fmt.Println("    -", e)
		}
		return 1
	}
	if len(res.Done) == 0 {
		fmt.Println()
		fmt.Println("Nothing to do.")
		return 0
	}
	fmt.Println()
	fmt.Println(reminder)
	return 0
}
