package storage

import (
	"gsheets-cli/internal/infrastructure/storage/jsonstore"

	"gsheets-cli/internal/domain/sheet"
)

type Storage interface {
	Load() (*sheet.SheetCache, error)
	Merge(*sheet.SheetCache) error
	Save() error
}

func New(title, name string) (Storage, error) {
	return jsonstore.New(title, name)
}
