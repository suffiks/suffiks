package wasi

import (
	"fmt"

	"github.com/suffiks/suffiks/internal/extension/oci"
	"github.com/urfave/cli/v2"
)

func download() *cli.Command {
	return &cli.Command{
		Name:        "download",
		Description: "Download a wasi file from a registry",
		ArgsUsage:   "A single argument is required: remote reference",
		Action: func(c *cli.Context) error {
			files, err := oci.Get(c.Context, "ghcr.io/suffiks/suffiks/test", "latest")
			if err != nil {
				return err
			}

			for f, b := range files {
				fmt.Println(f, len(b))
			}
			return nil
		},
	}
}
