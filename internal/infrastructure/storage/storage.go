package storage

import (
	"gsheets-cli/internal/infrastructure/storage/jsonstore"
	"gsheets-cli/internal/infrastructure/storage/presetstore"

	"gsheets-cli/internal/domain/sheet"
	"gsheets-cli/internal/domain/view"
)

type Data interface {
	Load() (*sheet.SheetCache, error)
	Merge(*sheet.SheetCache) error
	Save() error
}

func NewData(title, name string) (Data, error) {
	return jsonstore.New(title, name)
}

type Presentation interface {
	Load() (*view.Preset, error)
	Save(*view.Preset) error
}

func NewPresentation(title, table, name string, columns int) (Presentation, error) {
	ps := presetstore.New(title, table, name, columns)

}
