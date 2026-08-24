package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/Galdoba/gsheets-cli/internal/application"
	"github.com/Galdoba/gsheets-cli/internal/application/commands"
	"github.com/Galdoba/gsheets-cli/internal/application/flags"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure"
	"github.com/urfave/cli/v3"
)

func main() {
	inf, err := infrastructure.Initalize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	app := cli.Command{
		Name:    application.AppName,
		Aliases: []string{},
		Commands: []*cli.Command{
			commands.Read(inf.Config),
			commands.Fetch(inf.Config),
			commands.Update(inf.Config),
			commands.ProfileCommand(inf.Config),
			commands.TUI(inf.Config),
		},
		Flags: []cli.Flag{
			&flags.GlobalCredentials,
			&flags.GlobalSpreadsheet,
			&flags.GlobalTable,
		},
	}
	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
