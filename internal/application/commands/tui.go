package commands

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/config"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/tui"
	"github.com/urfave/cli/v3"
)

// TUI returns the command that launches the interactive viewer.
func TUI(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "tui",
		Aliases: []string{"view"},
		Usage:   "Launch interactive table viewer (select table, profile, then render data)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "spreadsheet",
				Aliases: []string{"s"},
				Usage:   "Spreadsheet ID or URL (skips table selection)",
			},
			&cli.StringFlag{
				Name:    "table",
				Aliases: []string{"t"},
				Usage:   "Sheet/table name (skips table selection)",
			},
			&cli.StringFlag{
				Name:    "profile",
				Aliases: []string{"p"},
				Usage:   "Profile name (skips profile selection)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			initialTableID := cmd.String("spreadsheet")
			initialSheetName := cmd.String("table")
			initialProfileName := cmd.String("profile")

			setupModel := tui.NewSetupModel(cfg, initialTableID, initialSheetName, initialProfileName)
			p := tea.NewProgram(setupModel)
			if _, err := p.Run(); err != nil {
				return fmt.Errorf("TUI failed: %w", err)
			}
			return nil
		},
	}
}
