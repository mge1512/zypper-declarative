// generated from spec: zypper-declarative.spec.md sha256:51284526723dc9238113984023bfb9a596d55b534c8ea580dfac1157cd70dd03
//
// Command zypper-declarative is the entry point. It only builds the argument
// vector and dispatches into internal/cli; all behaviour lives in the internal
// packages (SOURCE-PARTITIONING: one-entry-one-implementation).
package main

import (
	"os"

	"github.com/mge1512/zypper-declarative/internal/cli"
)

func main() {
	app := &cli.App{Stdout: os.Stdout, Stderr: os.Stderr}
	os.Exit(app.Run(os.Args[1:]))
}
