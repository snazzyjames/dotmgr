package cli

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/snazzyjames/dotmgr/internal/selfupdate"
)

// Swapped out in tests so they don't compile dotmgr or touch the real binary.
var (
	buildBinary   = selfupdate.Build
	installedPath = selfupdate.Installed
)

func cmdUpdate(args []string) int {
	fs := flag.NewFlagSet("update", flag.ContinueOnError)
	version := fs.String("version", "latest", "published version to build, e.g. v1.2.0")
	from := fs.String("from", "", "build from this local dotmgr checkout instead")
	fs.Usage = func() { fmt.Println("Usage: dotmgr update [--version <version> | --from <checkout>]") }
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		errf("update takes no arguments")
		fs.Usage()
		return 1
	}
	versionSet := false
	fs.Visit(func(f *flag.Flag) { versionSet = versionSet || f.Name == "version" })
	if versionSet && *from != "" {
		errf("use either --version or --from, not both")
		return 1
	}

	src := selfupdate.Source{Version: *version}
	if *from != "" {
		dir, err := filepath.Abs(*from)
		if err != nil {
			errf("%v", err)
			return 1
		}
		src = selfupdate.Source{Dir: dir}
	}

	exe, err := installedPath()
	if err != nil {
		errf("finding the installed dotmgr: %v", err)
		return 1
	}

	step("Updating dotmgr")
	info("installed: %s (%s)", exe, selfupdate.Version())
	info("building: %s", src)

	buildDir, err := os.MkdirTemp("", "dotmgr-update-")
	if err != nil {
		errf("%v", err)
		return 1
	}
	newBin, err := buildBinary(src, buildDir, os.Stderr)
	if err != nil {
		os.RemoveAll(buildDir)
		errf("%v", err)
		info("the installed dotmgr wasn't changed")
		return 1
	}

	if err := selfupdate.Replace(newBin, exe); err != nil {
		if errors.Is(err, selfupdate.ErrPermission) {
			// Keep the build so only the copy needs elevated rights; running
			// all of "sudo dotmgr update" often can't find go on sudo's PATH.
			errf("%v", err)
			fmt.Println()
			fmt.Println("The new version is built. Install it with:")
			fmt.Printf("  sudo install -m 0755 %s %s\n", newBin, exe)
			return 1
		}
		os.RemoveAll(buildDir)
		errf("replacing %s: %v", exe, err)
		info("the installed dotmgr wasn't changed")
		return 1
	}
	os.RemoveAll(buildDir)

	step("Done")
	info("now: %s", installedVersion(exe))
	return 0
}

// installedVersion asks the binary at exe for its version, since the running
// process still reports the version it was built as.
func installedVersion(exe string) string {
	out, err := exec.Command(exe, "version").Output()
	if err != nil {
		return "a version from before dotmgr version existed"
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(string(out)), "dotmgr "))
}

func cmdVersion(args []string) int {
	fmt.Println("dotmgr " + selfupdate.Version())
	return 0
}
