package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/Galdoba/gsheets-cli/internal/domain/profile"
	"github.com/Galdoba/gsheets-cli/internal/domain/render"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/config"
	persistience "github.com/Galdoba/gsheets-cli/internal/infrastructure/persistence"
	"github.com/urfave/cli/v3"
)

func Read(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "read",
		Aliases: []string{"r"},
		Usage:   "Read locally cached spreadsheet data and print to terminal",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "profile",
				Aliases: []string{"p"},
				Usage:   "ID of the rendering profile to use",
				Value:   "default",
			},
		},
		Action: readAction(cfg),
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
		profileID := cmd.String("profile")

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

		// 1. Load Profile from Store
		profileStore := persistience.NewProfiles()
		p, err := profileStore.Get(spreadsheetID, tableName, profileID)
		if err != nil {
			// Fallback to a default profile if not found on disk
			p = &profile.Profile{
				ID:        profileID,
				Name:      "Default",
				SheetID:   spreadsheetID,
				TableName: tableName,
				Layout:    profile.LayoutConfig{}, // Empty layout triggers "show all" fallback in builder
			}
		}

		// 2. Translate Domain Profile -> Render Config
		renderCfg, err := profile.BuildRenderConfig(sc, p)
		if err != nil {
			return fmt.Errorf("failed to build render config: %w", err)
		}

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
