// generated from spec: zypper-declarative.spec.md sha256:58e1636e2de82ab81a5cd3f81d6b3c9ac6a8976e18f9abb2bbd2b2aba56fe4d4
//
// Command zypper-declarative is the entry point. It contains only CLI dispatch:
// it builds the argument list and calls into internal/cli, then exits with the
// returned code. All behaviour lives in the internal packages.
package main

import (
	"os"

	"github.com/mge1512/zypper-declarative/internal/cli"
)

func main() {
	os.Exit(cli.New().Run(os.Args[1:]))
}
