package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/Galdoba/gsheets-cli/internal/domain/view"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/persistence/jsonstore"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/persistence/presetstore"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/tui"
)

func main() {
	spreadsheetTitle := "1Waa58usrgEal2Da6tyayaowiWujpm0rzd06P5ASYlsg"
	tableName := "График работ 2.0"

	// 1. Load Data from Local Storage
	dataStore, err := jsonstore.New(spreadsheetTitle, tableName)
	if err != nil {
		fmt.Printf("Failed to init storage: %v\n", err)
		os.Exit(1)
	}

	cache, err := dataStore.Load()
	if err != nil {
		fmt.Printf("Failed to load data: %v\n", err)
		os.Exit(1)
	}

	// Ensure dimensions are calculated if loading from an older JSON file
	cache.UpdateDimentions()

	fmt.Println("loaded", len(cache.Cells))
	// panic(0)

	// 2. Load User Preset (Presentation Layer)
	// We initialize the preset store. If no preset exists on disk, it defaults to NewDefault.
	pStore := presetstore.New(spreadsheetTitle, tableName, "Default", cache.ColCount())
	preset, err := pStore.Load()

	// If loading fails (e.g., file doesn't exist yet), fallback to a default preset
	if err != nil {
		defaultPreset := view.NewDefault(cache.ColCount())
		preset = &defaultPreset
	}

	// 3. Load Row Configuration (Optional, for now we pass nil to show all rows)
	// In the future, this will also be loaded from your persistence layer.
	var rowCfg *view.RowConfig

	// 4. Translate Domain -> Render
	renderCfg := view.BuildRenderConfig(cache, preset, rowCfg)

	// 5. Launch TUI
	m := tui.NewModel(cache, renderCfg)
	p := tea.NewProgram(m)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
