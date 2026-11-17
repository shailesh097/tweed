// Command tweed sets up a Fedora installation from a declarative config file.
package main

import (
	"os"

	"github.com/shailesh097/tweed/internal/cli"
)

// version is set at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	os.Exit(cli.Run(os.Args[1:], version))
}
