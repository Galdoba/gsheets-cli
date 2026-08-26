package commands

import (
	"context"
	"fmt"

	"github.com/Galdoba/gsheets-cli/internal/infrastructure/config"
	"github.com/Galdoba/gsheets-cli/internal/service"
	"github.com/urfave/cli/v3"
)

func Edit(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "edit",
		Aliases: []string{"e"},
		Usage:   "Apply changes to local cache only (no network)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "updates",
				Aliases:  []string{"U"},
				Usage:    "Semicolon-separated list of cell updates: Cell=field=value, e.g. B6798=value=val1;B6799=note=val2",
				Required: true,
			},
		},
		Action: editAction(cfg),
	}
}

func editAction(cfg config.Config) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		parameters, err := getSpreadsheetData(cmd, cfg)
		if err != nil {
			return fmt.Errorf("failed to collect spreadsheet data: %w", err)
		}

		params := service.SpreadsheetServiceParams{
			SpreadsheetID:  parameters[dataSheetID],
			SheetName:      parameters[dataSheetName],
			CredentialFile: parameters[dataCredFile],
		}

		updatesStr := cmd.String("updates")
		changes, err := parseUpdates(updatesStr)
		if err != nil {
			return err
		}

		svc := service.NewSpreadsheetService()
		if err := svc.ApplyLocalChanges(params, changes...); err != nil {
			return err
		}

		fmt.Printf("✅ Applied %d change(s) to local cache\n", len(changes))
		return nil
	}
}
