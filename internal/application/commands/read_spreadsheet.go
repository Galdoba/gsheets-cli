package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/Galdoba/gsheets-cli/internal/domain/render"
	"github.com/Galdoba/gsheets-cli/internal/domain/view"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/config"
	persistience "github.com/Galdoba/gsheets-cli/internal/infrastructure/persistence"
	"github.com/urfave/cli/v3"
)

func Read(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "read",
		Aliases: []string{"r"},
		Usage:   "Read locally cached spreadsheet data and print to terminal",
		Action:  readAction(cfg),
	}
}

func readAction(cfg config.Config) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		parameters, err := getSpreadsheetData(cmd, cfg)
		if err != nil {
			return fmt.Errorf("failed to collect spreadsheet data: %w", err)
		}

		// Use the extracted Sheet ID to guarantee consistency with fetchAction
		spreadsheetID := extractSheetID(parameters[dataSheetID])
		tableName := parameters[dataSheetName]

		dataStore, err := persistience.NewData(spreadsheetID, tableName)
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		sc, err := dataStore.Load()
		if err != nil {
			return fmt.Errorf("failed to load data: %w", err)
		}

		// If no data has been fetched yet, the cache will be empty
		if sc.RowCount() == 0 {
			fmt.Println("⚠️  No local data found. Please run the 'fetch' command first.")
			return nil
		}

		sc.UpdateDimentions()
		fmt.Println("Using local cached data…")

		// 1. Build the domain preset
		preset := view.NewDefault(sc.ColCount())

		// 2. Translate Domain -> Render using the Builder
		renderCfg := view.BuildRenderConfig(sc, &preset, nil)

		// 3. Render all rows for CLI output (viewportStart=0, viewportHeight=RowCount)
		output := render.Render(sc, renderCfg, 0, sc.RowCount())
		fmt.Print(output)

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
