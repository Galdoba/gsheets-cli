package main

import (
	"context"
	"fmt"
	"gsheets-cli/internal/application/commands"
	"gsheets-cli/internal/application/flags"
	"gsheets-cli/internal/infrastructure"
	"log"
	"os"

	"github.com/urfave/cli/v3"
)

func main() {
	inf, err := infrastructure.Initalize()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	app := cli.Command{
		Name:    infrastructure.AppName,
		Aliases: []string{},
		Commands: []*cli.Command{
			commands.Read(inf.Config),
			commands.Update(inf.Config),
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
