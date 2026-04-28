package presetstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gsheets-cli/internal/application"
	"gsheets-cli/internal/domain/view"

	"github.com/Galdoba/appcontext/xdg"
)

type presetStore struct {
	path string
	data *view.Preset
	mu   sync.Mutex
}

func New(sheet, table, name string, columns int) *presetStore {
	pr := view.NewDefault(columns)
	return &presetStore{
		path: definePath(sheet, table, name),
		data: &pr,
		mu:   sync.Mutex{},
	}
}

func (s *presetStore) Load() (*view.Preset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dataBytes, err := os.ReadFile(s.path)
	if err != nil {
		return nil, err
	}

	var preset view.Preset
	if err := json.Unmarshal(dataBytes, &preset); err != nil {
		return nil, err
	}

	s.data = &preset
	return s.data, nil
}

func (s *presetStore) Save(preset *view.Preset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	dataBytes, err := json.MarshalIndent(preset, "", "  ")
	if err != nil {
		return err
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(s.path, dataBytes, 0644); err != nil {
		return err
	}

	s.data = preset
	return nil
}
func definePath(sheet, table, name string) string {
	file := fmt.Sprintf("%s---%s---%s.json", sheet, table, name)
	path := xdg.Location(xdg.ForData(), xdg.WithProgramName(application.AppName))
	return filepath.Join(path, "presets", file)
}
