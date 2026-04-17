package storage

import (
	"gsheets-cli/internal/domain/cell"
	"gsheets-cli/internal/infrastructure/storage/jsonstore"
)

type Storage interface {
	Load(...string) (map[string]cell.Cell, error)
	Merge(...cell.Cell) error
	Save() error
}

func New(title, name string) (Storage, error) {
	return jsonstore.New(title, name)
}
