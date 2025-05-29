package main

import (
	"github.com/urfave/cli/v2"
	"log/slog"
	"os"
)

var (
	version string

	app = &cli.App{
		Name:    "private-folder",
		Usage:   "private folder",
		Version: version,
		Commands: []*cli.Command{
			{
				Name:   "init",
				Usage:  "init",
				Action: doInit,
			},
		},
	}
)

func main() {
	err := app.Run(os.Args)
	if err != nil {
		slog.Error("error",
			slog.Any("err", err),
		)
	}
}
