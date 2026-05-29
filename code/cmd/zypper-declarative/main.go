// generated from spec: zypper-declarative.spec.md sha256:f8ff76ecbc4bbc69a49e2e32b2924da3a64df1ad46196e05ce8c137b684429b2
//
// Entry point for zypper-declarative. This file contains only CLI dispatch:
// it forwards the process arguments to internal/cli and exits with the returned
// code. All behaviour lives in the internal packages (per SOURCE-PARTITIONING:
// one-entry-one-implementation).
package main

import (
	"os"

	"github.com/mge1512/zypper-declarative/internal/cli"
)

func main() {
	app := cli.New()
	os.Exit(app.Run(os.Args[1:]))
}
