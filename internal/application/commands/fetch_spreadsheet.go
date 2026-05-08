package commands

import (
	"context"
	"fmt"
	"gsheets-cli/internal/domain/sheet"
	"gsheets-cli/internal/infrastructure/config"
	persistience "gsheets-cli/internal/infrastructure/persistence"
	"gsheets-cli/internal/infrastructure/session"

	"github.com/urfave/cli/v3"
	"google.golang.org/api/googleapi"
)

func Fetch(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "fetch",
		Aliases: []string{"r"},
		Usage:   "Fetch cloud spreadsheet data and save to a local file",
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

		_, err = session.Restore()
		if err != nil {
			fmt.Printf("session cannot be restored: %v\n", err)
			fmt.Println("fallback to new state")
		}

		// commandError := errors.New("no action was taken")
		title := "spreadsheet"
		tableName := ""
		fetched := sheet.New("", "")
		fmt.Println("service created...")
		pullAtempt := 0
		for {
			pullAtempt++
			srv, err := getService(ctx, parameters[dataCredFile])
			if err != nil {
				return err
			}

			fields := "properties/title,sheets(data(rowData(values(formattedValue,note))))"
			actualID := extractSheetID(parameters[dataSheetID])
			tableName = parameters[dataSheetName]

			fmt.Printf("reading attempt %d...\r", pullAtempt)
			resp, err := srv.Spreadsheets.Get(actualID).
				Ranges(tableName).
				IncludeGridData(true).
				Fields(googleapi.Field(fields)).
				Do()
			if err != nil {
				fmt.Println("failed to read spreadsheet:", err)
				continue
			}
			if resp.Properties != nil {
				title = resp.Properties.Title
			}

			fmt.Println("updating...                      ")

			fetched = sheet.New(title, tableName)
			if len(resp.Sheets) > 0 {
				fetched.UpdateGridData(resp.Sheets[0])
			} else {
				fmt.Println("⚠️  No data found in the specified sheet.")
			}
			break
		}
		fmt.Println("loading storage...")
		store, err := persistience.NewData(title, tableName)
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
		// s.SetSheetName(fetched.SpreadsheetTitle)
		// s.SetTableName(fetched.SheetName)
		fmt.Println(fetched.Cells["B118"].Value)
		return nil
	}
}
