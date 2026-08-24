package commands

import (
	"context"
	"fmt"
	"os"

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

	data := map[string]string{
		dataCredFile:  credentialsPath,
		dataSheetID:   tableID,
		dataSheetName: sheetName,
	}
	return data, nil
}
