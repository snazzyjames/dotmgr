package cli

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/snazzyjames/dotmgr/internal/config"
	"github.com/snazzyjames/dotmgr/internal/linker"
	"github.com/snazzyjames/dotmgr/internal/store"
)

// stdin is where confirmation prompts read from; tests replace it.
var stdin io.Reader = os.Stdin

func cmdUninstall(args []string) int {
	fs := flag.NewFlagSet("uninstall", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "show what would happen without changing anything")
	fs.BoolVar(dryRun, "n", false, "shorthand for --dry-run")
	restoreBackups := fs.Bool("restore-backups", false, "go back to what this machine had before dotmgr, where a backup exists, instead of keeping your current configs")
	yes := fs.Bool("yes", false, "don't ask for confirmation")
	fs.BoolVar(yes, "y", false, "shorthand for --yes")
	force := fs.Bool("force", false, "uninstall even if the local clone has unpushed changes")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	cfgPath, err := config.DefaultPath()
	if err != nil {
		errf("%v", err)
		return 1
	}
	cfg, err := loadConfig()
	if err != nil {
		errf("%v", err)
		return 1
	}

	if err := checkClonePath(cfg.Repo.Path); err != nil {
		errf("%v", err)
		return 1
	}

	if !*force {
		dirty, derr := store.Dirty(cfg.Repo.Path)
		unpushed, uerr := store.Unpushed(cfg.Repo.Path)
		if derr != nil || uerr != nil {
			errf("couldn't check %s for unpushed changes; run dotmgr push first, or re-run with --force", cfg.Repo.Path)
			return 1
		}
		if dirty || unpushed {
			errf("the local clone has changes that aren't on your remote yet; run dotmgr push first, or re-run with --force to discard them")
			return 1
		}
	}

	step("Uninstalling dotmgr")
	if *dryRun {
		info("dry run: no changes will be made")
	}

	// Preview first, so the confirmation prompt can say what will happen.
	plan, err := linker.Uninstall(cfg, *restoreBackups, true)
	if err != nil {
		errf("%v", err)
		return 1
	}
	printUninstallPlan(plan, cfg, cfgPath)

	if *dryRun {
		fmt.Println()
		fmt.Println("Dry run complete: no changes made.")
		return 0
	}

	if !*yes && !confirm("Continue?") {
		fmt.Println("Aborted: nothing changed.")
		return 1
	}

	res, err := linker.Uninstall(cfg, *restoreBackups, false)
	if err != nil {
		errf("%v", err)
		return 1
	}
	if res.Failed > 0 {
		step("Some files couldn't be restored")
		for _, e := range res.Errors {
			fmt.Println("    -", e)
		}
		fmt.Println()
		fmt.Println("Left the local clone and config in place so nothing is lost.")
		fmt.Println("Fix the errors above and re-run: dotmgr uninstall")
		return 1
	}

	step("Removing dotmgr's local data")
	if err := os.RemoveAll(cfg.Repo.Path); err != nil {
		errf("removing %s: %v", cfg.Repo.Path, err)
		return 1
	}
	info("deleted local clone: %s", tilde(cfg.Repo.Path))
	if err := os.Remove(cfgPath); err != nil && !os.IsNotExist(err) {
		errf("removing %s: %v", cfgPath, err)
		return 1
	}
	info("deleted config: %s", tilde(cfgPath))
	// Tidy up the dotmgr/ directories that held them, but only if empty.
	os.Remove(filepath.Dir(cfg.Repo.Path))
	os.Remove(filepath.Dir(cfgPath))

	step("Done")
	if len(res.Backups) > 0 {
		fmt.Println("  These backups are still in place; delete or restore them yourself:")
		for _, b := range res.Backups {
			fmt.Println("    -", tilde(b))
		}
	}
	fmt.Printf("  Your remote repo (%s) was not touched.\n", cfg.Repo.Remote)
	if exe, err := os.Executable(); err == nil {
		fmt.Printf("  To remove the dotmgr binary itself: rm %s\n", exe)
	}
	return 0
}

func printUninstallPlan(plan *linker.UninstallResult, cfg *config.Config, cfgPath string) {
	// A file whose backup is restored ends up as the backup, not the current
	// version, so list it only under the backups heading.
	fromBackup := map[string]bool{}
	for _, b := range plan.BackupsRestored {
		fromBackup[backupOriginal(b)] = true
	}
	var keep []string
	for _, rel := range plan.Restored {
		if !fromBackup[tilde(filepath.Join(homeDir(), rel))] {
			keep = append(keep, rel)
		}
	}

	if len(keep) > 0 {
		fmt.Println("  Keep your current configs (each symlink becomes a regular file):")
		for _, rel := range keep {
			fmt.Println("    ~/" + rel)
		}
	}
	if len(plan.BackupsRestored) > 0 {
		fmt.Println("  Go back to what this machine had before dotmgr:")
		for _, b := range plan.BackupsRestored {
			fmt.Printf("    %s  (from %s)\n", backupOriginal(b), filepath.Base(b))
		}
	}
	if len(plan.Backups) > 0 {
		fmt.Println("  Leave these pre-dotmgr backups in place (--restore-backups puts the oldest back instead):")
		for _, b := range plan.Backups {
			fmt.Println("    " + tilde(b))
		}
	}
	if len(plan.Skipped) > 0 {
		fmt.Println("  Leave alone (not linked by dotmgr on this machine):")
		for _, rel := range plan.Skipped {
			fmt.Println("    ~/" + rel)
		}
	}
	fmt.Println("  Delete the local clone: " + tilde(cfg.Repo.Path))
	fmt.Println("  Delete the config:      " + tilde(cfgPath))
	fmt.Println("  Your remote repo is not touched.")
}

// checkClonePath guards the RemoveAll in cmdUninstall against a config whose
// repo path was edited to point somewhere that isn't a dotmgr clone.
func checkClonePath(path string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	clean := filepath.Clean(path)
	if path == "" || !filepath.IsAbs(clean) || clean == filepath.Dir(clean) || clean == filepath.Clean(home) {
		return fmt.Errorf("refusing to uninstall: repo path %q in the config doesn't look like a dotmgr clone", path)
	}
	if _, err := os.Stat(filepath.Join(clean, ".git")); err != nil {
		return fmt.Errorf("refusing to uninstall: %s is not a git clone", clean)
	}
	return nil
}

// tilde shortens a path under $HOME to ~/... for display.
func tilde(path string) string {
	home := homeDir()
	if home == "" {
		return path
	}
	if rel, err := filepath.Rel(home, path); err == nil && !strings.HasPrefix(rel, "..") {
		return filepath.Join("~", rel)
	}
	return path
}

func homeDir() string {
	home, _ := os.UserHomeDir()
	return home
}

// backupOriginal returns the ~/... path a <path>.bak.<timestamp> backup was
// made from.
func backupOriginal(backup string) string {
	if i := strings.LastIndex(backup, ".bak."); i >= 0 {
		backup = backup[:i]
	}
	return tilde(backup)
}

func confirm(prompt string) bool {
	fmt.Printf("\n%s [y/N] ", prompt)
	line, _ := bufio.NewReader(stdin).ReadString('\n')
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes"
}
