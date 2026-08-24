package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Galdoba/gsheets-cli/internal/domain/cell"
	"github.com/Galdoba/gsheets-cli/internal/domain/sheet"
	persistience "github.com/Galdoba/gsheets-cli/internal/infrastructure/persistence"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"
)

// SpreadsheetServiceParams identifies a specific table and the credentials to access it.
type SpreadsheetServiceParams struct {
	SpreadsheetID  string
	SheetName      string
	CredentialFile string
}

// CellChange represents a single cell update (either value or note).
type CellChange struct {
	Row   int    // 1-based row index
	Col   int    // 1-based column index
	Field string // "value" (default) or "note"
	Value string
}

// SpreadsheetService provides methods for fetching, caching, and updating Google Sheets data.
type SpreadsheetService struct{}

// NewSpreadsheetService creates a new SpreadsheetService.
func NewSpreadsheetService() *SpreadsheetService {
	return &SpreadsheetService{}
}

// Fetch retrieves remote spreadsheet data and returns it as a fresh SheetCache.
// It does not touch local storage.
func (s *SpreadsheetService) Fetch(ctx context.Context, p SpreadsheetServiceParams) (*sheet.SheetCache, error) {
	if p.CredentialFile == "" {
		return nil, fmt.Errorf("credential file is required")
	}
	if p.SpreadsheetID == "" || p.SheetName == "" {
		return nil, fmt.Errorf("spreadsheet ID and sheet name are required")
	}

	srv, err := newSheetsService(ctx, p.CredentialFile)
	if err != nil {
		return nil, err
	}

	actualID := extractSheetID(p.SpreadsheetID)
	fields := "properties/title,sheets(data(rowData(values(formattedValue,note))))"

	resp, err := srv.Spreadsheets.Get(actualID).
		Ranges(p.SheetName).
		IncludeGridData(true).
		Fields(googleapi.Field(fields)).
		Do()
	if err != nil {
		return nil, fmt.Errorf("failed to read spreadsheet: %w", err)
	}

	fetched := sheet.New(actualID, p.SheetName)
	if len(resp.Sheets) > 0 {
		fetched.UpdateGridData(resp.Sheets[0])
	} else {
		return nil, fmt.Errorf("no data found in the specified sheet")
	}

	return fetched, nil
}

// LoadCache reads the locally cached data for the given table.
func (s *SpreadsheetService) LoadCache(p SpreadsheetServiceParams) (*sheet.SheetCache, error) {
	if p.SpreadsheetID == "" || p.SheetName == "" {
		return nil, fmt.Errorf("spreadsheet ID and sheet name are required")
	}

	actualID := extractSheetID(p.SpreadsheetID)
	store, err := persistience.NewData(actualID, p.SheetName)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	cache, err := store.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load cached data: %w", err)
	}
	return cache, nil
}

// SetCache writes the given SheetCache to local storage, completely replacing the existing snapshot.
func (s *SpreadsheetService) SetCache(p SpreadsheetServiceParams, cache *sheet.SheetCache) error {
	if p.SpreadsheetID == "" || p.SheetName == "" {
		return fmt.Errorf("spreadsheet ID and sheet name are required")
	}
	if cache == nil {
		return fmt.Errorf("cache cannot be nil")
	}

	actualID := extractSheetID(p.SpreadsheetID)
	store, err := persistience.NewData(actualID, p.SheetName)
	if err != nil {
		return fmt.Errorf("failed to initialize storage: %w", err)
	}

	if err := store.Merge(cache); err != nil {
		return fmt.Errorf("failed to merge cache: %w", err)
	}
	if err := store.Save(); err != nil {
		return fmt.Errorf("failed to save cache: %w", err)
	}
	return nil
}

// Update sends one or more cell changes to the remote spreadsheet.
// Value changes and note changes are batched separately.
func (s *SpreadsheetService) Update(ctx context.Context, p SpreadsheetServiceParams, changes ...CellChange) error {
	if p.CredentialFile == "" {
		return fmt.Errorf("credential file is required")
	}
	if p.SpreadsheetID == "" || p.SheetName == "" {
		return fmt.Errorf("spreadsheet ID and sheet name are required")
	}
	if len(changes) == 0 {
		return fmt.Errorf("at least one cell change is required")
	}

	srv, err := newSheetsService(ctx, p.CredentialFile)
	if err != nil {
		return err
	}
	actualID := extractSheetID(p.SpreadsheetID)

	// Separate changes into value and note updates
	var valueRanges []*sheets.ValueRange
	var noteRequests []*sheets.Request

	// Resolve sheet ID only if needed (for notes)
	var sheetID int64
	needSheetID := false
	for _, ch := range changes {
		if ch.Field == "note" {
			needSheetID = true
			break
		}
	}
	if needSheetID {
		sheetID, err = resolveSheetID(srv, actualID, p.SheetName)
		if err != nil {
			return err
		}
	}

	for _, ch := range changes {
		field := strings.ToLower(ch.Field)
		switch field {
		case "", "value":
			cellRef := fmt.Sprintf("%s!%s%d", p.SheetName, cell.ColIndexToLetter(ch.Col-1), ch.Row)
			valueRanges = append(valueRanges, &sheets.ValueRange{
				Range:  cellRef,
				Values: [][]any{{ch.Value}},
			})
		case "note":
			// Validate row/col
			if ch.Row < 1 || ch.Col < 1 {
				return fmt.Errorf("invalid cell position for note update: row=%d col=%d", ch.Row, ch.Col)
			}
			req := &sheets.Request{
				UpdateCells: &sheets.UpdateCellsRequest{
					Range: &sheets.GridRange{
						SheetId:          sheetID,
						StartRowIndex:    int64(ch.Row - 1),
						EndRowIndex:      int64(ch.Row),
						StartColumnIndex: int64(ch.Col - 1),
						EndColumnIndex:   int64(ch.Col),
					},
					Rows: []*sheets.RowData{
						{
							Values: []*sheets.CellData{
								{
									Note: ch.Value,
								},
							},
						},
					},
					Fields: "note",
				},
			}
			noteRequests = append(noteRequests, req)
		default:
			return fmt.Errorf("unsupported field %q: only 'value' or 'note' allowed", field)
		}
	}

	// Execute value updates
	if len(valueRanges) > 0 {
		req := &sheets.BatchUpdateValuesRequest{
			ValueInputOption: "USER_ENTERED",
			Data:             valueRanges,
		}
		if _, err := srv.Spreadsheets.Values.BatchUpdate(actualID, req).Do(); err != nil {
			return fmt.Errorf("failed to update values: %w", err)
		}
	}

	// Execute note updates
	if len(noteRequests) > 0 {
		batchReq := &sheets.BatchUpdateSpreadsheetRequest{
			Requests: noteRequests,
		}
		if _, err := srv.Spreadsheets.BatchUpdate(actualID, batchReq).Do(); err != nil {
			return fmt.Errorf("failed to update notes: %w", err)
		}
	}

	return nil
}

// resolveSheetID returns the numeric sheet ID for a given sheet name.
func resolveSheetID(srv *sheets.Service, spreadsheetID, sheetName string) (int64, error) {
	resp, err := srv.Spreadsheets.Get(spreadsheetID).Fields("sheets(properties(sheetId,title))").Do()
	if err != nil {
		return 0, fmt.Errorf("failed to get spreadsheet metadata: %w", err)
	}
	for _, s := range resp.Sheets {
		if s.Properties.Title == sheetName {
			return s.Properties.SheetId, nil
		}
	}
	return 0, fmt.Errorf("sheet %q not found", sheetName)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func newSheetsService(ctx context.Context, credFile string) (*sheets.Service, error) {
	data, err := os.ReadFile(credFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read credential file: %w", err)
	}
	return sheets.NewService(ctx,
		option.WithAuthCredentialsJSON(option.ImpersonatedServiceAccount, data),
		option.WithScopes(sheets.SpreadsheetsScope),
	)
}

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

// func colToLetter(n int) string {
// 	if n <= 0 {
// 		return "A"
// 	}
// 	letter := ""
// 	for n > 0 {
// 		n--
// 		letter = string(rune('A'+n%26)) + letter
// 		n /= 26
// 	}
// 	return letter
// }
