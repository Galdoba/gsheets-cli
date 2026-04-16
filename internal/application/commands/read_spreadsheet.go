package commands

import (
	"context"
	"fmt"
	"gsheets-cli/internal/application/flags"
	"gsheets-cli/internal/domain/sheet"
	"gsheets-cli/internal/infrastructure/config"
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

		outputFile := "/home/galdoba/go/src/github.com/Galdoba/gsheets-cli/cmd/gsheets-cli/output.json"
		fmt.Println("service created...")

		srv, err := getService(ctx, parameters[dataCredFile])
		if err != nil {
			return err
		}

		fields := "sheets(data(rowData(values(formattedValue,note))))"
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
		fmt.Println("updating...")

		sc := sheet.New(actualID, tableName)
		if len(resp.Sheets) > 0 { //TODO: later we must have ability to read multile tables in a row
			sc.UpdateGridData(resp.Sheets[0])
		} else {
			fmt.Println("⚠️  No data found in the specified sheet. (sheets)")
		}

		fmt.Println("saving...")
		if err := sc.SaveAs(outputFile); err != nil {
			return fmt.Errorf("failed to save table data: %w", err)
		}

		fmt.Printf("✅ Successfully saved %d rows to %s\n", sc.Rows, outputFile)
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
