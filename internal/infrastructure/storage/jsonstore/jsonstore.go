package jsonstore

import (
	"encoding/json"
	"fmt"
	"gsheets-cli/internal/application"
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
	}
	return &js, nil
}

func definePath(title, Name string) string {
	file := fmt.Sprintf("%s---%s.json", title, Name)
	path := xdg.Location(xdg.ForData(), xdg.WithProgramName(application.AppName))
	defined := filepath.Join(path, "sheets", file)
	fmt.Printf("path: %q", defined)
	return defined

}

func (js *jsonStore) Load() (*sheet.SheetCache, error) {
	c := &sheet.SheetCache{}
	data, err := os.ReadFile(js.path)
	if err != nil {
		return nil, fmt.Errorf("failed to load json storage: %w", err)
	}
	if err := json.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("failed to unmarshal json storage file: %w", err)
	}
	return c, nil
}

/*

Summary of Required Implementations
Type / Function	Purpose
Storage struct	Holds path, mutex, and *sheet.SheetCache.
New(sheetID, sheetName string) (*Storage, error)	Constructor; initialises file path and loads existing data.
load() error	Reads JSON from disk into s.data.
Save() error	Writes s.data as indented JSON.
Read(a1 string) (cell.Cell, error)	Satisfies storage.Storage.
Update(cells map[string]cell.Cell) error	Satisfies storage.Storage.
CreateCell, UpdateCell, DeleteCell	Domain‑level operations with auto‑save (recommended).
By following this blueprint, the jsonstore package will be a robust, thread‑safe persistence layer that cleanly integrates with the rest of the application.

*/
