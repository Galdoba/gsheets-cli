package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/Galdoba/gsheets-cli/internal/domain/profile"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/persistence/jsonstore"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/tui"
	"github.com/Galdoba/gsheets-cli/pkg/filter"
)

// func main() {
// 	spreadsheetTitle := "1Waa58usrgEal2Da6tyayaowiWujpm0rzd06P5ASYlsg"
// 	tableName := "График работ 2.0"

// 	// 1. Load Data from Local Storage
// 	dataStore, err := jsonstore.New(spreadsheetTitle, tableName)
// 	if err != nil {
// 		fmt.Printf("Failed to init storage: %v\n", err)
// 		os.Exit(1)
// 	}

// 	cache, err := dataStore.Load()
// 	if err != nil {
// 		fmt.Printf("Failed to load data: %v\n", err)
// 		os.Exit(1)
// 	}

// 	// Ensure dimensions are calculated if loading from an older JSON file
// 	cache.UpdateDimentions()

// 	fmt.Println("loaded", len(cache.Cells))
// 	// panic(0)

// 	// 2. Load User Preset (Presentation Layer)
// 	// We initialize the preset store. If no preset exists on disk, it defaults to NewDefault.
// 	pStore := presetstore.New(spreadsheetTitle, tableName, "Default", cache.ColCount())
// 	preset, err := pStore.Load()

// 	// If loading fails (e.g., file doesn't exist yet), fallback to a default preset
// 	if err != nil {
// 		defaultPreset := view.NewDefault(cache.ColCount())
// 		preset = &defaultPreset
// 	}

// 	// 3. Load Row Configuration (Optional, for now we pass nil to show all rows)
// 	// In the future, this will also be loaded from your persistence layer.
// 	var rowCfg *view.RowConfig

// 	// 4. Translate Domain -> Render
// 	renderCfg := view.BuildRenderConfig(cache, preset, rowCfg)

// 	// 5. Launch TUI
// 	m := tui.NewModel(cache, renderCfg)
// 	p := tea.NewProgram(m)

// 	if _, err := p.Run(); err != nil {
// 		fmt.Printf("Error running TUI: %v\n", err)
// 		os.Exit(1)
// 	}
// }

func main() {
	fmt.Println("Initializing mock data...")

	spreadsheetTitle := "1Waa58usrgEal2Da6tyayaowiWujpm0rzd06P5ASYlsg"
	tableName := "График работ 2.0"

	// 1. Load Data from Local Storage
	dataStore, err := jsonstore.New(spreadsheetTitle, tableName)
	if err != nil {
		fmt.Printf("Failed to init storage: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(dataStore.Load())

	cache, err := dataStore.Load()
	if err != nil {
		fmt.Printf("Failed to load data: %v\n", err)
		os.Exit(1)
	}

	// Ensure dimensions are calculated if loading from an older JSON file
	cache.UpdateDimentions()
	fmt.Println(len(cache.Grid), "rows")

	panic(0)
	// 2. Define a Domain Profile
	// This struct represents what would normally be loaded from a JSON file on disk.
	p := profile.Profile{
		ID:        "vip-view",
		Name:      "VIP Clients Only",
		SheetID:   "test-sheet",
		TableName: "Sheet1",
		Layout: profile.LayoutConfig{
			// The order of this slice dictates the visual order in the TUI!
			Columns: []profile.ColumnState{
				{OriginalIndex: 0, Visibility: profile.Visible, WidthMode: profile.WidthFixed, FixedWidth: 12, AlignRight: true}, // ID (Fixed, Right-aligned)
				{OriginalIndex: 1, Visibility: profile.Visible, WidthMode: profile.WidthFixed, FixedWidth: 36},                   // Name (Auto width)
				{OriginalIndex: 2, Visibility: profile.Hidden},                                                                   // Department (Completely Hidden)
				{OriginalIndex: 4, Visibility: profile.Visible, WidthMode: profile.WidthAuto},                                    // Notes (Moved before Status)
				{OriginalIndex: 3, Visibility: profile.Collapsed, WidthMode: profile.WidthFixed, FixedWidth: 4},                  // Status (Collapsed into "+1" marker)
			},
		},
		Filters: filter.Group{
			Logic: "and",
			Rules: []filter.Rule{
				// Filter: Only show rows where the 'note' attribute of Column E (Index 4) is NOT EMPTY.
				// This will filter out 40 rows and only show the 10 "VIP Client" rows!
				{
					// Filter: Only show rows where the 'note' attribute of Column E (Index 4) is NOT EMPTY.
					// This will filter out 40 rows and only show the 10 "VIP Client" rows!
					ColumnIndex: 1,
					Attribute:   "value",
					Op:          filter.OpContains,
					Value:       "testtv",
					ColumnName:  "",
					AllCells:    false,
				},
			},
		},
	}

	// 3. Translate Domain Profile -> Render Config
	renderCfg, err := profile.BuildRenderConfig(cache, &p)
	if err != nil {
		fmt.Printf("Failed to build render config: %v\n", err)
		os.Exit(1)
	}

	// Optional: Patch the render config to show note markers ("*") for the Notes column
	// for i := range renderCfg.Columns {
	// 	if renderCfg.Columns[i].OriginalIndex == 4 {
	// 		renderCfg.Columns[i].NoteMode = render.NoteMark
	// 	}
	// }

	fmt.Println("Launching TUI...")

	// 4. Launch TUI
	m := tui.NewModel(cache, renderCfg)
	pTUI := tea.NewProgram(m)

	if _, err := pTUI.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
