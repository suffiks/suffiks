package wasi

import (
	"github.com/urfave/cli/v2"
)

func New() *cli.Command {
	return &cli.Command{
		Name: "wasi",
		Subcommands: []*cli.Command{
			publish(),
			download(),
			testCmd(),
		},
	}
}
