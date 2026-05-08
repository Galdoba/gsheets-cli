package commands

import (
	"context"
	"fmt"
	"gsheets-cli/internal/infrastructure/config"
	persistience "gsheets-cli/internal/infrastructure/persistence"
	"strings"

	"github.com/urfave/cli/v3"
)

func Read(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "read",
		Aliases: []string{"r"},
		Usage:   "Read spreadsheet data and save to a CSV file",
		Action:  readAction(cfg),
	}
}

func readAction(cfg config.Config) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		parameters, err := getSpreadsheetData(cmd, cfg)
		if err != nil {
			return fmt.Errorf("failed to collect spreadsheet data: %w", err)
		}

		dataStore, err := persistience.NewData(parameters[dataSheetName], parameters[dataLastTableName])
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}
		sc, err := dataStore.Load()
		if err != nil {
			return fmt.Errorf("failed to load data: %w", err)
		}
		if err == nil {
			sc.UpdateDimentions()
			fmt.Println("Using local cached data…")
		}
		row := sc.GetRow(5000)
		fmt.Println(row)

		return nil
	}
}

func extractSheetIdAndName(cfg config.Config) (string, string) {
	lastUsed := cfg.Sheets.LastUsedTable
	data := strings.Split(lastUsed, "::")
	if len(data) != 2 {
		return "", ""
	}
	key := data[0]
	name := data[1]
	address := cfg.Sheets.Tables[key].Address
	return address, name
}
