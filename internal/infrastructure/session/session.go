package session

import (
	"encoding/json"
	"fmt"
	"gsheets-cli/internal/application"
	"os"
	"path/filepath"
	"sync"

	"github.com/Galdoba/appcontext/xdg"
)

var programDataDir string
var sessionDataDir string
var sessionFile string

func init() {
	programDataDir = xdg.Location(xdg.ForData(), xdg.WithProgramName(application.AppName))
	sessionDataDir = filepath.Join(programDataDir, "sessions")
	sessionFile = filepath.Join(sessionDataDir, "active.json")
}

type Session struct {
	mu           sync.RWMutex
	SheetName    string `json:"sheet_name"`
	TableName    string `json:"table_name"`
	TableIndex   int    `json:"table_index"`
	Presentation string `json:"presentation"`
}

func Restore() (*Session, error) {
	s := &Session{}
	data, err := os.ReadFile(sessionFile)
	if err != nil {
		if os.IsNotExist(err) {
			s.atomicSave()
		}
		return s, nil
	}

	if err := json.Unmarshal(data, s); err != nil {
		return s, fmt.Errorf("failed to unmarshal session data: %w", err)
	}
	return s, nil
}

func (s *Session) SetSheetName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.SheetName = name
	if err := s.atomicSave(); err != nil {
		fmt.Println(err)
	}
}

func (s *Session) SetTableName(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.TableName = name
	s.atomicSave()
}

func (s *Session) SetPresentation(pres string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Presentation = pres
	s.atomicSave()
}

func (s *Session) ActiveSheet() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.SheetName
}

func (s *Session) ActiveTable() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.TableName
}

func (s *Session) ActivePresentation() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Presentation
}

func (s *Session) atomicSave() error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}

	dir := filepath.Dir(sessionFile)
	os.MkdirAll(dir, 0777)
	tmpFile, err := os.CreateTemp(dir, sessionFile+".tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpName := tmpFile.Name()

	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return fmt.Errorf("write to temp file: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpName)
		return fmt.Errorf("sync temp file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpName, sessionFile); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("rename temp file to target: %w", err)
	}
	return nil
}
