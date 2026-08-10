package profilestore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/Galdoba/appcontext/xdg"
	"github.com/Galdoba/gsheets-cli/internal/application"
	"github.com/Galdoba/gsheets-cli/internal/domain/profile"
)

type Store struct {
	dir string
	mu  sync.Mutex
}

func New() *Store {
	dir := xdg.Location(xdg.ForData(), xdg.WithProgramName(application.AppName), xdg.WithSubDir([]string{"profiles"}))
	return &Store{dir: dir}
}

func (s *Store) fileName(sheetID, tableName, profileID string) string {
	return fmt.Sprintf("%s---%s---%s.json", sheetID, tableName, profileID)
}

func (s *Store) List(sheetID, tableName string) ([]profile.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	prefix := fmt.Sprintf("%s---%s---", sheetID, tableName)
	var profiles []profile.Profile

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) && strings.HasSuffix(entry.Name(), ".json") {
			p, err := s.loadFromFile(filepath.Join(s.dir, entry.Name()))
			if err == nil {
				profiles = append(profiles, *p)
			}
		}
	}
	return profiles, nil
}

func (s *Store) Get(sheetID, tableName, profileID string) (*profile.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadFromFile(filepath.Join(s.dir, s.fileName(sheetID, tableName, profileID)))
}

func (s *Store) Save(p *profile.Profile) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return err
	}

	path := filepath.Join(s.dir, s.fileName(p.SheetID, p.TableName, p.ID))
	return os.WriteFile(path, data, 0644)
}

func (s *Store) Delete(sheetID, tableName, profileID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.dir, s.fileName(sheetID, tableName, profileID))
	return os.Remove(path)
}

func (s *Store) loadFromFile(path string) (*profile.Profile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var p profile.Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, err
	}
	return &p, nil
}
