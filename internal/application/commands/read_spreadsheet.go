package commands

import (
	"context"
	"fmt"
	"gsheets-cli/internal/application/flags"
	"gsheets-cli/internal/domain/cell"
	"gsheets-cli/internal/domain/render"
	"gsheets-cli/internal/domain/sheet"
	"gsheets-cli/internal/domain/view"
	"gsheets-cli/internal/infrastructure/config"
	"gsheets-cli/internal/infrastructure/storage"
	"strings"

	"github.com/urfave/cli/v3"
	"google.golang.org/api/googleapi"
)

func Read(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "read",
		Aliases: []string{"r"},
		Usage:   "Read spreadsheet data and save to a CSV file",
		Flags: []cli.Flag{
			&flags.ReadCmdOutput,
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
		preset := view.DefaultPreset()
		preset.Columns = make(map[int]view.ColumnConfig)
		for i := 0; i < 17; i++ {

			preset.Columns[i] = view.ColumnConfig{
				Index:          i,
				RenderPosition: i,
				Letter:         cell.ColIndexToLetter(i),
				Visibility:     view.ColVisible,
				WidthMode:      0,
				WidthValue:     0,
				AlignRight:     false,
				FormatHint:     "",
				NoteHint:       "",
				Frosen:         false,
				GroupID:        "",
			}
		}
		fmt.Println(render.Render(fetched, &preset).String())

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
