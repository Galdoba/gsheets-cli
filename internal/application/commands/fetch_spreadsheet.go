package commands

import (
	"context"
	"fmt"

	"github.com/Galdoba/gsheets-cli/internal/domain/sheet"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/config"
	persistience "github.com/Galdoba/gsheets-cli/internal/infrastructure/persistence"
	"github.com/urfave/cli/v3"
	"google.golang.org/api/googleapi"
)

func Fetch(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "fetch",
		Aliases: []string{"f", "sync"},
		Usage:   "Fetch remote spreadsheet data and save to local storage",
		Flags:   []cli.Flag{},
		Action:  fetchAction(cfg),
	}
}

func fetchAction(cfg config.Config) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		parameters, err := getSpreadsheetData(cmd, cfg)
		if err != nil {
			return fmt.Errorf("failed to collect spreadsheet data: %w", err)
		}

		fmt.Println("initializing service...")

		srv, err := getService(ctx, parameters[dataCredFile])
		if err != nil {
			return err
		}

		fields := "properties/title,sheets(data(rowData(values(formattedValue,note))))"
		actualID := extractSheetID(parameters[dataSheetID])
		tableName := parameters[dataSheetName]

		fmt.Println("reading remote data...")

		resp, err := srv.Spreadsheets.Get(actualID).
			Ranges(tableName).
			IncludeGridData(true).
			Fields(googleapi.Field(fields)).
			Do()
		if err != nil {
			return fmt.Errorf("failed to read spreadsheet: %w", err)
		}

		// We use the actualID as the storage title to guarantee consistency
		// between fetch and read commands, avoiding mismatched Google Sheet titles.
		fmt.Println("updating local cache...")

		fetched := sheet.New(actualID, tableName)
		if len(resp.Sheets) > 0 {
			fetched.UpdateGridData(resp.Sheets[0])
		} else {
			fmt.Println("⚠️  No data found in the specified sheet.")
		}

		fmt.Println("loading storage...")
		store, err := persistience.NewData(actualID, tableName)
		if err != nil {
			return fmt.Errorf("failed to initialize storage: %w", err)
		}

		fmt.Println("merging data...")
		if err := store.Merge(fetched); err != nil {
			return fmt.Errorf("failed to merge data: %w", err)
		}

		fmt.Println("saving...")
		if err := store.Save(); err != nil {
			return fmt.Errorf("failed to save storage: %w", err)
		}

		fmt.Printf("✅ Successfully synced %d rows to local storage\n", fetched.Rows)

		return nil
	}
}
