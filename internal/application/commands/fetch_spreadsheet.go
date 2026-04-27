package commands

import (
	"context"
	"fmt"
	"gsheets-cli/internal/domain/sheet"
	"gsheets-cli/internal/infrastructure/config"
	"gsheets-cli/internal/infrastructure/storage"

	"github.com/urfave/cli/v3"
	"google.golang.org/api/googleapi"
)

func Fetch(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "read",
		Aliases: []string{"r"},
		Usage:   "Read spreadsheet data and save to a CSV file",
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

		fmt.Println("service created...")

		srv, err := getService(ctx, parameters[dataCredFile])
		if err != nil {
			return err
		}

		fields := "properties/title,sheets(data(rowData(values(formattedValue,note))))"
		actualID := extractSheetID(parameters[dataSheetID])
		tableName := parameters[dataSheetName]

		fmt.Println("reading...")

		resp, err := srv.Spreadsheets.Get(actualID).
			Ranges(tableName).
			IncludeGridData(true).
			Fields(googleapi.Field(fields)).
			Do()
		if err != nil {
			return fmt.Errorf("failed to read spreadsheet: %w", err)
		}
		title := "spreadsheet"
		if resp.Properties != nil {
			title = resp.Properties.Title
		}
		fmt.Println("updating...")

		fetched := sheet.New(title, tableName)
		if len(resp.Sheets) > 0 {
			fetched.UpdateGridData(resp.Sheets[0])
		} else {
			fmt.Println("⚠️  No data found in the specified sheet.")
		}

		fmt.Println("loading storage...")
		store, err := storage.New(title, tableName)
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
