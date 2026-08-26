package commands

import (
	"context"
	"fmt"

	"github.com/Galdoba/gsheets-cli/internal/infrastructure/config"
	"github.com/Galdoba/gsheets-cli/internal/service"
	"github.com/urfave/cli/v3"
)

func Sync(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "sync",
		Aliases: []string{"s"},
		Usage:   "Synchronize local changes with remote spreadsheet",
		Action:  syncAction(cfg),
	}
}

func syncAction(cfg config.Config) cli.ActionFunc {
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
		err = svc.Sync(ctx, params)
		if err != nil {
			if syncErr, ok := err.(*service.SyncError); ok {
				fmt.Println("⚠️  Safeguard prevents sync due to conflicts or ambiguities:")
				fmt.Print(syncErr.Error())
				return nil
			}
			return err
		}

		fmt.Println("✅ Synchronization successful")
		actualID := parameters[dataSheetID]
		if err := config.UpdateLastUsed(actualID, parameters[dataSheetName]); err != nil {
			return fmt.Errorf("failed to update last used table: %w", err)
		}
		return nil
	}
}
