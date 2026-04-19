package jsonstore

import (
	"encoding/json"
	"errors"
	"fmt"
	"gsheets-cli/internal/application"
	"gsheets-cli/internal/domain/cell"
	"gsheets-cli/internal/domain/sheet"
	"os"
	"path/filepath"
	"sync"

	"github.com/Galdoba/appcontext/xdg"
)

type jsonStore struct {
	path string
	data *sheet.SheetCache
	mu   sync.Mutex
}

func New(title, name string) (*jsonStore, error) {
	js := jsonStore{
		path: definePath(title, name),
		data: sheet.New(title, name),
	}
	if err := js.loadData(); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to load existing storage: %w", err)
		}
	}
	return &js, nil
}

func definePath(title, name string) string {
	file := fmt.Sprintf("%s---%s.json", title, name)
	path := xdg.Location(xdg.ForData(), xdg.WithProgramName(application.AppName))
	return filepath.Join(path, "sheets", file)
}

func (js *jsonStore) loadData() error {
	js.mu.Lock()
	defer js.mu.Unlock()

	data, err := os.ReadFile(js.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			js.data = sheet.New(js.data.SpreadsheetTitle, js.data.SheetName)
			return nil
		}
		return fmt.Errorf("failed to read json storage: %w", err)
	}
	if err := json.Unmarshal(data, js.data); err != nil {
		return fmt.Errorf("failed to unmarshal json storage file: %w", err)
	}
	return nil
}

func (js *jsonStore) Load() (*sheet.SheetCache, error) {
	js.mu.Lock()
	defer js.mu.Unlock()
	return js.data, nil
}

func (js *jsonStore) Merge(fetched *sheet.SheetCache) error {
	js.mu.Lock()
	defer js.mu.Unlock()

	for a1, newCell := range fetched.Cells {
		oldCell, exists := js.data.Cells[a1]
		if !exists {
			newCell.UpdatedAt = fetched.LastSync
			js.data.Cells[a1] = newCell
			continue
		}
		if cell.Equal(oldCell, newCell) {
			continue
		}
		newCell.UpdatedAt = fetched.LastSync
		js.data.Cells[a1] = newCell
	}

	if fetched.RevisionID != "" {
		js.data.RevisionID = fetched.RevisionID
	}
	js.data.LastSync = fetched.LastSync

	return nil
}

func (js *jsonStore) Save() error {
	js.mu.Lock()
	defer js.mu.Unlock()

	dir := filepath.Dir(js.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(js.data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	if err := os.WriteFile(js.path, data, 0644); err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	return nil
}