package main

import (
	"github.com/urfave/cli/v2"
)

func doInit(cCtx *cli.Context) error {
	helper, err := NewPrivateFolderHelper()
	if err != nil {
		return err
	}

	err = helper.Init()
	if err != nil {
		return err
	}

	return nil
}
