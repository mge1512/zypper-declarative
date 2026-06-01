// generated from spec: zypper-declarative.spec.md sha256:b2d0de88fbed1163678e59e931c741b9d999b71f902f6eb01db8790bb813d057
//
// Entry point for zypper-declarative. Thin CLI dispatch only: build args and
// call into internal/cli. No behaviour is implemented here (per
// SOURCE-PARTITIONING: one-entry-one-implementation).
package main

import (
	"os"

	"github.com/mge1512/zypper-declarative/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
