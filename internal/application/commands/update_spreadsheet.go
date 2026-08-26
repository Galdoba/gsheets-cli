package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/Galdoba/gsheets-cli/internal/domain/cell"
	"github.com/Galdoba/gsheets-cli/internal/domain/sheet"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/config"
	"github.com/Galdoba/gsheets-cli/internal/service"
	"github.com/urfave/cli/v3"
)

func Update(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "update",
		Aliases: []string{"u"},
		Usage:   "Update cells in remote spreadsheet (edit + sync)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "updates",
				Aliases:  []string{"U"},
				Usage:    "Semicolon-separated list of cell updates: Cell=field=value, e.g. B6798=value=val1;B6799=note=val2",
				Required: true,
			},
		},
		Action: updateAction(cfg),
	}
}

func updateAction(cfg config.Config) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		parameters, err := getSpreadsheetData(cmd, cfg)
		if err != nil {
			return fmt.Errorf("failed to collect spreadsheet data: %w", err)
		}

		params := service.SpreadsheetServiceParams{
			SpreadsheetID:  parameters[dataSheetID],
			SheetName:      parameters[dataSheetName],
			CredentialFile: parameters[dataCredFile],
		}

		// Проверяем, есть ли уже накопленные изменения
		svc := service.NewSpreadsheetService()
		localCache, err := svc.LoadCache(params)
		if err == nil {
			if hasLocalChanges(localCache) {
				return fmt.Errorf("local cache already contains unsynced changes; use 'sync' to push them or 'edit' to add more")
			}
		}

		updatesStr := cmd.String("updates")
		changes, err := parseUpdates(updatesStr)
		if err != nil {
			return err
		}

		// Применяем изменения локально
		if err := svc.ApplyLocalChanges(params, changes...); err != nil {
			return err
		}

		// Синхронизируем
		err = svc.Sync(ctx, params)
		if err != nil {
			if syncErr, ok := err.(*service.SyncError); ok {
				fmt.Println("⚠️  Safeguard prevents sync due to conflicts or ambiguities:")
				fmt.Print(syncErr.Error())
				return nil
			}
			return err
		}

		fmt.Printf("✅ Successfully updated %d cell(s) and synced\n", len(changes))
		actualID := parameters[dataSheetID]
		if err := config.UpdateLastUsed(actualID, parameters[dataSheetName]); err != nil {
			return fmt.Errorf("failed to update last used table: %w", err)
		}
		return nil
	}
}

// hasLocalChanges возвращает true, если в кэше есть хотя бы одна изменённая ячейка
// (OriginalValue или OriginalNote непусты).
func hasLocalChanges(sc *sheet.SheetCache) bool {
	for _, row := range sc.Grid {
		for _, c := range row {
			if c.ValueChanged || c.NoteChanged {
				return true
			}
		}
	}
	return false
}

// parseUpdates converts the --updates string into []service.CellChange.
// Format: Cell=field=value (field optional, defaults to "value").
// Examples:
//
//	B6798=value=val1
//	B6799=note=val2
//	C3=hello        (same as C3=value=hello)
func parseUpdates(input string) ([]service.CellChange, error) {
	var changes []service.CellChange
	entries := strings.Split(input, ";")
	for _, entry := range entries {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 3)
		if len(parts) < 2 {
			return nil, fmt.Errorf("invalid update entry %q: expected Cell=value or Cell=field=value", entry)
		}
		cellRef := strings.TrimSpace(parts[0])
		field := "value"
		value := parts[1]
		if len(parts) == 3 {
			field = strings.TrimSpace(parts[1])
			value = parts[2]
		}

		row, col, err := cell.A1ToPosition(cellRef)
		if err != nil {
			return nil, fmt.Errorf("invalid cell reference %q: %w", cellRef, err)
		}
		changes = append(changes, service.CellChange{
			Row:   row,
			Col:   col,
			Field: strings.ToLower(field),
			Value: value,
		})
	}
	if len(changes) == 0 {
		return nil, fmt.Errorf("no valid updates provided")
	}
	return changes, nil
}
