package main

import (
	"os"

	"github.com/snazzyjames/dotmgr/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:]))
}
