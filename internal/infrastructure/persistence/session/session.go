package session

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
)

type Sesion struct {
	ID               string    `json:"id"`
	Name             string    `json:"name,omitempty"`
	Address          string    `json:"address"`
	SheetName        string    `json:"sheet_name"`
	TableName        string    `json:"table_name"`
	DataPath         string    `json:"data_path"`
	PresentationPath string    `json:"presentation_path"`
	LastSync         time.Time `json:"last_sync"`
}

type Configuration struct {
	Name             string `json:"name,omitempty"`
	Address          string `json:"address"`
	SheetName        string `json:"sheet_name"`
	TableName        string `json:"table_name"`
	DataPath         string `json:"data_path"`
	PresentationPath string `json:"presentation_path"`
}

func new(path string, conf Configuration) (*Sesion, error) {
	s := Sesion{
		ID:               uuid.NewString(),
		Name:             conf.Name,
		Address:          conf.Address,
		SheetName:        conf.SheetName,
		TableName:        conf.TableName,
		DataPath:         conf.DataPath,
		PresentationPath: conf.PresentationPath,
	}
	if err := s.atomicSave(path); err != nil {
		return nil, fmt.Errorf("failed to save session data: %w", err)
	}
	return &s, nil
}

func (s *Sesion) atomicSave(path string) error {
	data, err := json.MarshalIndent(&s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session state")
	}
	temp := path + ".tmp"
	f, err := os.Create(temp)
	if err != nil {
		return fmt.Errorf("failed to create session file: %w", err)
	}

	if _, err := f.Write(data); err != nil {
		return fmt.Errorf("failed to write %s: %w", s.ID, err)
	}
	if err := os.Rename(temp, path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	return nil
}

func (s *Sesion) Sync() {
	s.LastSync = time.Now()
}
