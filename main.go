package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// extractSheetID pulls the ID from a full Google Sheets URL or returns it as-is
func extractSheetID(input string) string {
	if idx := strings.Index(input, "/d/"); idx != -1 {
		rest := input[idx+3:]
		end := strings.Index(rest, "/")
		if end == -1 {
			end = len(rest)
		}
		return rest[:end]
	}
	return input
}

// getService initializes the Sheets API client with service account credentials
func getService(ctx context.Context, credFile string) (*sheets.Service, error) {
	data, err := os.ReadFile(credFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credential file: %w", err)
	}
	// cred := option.WithAuthCredentialsJSON(option.ImpersonatedServiceAccount, data)
	srv, err := sheets.NewService(ctx,
		// option.WithCredentialsFile(credFile),
		option.WithAuthCredentialsJSON(option.ImpersonatedServiceAccount, data),
		option.WithScopes(sheets.SpreadsheetsScope),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Google Sheets client: %w", err)
	}
	return srv, nil
}

// colToLetter converts 1-based column index to Excel-style A1 notation
func colToLetter(n int) string {
	if n <= 0 {
		return "A"
	}
	letter := ""
	for n > 0 {
		n--
		letter = string(rune('A'+n%26)) + letter
		n /= 26
	}
	return letter
}

// runRead handles the "read" command
func runRead(ctx context.Context, cmd *cli.Command) error {
	credFile := cmd.Root().String("credentials")
	sheetID := cmd.Root().String("spreadsheet")
	sheetName := cmd.Root().String("sheet")
	outputFile := cmd.String("output")

	srv, err := getService(ctx, credFile)
	if err != nil {
		return err
	}

	actualID := extractSheetID(sheetID)
	resp, err := srv.Spreadsheets.Values.Get(actualID, sheetName).Do()
	if err != nil {
		return fmt.Errorf("failed to read spreadsheet: %w", err)
	}

	if len(resp.Values) == 0 {
		fmt.Println("⚠️  No data found in the specified sheet.")
		return nil
	}

	// Ensure output directory exists
	dir := filepath.Dir(outputFile)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory: %w", err)
		}
	}

	file, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	for _, r := range resp.Values {
		strRow := make([]string, len(r))
		for i, v := range r {
			strRow[i] = fmt.Sprintf("%v", v)
		}
		if err := writer.Write(strRow); err != nil {
			return fmt.Errorf("failed to write CSV row: %w", err)
		}
	}

	fmt.Printf("✅ Successfully saved %d rows to %s\n", len(resp.Values), outputFile)
	return nil
}

// runUpdate handles the "update" command
func runUpdate(ctx context.Context, cmd *cli.Command) error {
	credFile := cmd.Root().String("credentials")
	sheetID := cmd.Root().String("spreadsheet")
	sheetName := cmd.Root().String("sheet")
	row := cmd.Int("row")
	col := cmd.Int("col")
	value := cmd.String("value")

	srv, err := getService(ctx, credFile)
	if err != nil {
		return err
	}

	actualID := extractSheetID(sheetID)
	cellRef := fmt.Sprintf("%s!%s%d", sheetName, colToLetter(col), row)

	rb := &sheets.ValueRange{
		Values: [][]interface{}{{value}},
	}

	_, err = srv.Spreadsheets.Values.Update(actualID, cellRef, rb).
		ValueInputOption("USER_ENTERED").
		Do()
	if err != nil {
		return fmt.Errorf("failed to update cell: %w", err)
	}

	fmt.Printf("✅ Successfully updated cell %s with value '%s'\n", cellRef, value)
	return nil
}

func main() {
	app := &cli.Command{
		Name:        "gsheets-cli",
		Usage:       "Cross-platform CLI for Google Sheets operations",
		HideVersion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "credentials",
				Aliases:  []string{"c"},
				Usage:    "Path to Google Service Account JSON file",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "spreadsheet",
				Aliases:  []string{"s"},
				Usage:    "Spreadsheet ID or full URL",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "sheet",
				Aliases:  []string{"n"},
				Usage:    "Sheet name (tab name)",
				Required: true,
			},
		},
		Commands: []*cli.Command{
			{
				Name:    "read",
				Aliases: []string{"r"},
				Usage:   "Read spreadsheet data and save to a CSV file",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "output",
						Aliases: []string{"o"},
						Usage:   "Path to save CSV file",
						Value:   "output.csv",
					},
				},
				Action: runRead,
			},
			{
				Name:    "update",
				Aliases: []string{"u"},
				Usage:   "Update a specific cell in the spreadsheet",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:    "row",
						Aliases: []string{"r"},
						Usage:   "Row number (1-based)",
						Value:   1,
					},
					&cli.IntFlag{
						Name:    "col",
						Aliases: []string{"k"},
						Usage:   "Column number (1-based)",
						Value:   1,
					},
					&cli.StringFlag{
						Name:     "value",
						Aliases:  []string{"v"},
						Usage:    "New cell value",
						Required: true,
					},
				},
				Action: runUpdate,
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
