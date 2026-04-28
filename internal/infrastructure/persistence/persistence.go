package persistience

import (
	"gsheets-cli/internal/application"
	"gsheets-cli/internal/infrastructure/persistence/jsonstore"
	"gsheets-cli/internal/infrastructure/persistence/presetstore"
	"gsheets-cli/internal/infrastructure/persistence/session"

	"gsheets-cli/internal/domain/sheet"
	"gsheets-cli/internal/domain/view"

	"github.com/Galdoba/appcontext/xdg"
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
	return ps, nil
}

func NewSession() (*session.Sesion, error) {
	path := Path("active.json")
}

func Path(name string, subdirs ...string) string {
	options := []xdg.PathOption{}
	options = append(options, xdg.ForData())
	options = append(options, xdg.WithProgramName(application.AppName))
	if name != "" {
		options = append(options, xdg.WithFileName(name))
	}
	if len(subdirs) > 0 {
		options = append(options, xdg.WithSubDir(subdirs))
	}
	return xdg.Location(options...)
}
