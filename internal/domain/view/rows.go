package view

import "fmt"

type RowState uint8

const (
	RowStateUnset RowState = iota
	RowVisible             //normal view
	RowCollapsed           //is a part of RowGroup
	RowDirty               //local data modified
)

type RowGroup struct {
	Summary       string `json:"summary"`        //line of text that replace contained rows
	ContainedRows []int  `json:"contained_rows"` //indexes if rows assigned to group
	Collapsed     bool   `json:"collapsed"`      //skip group rendering or render summary line
}

type RowConfig struct {
	States map[int]RowState `json:"states"` //normal rows: key is 0-based row index
	Groups map[int]RowGroup `json:"groups"` //hidden groups: key is 0-based enumeration
}

func newRowConfiguration() *RowConfig {
	rc := RowConfig{}
	rc.States = make(map[int]RowState)
	rc.Groups = make(map[int]RowGroup)
	return &rc
}

func newGroup(summary string, rowIndexes ...int) (*RowGroup, error) {
	for _, row := range rowIndexes {
		fmt.Println(row)
		//return error if rows are not subsequent
	}
	gr := RowGroup{
		Summary:       summary,
		ContainedRows: rowIndexes,
		Collapsed:     false,
	}
	return &gr, nil
}

func (rc *RowConfig) Validate() error {
	return nil
}
