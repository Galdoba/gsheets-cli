package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Galdoba/gsheets-cli/internal/domain/cell"
	"github.com/Galdoba/gsheets-cli/internal/domain/rowmatch"
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
	MatchConfig    *rowmatch.Config
}

// CellChange – алиас на тип из sheet для удобства.
type CellChange = sheet.CellChange

// SpreadsheetService provides methods for fetching, caching, and updating Google Sheets data.
type SpreadsheetService struct{}

// NewSpreadsheetService creates a new SpreadsheetService.
func NewSpreadsheetService() *SpreadsheetService {
	return &SpreadsheetService{}
}

// Fetch retrieves remote spreadsheet data and returns it as a fresh SheetCache.
// It does not touch local storage.
// Если p.MatchConfig не nil, конфигурация будет установлена в кэш (но отпечатки не вычисляются).
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

	if p.MatchConfig != nil {
		if err := fetched.SetMatchConfig(p.MatchConfig); err != nil {
			return nil, fmt.Errorf("failed to set match config: %w", err)
		}
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

// Sync синхронизирует уже изменённый локальный кэш (с OriginalValue/OriginalNote) с удалённой таблицей.
// При успехе сбрасывает Original-поля и сохраняет обновлённый кэш.
func (s *SpreadsheetService) Sync(ctx context.Context, p SpreadsheetServiceParams) error {
	local, err := s.LoadCache(p)
	if err != nil {
		return fmt.Errorf("failed to load local cache: %w (run 'fetch' first)", err)
	}
	return s.syncWithCache(ctx, p, local)
}

// Update применяет изменения к локальному кэшу и синхронизирует с сервером.
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

	// Загружаем локальный кэш
	local, err := s.LoadCache(p)
	if err != nil {
		return fmt.Errorf("failed to load local cache: %w (run 'fetch' first)", err)
	}

	// Создаём копию и применяем изменения
	working := local.Clone()
	if err := working.ApplyChanges(changes); err != nil {
		return fmt.Errorf("failed to apply local changes: %w", err)
	}

	// Синхронизируем копию
	return s.syncWithCache(ctx, p, working)
}

type pendingUpdate struct {
	NewRow int
	Change CellChange
}

// syncWithCache выполняет проверку конфликтов и отправку изменений на сервер.
// После успеха сбрасывает Original-поля и сохраняет кэш.
func (s *SpreadsheetService) syncWithCache(ctx context.Context, p SpreadsheetServiceParams, local *sheet.SheetCache) error {
	fmt.Println("start syncWithCache")
	// Проверяем наличие конфигурации safeguard
	if local.MatchRules == nil || len(local.MatchRules.Rules) == 0 {
		return fmt.Errorf("safeguard not configured; run 'fetch' with a match config")
	}

	// Получаем актуальные удалённые данные
	p2 := p
	p2.MatchConfig = local.MatchRules // передаём конфиг, чтобы Fetch установил его
	remote, err := s.Fetch(ctx, p2)
	if err != nil {
		return err
	}

	// Вычисляем отпечатки
	localFPs, err := local.ComputeFingerprints()
	if err != nil {
		return fmt.Errorf("failed to compute local fingerprints: %w", err)
	}
	remoteFPs, err := remote.ComputeFingerprints()
	if err != nil {
		return fmt.Errorf("failed to compute remote fingerprints: %w", err)
	}

	// Находим все изменённые строки (где есть OriginalValue или OriginalNote)
	changedRows := make(map[int]bool)
	fmt.Println("chandedRows created")
	for rowIdx := 0; rowIdx < local.Rows; rowIdx++ {
		for colIdx := 0; colIdx < local.Cols; colIdx++ {
			c := local.Grid[rowIdx][colIdx]
			if c.ValueChanged || c.NoteChanged {
				changedRows[rowIdx] = true
				break
			}
		}
	}

	// В syncWithCache, после цикла поиска изменённых строк:
	fmt.Printf("DEBUG: found %d changed rows\n", len(changedRows))

	// Для каждой изменённой строки выполняем сопоставление и проверку конфликтов
	var safeUpdates []pendingUpdate
	var escalations []rowmatch.MatchResult
	var conflicts []Conflict

	for oldRowIdx := range changedRows {
		oldRow := oldRowIdx + 1
		localFP := localFPs[oldRowIdx]
		matchResult := rowmatch.Match(localFP, remoteFPs, local.MatchRules)

		switch matchResult.Decision {
		case rowmatch.DecisionAutoAccept:
			newIdx := matchResult.Best.Index // 0‑based индекс в удалённой сетке
			// Проверяем каждую изменённую ячейку в этой строке
			rowCells := local.Grid[oldRowIdx]
			for colIdx := 0; colIdx < len(rowCells); colIdx++ {
				c := rowCells[colIdx]
				if c.OriginalValue == "" && c.OriginalNote == "" {
					continue // ячейка не менялась
				}
				// Столбец (1‑based)
				col := colIdx + 1
				remoteCell := remote.GetCell(newIdx+1, col)

				// Проверяем значение
				if c.ValueChanged {
					if c.OriginalValue != remoteCell.Value {
						conflicts = append(conflicts, Conflict{
							Row: oldRow, Col: col, Field: "value",
							Reason: fmt.Sprintf("cell value changed concurrently: local=%q remote=%q", c.OriginalValue, remoteCell.Value),
						})
						continue
					}
					safeUpdates = append(safeUpdates, pendingUpdate{
						NewRow: newIdx + 1,
						Change: CellChange{Row: oldRow, Col: col, Field: "value", Value: c.Value},
					})
				}

				// Проверяем заметку
				if c.NoteChanged {
					if c.OriginalNote != remoteCell.Note {
						conflicts = append(conflicts, Conflict{
							Row: oldRow, Col: col, Field: "note",
							Reason: fmt.Sprintf("cell note changed concurrently: local=%q remote=%q", c.OriginalNote, remoteCell.Note),
						})
						continue
					}
					safeUpdates = append(safeUpdates, pendingUpdate{
						NewRow: newIdx + 1,
						Change: CellChange{Row: oldRow, Col: col, Field: "note", Value: c.Note},
					})
				}
			}
		case rowmatch.DecisionEscalate:
			escalations = append(escalations, matchResult)
		case rowmatch.DecisionReject:
			conflicts = append(conflicts, Conflict{
				Row: oldRow, Col: 0, Field: "", Reason: matchResult.Reason,
			})
		}
	}

	// После цикла обработки строк:
	fmt.Printf("DEBUG: safeUpdates = %d, escalations = %d, conflicts = %d\n",
		len(safeUpdates), len(escalations), len(conflicts))

	// Если есть конфликты или эскалации — ничего не отправляем
	if len(escalations) > 0 || len(conflicts) > 0 {
		return &SyncError{
			Escalations: escalations,
			Conflicts:   conflicts,
		}
	}

	// Все изменения безопасны — отправляем
	if err := s.sendUpdates(ctx, p, safeUpdates); err != nil {
		return err
	}

	// Сбрасываем Original-поля во всех ячейках
	for rowIdx := 0; rowIdx < local.Rows; rowIdx++ {
		for colIdx := 0; colIdx < local.Cols; colIdx++ {
			local.Grid[rowIdx][colIdx].ResetOriginal()
		}
	}

	// Сохраняем обновлённый кэш
	if err := s.SetCache(p, local); err != nil {
		return fmt.Errorf("failed to save synced cache: %w", err)
	}
	return nil
}

// sendUpdates отправляет безопасные изменения на сервер.
func (s *SpreadsheetService) sendUpdates(ctx context.Context, p SpreadsheetServiceParams, updates []pendingUpdate) error {
	if len(updates) == 0 {
		return nil
	}
	srv, err := newSheetsService(ctx, p.CredentialFile)
	if err != nil {
		return err
	}
	actualID := extractSheetID(p.SpreadsheetID)

	var valueRanges []*sheets.ValueRange
	var noteRequests []*sheets.Request
	var sheetID int64
	needSheetID := false
	for _, u := range updates {
		if u.Change.Field == "note" {
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

	for _, u := range updates {
		ch := u.Change
		newRow := u.NewRow
		field := strings.ToLower(ch.Field)
		switch field {
		case "", "value":
			cellRef := fmt.Sprintf("%s!%s%d", p.SheetName, cell.ColIndexToLetter(ch.Col-1), newRow)
			valueRanges = append(valueRanges, &sheets.ValueRange{
				Range:  cellRef,
				Values: [][]any{{ch.Value}},
			})
		case "note":
			req := &sheets.Request{
				UpdateCells: &sheets.UpdateCellsRequest{
					Range: &sheets.GridRange{
						SheetId:          sheetID,
						StartRowIndex:    int64(newRow - 1),
						EndRowIndex:      int64(newRow),
						StartColumnIndex: int64(ch.Col - 1),
						EndColumnIndex:   int64(ch.Col),
					},
					Rows: []*sheets.RowData{
						{
							Values: []*sheets.CellData{
								{Note: ch.Value},
							},
						},
					},
					Fields: "note",
				},
			}
			noteRequests = append(noteRequests, req)
		}
	}

	if len(valueRanges) > 0 {
		req := &sheets.BatchUpdateValuesRequest{
			ValueInputOption: "USER_ENTERED",
			Data:             valueRanges,
		}
		if _, err := srv.Spreadsheets.Values.BatchUpdate(actualID, req).Do(); err != nil {
			return fmt.Errorf("failed to update values: %w", err)
		}
	}
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

// Conflict описывает одно конфликтное изменение ячейки.
type Conflict struct {
	Row    int
	Col    int
	Field  string
	Reason string
}

// SyncError объединяет эскалации и конфликты.
type SyncError struct {
	Escalations []rowmatch.MatchResult
	Conflicts   []Conflict
}

func (e *SyncError) Error() string {
	var sb strings.Builder
	sb.WriteString("safeguard failed:\n")
	if len(e.Escalations) > 0 {
		sb.WriteString("Escalations:\n")
		for _, esc := range e.Escalations {
			sb.WriteString(fmt.Sprintf("  row %d: confidence %.2f, reason: %s, candidates:\n", esc.Best.Index+1, esc.Confidence, esc.Reason))
			for _, c := range esc.Candidates {
				sb.WriteString(fmt.Sprintf("    - %s\n", c.String()))
			}
		}
	}
	if len(e.Conflicts) > 0 {
		sb.WriteString("Conflicts:\n")
		for _, c := range e.Conflicts {
			sb.WriteString(fmt.Sprintf("  row %d, col %d (%s): %s\n", c.Row, c.Col, c.Field, c.Reason))
		}
	}
	return sb.String()
}

// ApplyLocalChanges загружает локальный кэш, применяет изменения и сохраняет его.
// Этот метод не обращается к Google API и может вызываться многократно.
func (s *SpreadsheetService) ApplyLocalChanges(p SpreadsheetServiceParams, changes ...CellChange) error {
	if len(changes) == 0 {
		return fmt.Errorf("at least one change is required")
	}
	local, err := s.LoadCache(p)
	if err != nil {
		return fmt.Errorf("failed to load local cache: %w (run 'fetch' first)", err)
	}
	if err := local.ApplyChanges(changes); err != nil {
		return err
	}
	if err := s.SetCache(p, local); err != nil {
		return err
	}
	return nil
}
