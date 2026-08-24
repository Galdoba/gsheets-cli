package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Galdoba/gsheets-cli/internal/infrastructure/config"
	"github.com/urfave/cli/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

const (
	dataCredFile            = "credFile"
	dataSheetID             = "sheetID"
	dataSheetName           = "sheetName"
	dataSpreadsheetheetName = "spreadsheetName"
	dataLastSheetName       = "lastSpreadsheetName"
	dataLastTableName       = "lastTableName"
)

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

func getPrioritized(sources ...string) string {
	s := ""
	for _, value := range sources {
		if value == "" {
			continue
		}
		s = value
		break
	}
	return s
}

func getSpreadsheetData(cmd *cli.Command, cfg config.Config) (map[string]string, error) {
	// credFile := getPrioritized(
	// 	cmd.Root().String(flags.Credentials),
	// 	cfg.Credentials.ServiceAccounts[cfg.Credentials.ActiveAccount],
	// )
	// if credFile == "" {
	// 	return nil, fmt.Errorf("no service account credential file provided")
	// }

	// lastSheetID, lastTableName := extractSheetIdAndName(cfg)
	// fmt.Printf("lastSheetID: %q\n", lastSheetID)

	// sheetID := getPrioritized(
	// 	cmd.Root().String(flags.Spreadsheet),
	// 	lastSheetID,
	// )
	// if sheetID == "" {
	// 	return nil, fmt.Errorf("no sheet ID provided")
	// }

	// sheetName := getPrioritized(
	// 	cmd.Root().String(flags.Sheet),
	// 	lastTableName,
	// )
	// if sheetName == "" {
	// 	return nil, fmt.Errorf("no sheet name provided")
	// }

	// data := make(map[string]string, 3)
	// data[dataCredFile] = credFile
	// data[dataSheetID] = sheetID
	// data[dataSheetName] = sheetName
	// lastSheet, lastTable, _ := cfg.LastUsedTable()
	// data[dataLastSheetName] = lastSheet
	// data[dataLastTableName] = lastTable
	credentialsPath := cfg.Credentials.ServiceAccounts[cfg.Credentials.ActiveAccount]
	if c := cmd.String("credentials"); c != "" {
		credentialsPath = c
	}
	if credentialsPath == "" {
		return nil, fmt.Errorf("credentials file not passed")
	}

	tableID := cfg.Sheets.LastUsed.TableID
	if s := cmd.String("spreadsheet"); s != "" {
		tableID = s
	}
	if tableID == "" {
		return nil, fmt.Errorf("table id not passed")
	}

	sheetName := cfg.Sheets.LastUsed.SheetName
	if n := cmd.String("table"); n != "" {
		sheetName = n
	}
	if sheetName == "" {
		return nil, fmt.Errorf("sheet name not passed")
	}

	data := make(map[string]string, 3)
	data[dataCredFile] = credentialsPath
	data[dataSheetID] = tableID
	data[dataSheetName] = sheetName
	return data, nil
}
