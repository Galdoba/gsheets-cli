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

func (s *Store) fileName(name string) string {
	// Sanitize: remove any path separators, just in case.
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, `\`, "_")
	return fmt.Sprintf("%s.json", name)
}

func (s *Store) List() ([]profile.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var profiles []profile.Profile

	if err := os.MkdirAll(s.dir, 0755); err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			p, err := s.loadFromFile(filepath.Join(s.dir, entry.Name()))
			if err == nil {
				profiles = append(profiles, *p)
			}
		}
	}
	return profiles, nil
}

func (s *Store) Get(name string) (*profile.Profile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadFromFile(filepath.Join(s.dir, s.fileName(name)))
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

	path := filepath.Join(s.dir, s.fileName(p.Name))
	return os.WriteFile(path, data, 0644)
}

func (s *Store) Delete(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	path := filepath.Join(s.dir, s.fileName(name))
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
