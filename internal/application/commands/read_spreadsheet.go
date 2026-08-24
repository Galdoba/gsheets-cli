package commands

import (
	"context"
	"fmt"

	"github.com/Galdoba/gsheets-cli/internal/domain/profile"
	"github.com/Galdoba/gsheets-cli/internal/domain/render"
	"github.com/Galdoba/gsheets-cli/internal/domain/sheet"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/config"
	persistience "github.com/Galdoba/gsheets-cli/internal/infrastructure/persistence"
	"github.com/Galdoba/gsheets-cli/internal/service"
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
				Usage:   "Name of the rendering profile to use (defaults to table binding or 'default')",
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

		params := service.SpreadsheetServiceParams{
			SpreadsheetID:  parameters[dataSheetID],
			SheetName:      parameters[dataSheetName],
			CredentialFile: parameters[dataCredFile],
		}

		svc := service.NewSpreadsheetService()

		// Load local cache
		sc, err := svc.LoadCache(params)
		if err != nil {
			return err
		}

		if sc.RowCount() == 0 {
			fmt.Println("⚠️  No local data found. Please run the 'fetch' command first.")
			return nil
		}

		sc.UpdateDimentions()
		fmt.Println("Using local cached data…")

		// Resolve profile name
		spreadsheetID := sheet.ExtractSheetID(parameters[dataSheetID]) // using existing helper in shared.go
		tableKey := spreadsheetID + "---" + parameters[dataSheetName]
		profileName := cmd.String("profile")
		if profileName == "" {
			profileName = cfg.ResolveProfileForTable(tableKey)
		}

		profileStore := persistience.NewProfiles()
		p, err := profileStore.Get(profileName)
		if err != nil {
			// Fallback to built-in default if the resolved profile doesn't exist.
			p = profile.DefaultProfile()
			p.Name = profileName
		}

		// Build render config and render
		renderCfg, err := profile.BuildRenderConfig(sc, p)
		if err != nil {
			return fmt.Errorf("failed to build render config: %w", err)
		}

		output := render.Render(sc, renderCfg, 0, sc.RowCount())
		fmt.Print(output)

		// TODO: Add optional fetch step here if flags/conditions allow.
		// e.g., if --fetch flag is set, call svc.Fetch then svc.SetCache then reload cache.

		return nil
	}
}
