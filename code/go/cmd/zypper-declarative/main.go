// generated from spec: zypper-declarative.spec.md sha256:27aee8e374eb3507189bad0b78339109d0116e6a13b55ae4df2ba9a18e769fc4
//
// Command zypper-declarative is the CLI entry point. It contains only dispatch:
// it forwards the process arguments and standard streams to internal/cli.Run and
// exits with the returned code. All behaviour lives in the internal packages.
package main

import (
	"os"

	"github.com/mge1512/zypper-declarative/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
