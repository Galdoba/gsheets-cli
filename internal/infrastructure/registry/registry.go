package registry

import "sync"

type registry struct {
	mu sync.Mutex
}

type sheetData struct {
	Address   string
	ID        string
	SheetName string
	Tables    []string
}

func (sd *sheetData) path(table string) (string, error) {

}
