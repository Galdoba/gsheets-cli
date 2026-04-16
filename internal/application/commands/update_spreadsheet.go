package commands

import (
	"context"
	"fmt"
	"gsheets-cli/internal/infrastructure/config"

	"github.com/urfave/cli/v3"
	"google.golang.org/api/sheets/v4"
)

func Update(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "update",
		Aliases: []string{"u"},
		Usage:   "Update a specific cell in the spreadsheet",
		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:    "row",
				Aliases: []string{"r"},
				Usage:   "Row number (1-based)",
				Value:   1,
			},
			&cli.IntFlag{
				Name:    "col",
				Aliases: []string{"k"},
				Usage:   "Column number (1-based)",
				Value:   1,
			},
			&cli.StringFlag{
				Name:     "value",
				Aliases:  []string{"v"},
				Usage:    "New cell value",
				Required: true,
			},
		},
		Action: updateAction(cfg),
	}
}

func updateAction(cfg config.Config) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		parameters, err := getSpreadsheetData(cmd, cfg)
		if err != nil {
			return fmt.Errorf("failed to collect spreadsheet data: %w", err)
		}
		credFile := parameters[dataCredFile]
		sheetID := parameters[dataSheetID]
		sheetName := parameters[dataSheetName]
		row := cmd.Int("row")
		col := cmd.Int("col")
		value := cmd.String("value")

		srv, err := getService(ctx, credFile)
		if err != nil {
			return err
		}

		actualID := extractSheetID(sheetID)
		cellRef := fmt.Sprintf("%s!%s%d", sheetName, colToLetter(col), row)

		rb := &sheets.ValueRange{
			Values: [][]any{{value}},
		}

		_, err = srv.Spreadsheets.Values.Update(actualID, cellRef, rb).
			ValueInputOption("USER_ENTERED").
			Do()
		if err != nil {
			return fmt.Errorf("failed to update cell: %w", err)
		}

		fmt.Printf("✅ Successfully updated cell %s with value '%s'\n", cellRef, value)
		return nil

	}
}
