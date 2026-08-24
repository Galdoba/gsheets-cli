package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Galdoba/gsheets-cli/internal/domain/profile"
	"github.com/Galdoba/gsheets-cli/internal/domain/sheet"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/config"
	persistience "github.com/Galdoba/gsheets-cli/internal/infrastructure/persistence"
	"github.com/urfave/cli/v3"
)

// ProfileCommand is the parent command for all profile management actions.
func ProfileCommand(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "profile",
		Aliases: []string{"p"},
		Usage:   "Manage rendering profiles (create, edit, delete, bind to tables)",
		Commands: []*cli.Command{
			profileList(cfg),
			profileShow(cfg),
			profileCreate(cfg),
			profileImport(cfg),
			profileExport(cfg),
			profileEdit(cfg),
			profileDelete(cfg),
			profileUse(cfg),
		},
	}
}

// ---------------------------------------------------------------------------
// Subcommand constructors
// ---------------------------------------------------------------------------

func profileList(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "list",
		Aliases: []string{"ls"},
		Usage:   "List all available profiles",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			repo := persistience.NewProfiles()
			profiles, err := repo.List()
			if err != nil {
				return fmt.Errorf("failed to list profiles: %w", err)
			}
			if len(profiles) == 0 {
				fmt.Println("No profiles found.")
				return nil
			}
			fmt.Println("Available profiles:")
			for _, p := range profiles {
				desc := ""
				if p.Description != "" {
					desc = fmt.Sprintf(" - %s", p.Description)
				}
				fmt.Printf("  %s%s\n", p.Name, desc)
			}
			return nil
		},
	}
}

func profileShow(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "show",
		Aliases: []string{"get"},
		Usage:   "Show a profile as JSON",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().First()
			if name == "" {
				return fmt.Errorf("profile name is required")
			}
			repo := persistience.NewProfiles()
			p, err := repo.Get(name)
			if err != nil {
				return fmt.Errorf("profile %q not found: %w", name, err)
			}
			data, err := json.MarshalIndent(p, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal profile: %w", err)
			}
			fmt.Println(string(data))
			return nil
		},
	}
}

func profileCreate(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "create",
		Aliases: []string{"new"},
		Usage:   "Create a new profile with default (show all) layout",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "description",
				Aliases: []string{"d"},
				Usage:   "Description for the profile",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().First()
			if name == "" {
				return fmt.Errorf("profile name is required")
			}
			// Use the builder to create a new profile and set description.
			p := profile.New(name)
			if desc := cmd.String("description"); desc != "" {
				p.WithDescription(desc)
			}
			if err := p.Validate(); err != nil {
				return err
			}
			repo := persistience.NewProfiles()
			// Check if already exists
			if _, err := repo.Get(name); err == nil {
				return fmt.Errorf("profile %q already exists", name)
			}
			if err := repo.Save(p); err != nil {
				return fmt.Errorf("failed to save profile: %w", err)
			}
			fmt.Printf("✅ Created profile %q\n", name)
			return nil
		},
	}
}

func profileImport(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "import",
		Aliases: []string{"load"},
		Usage:   "Import a profile from a JSON file",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
				Usage:   "Overwrite existing profile",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			filePath := cmd.Args().First()
			if filePath == "" {
				return fmt.Errorf("path to JSON file is required")
			}
			data, err := os.ReadFile(filePath)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			var p profile.Profile
			if err := json.Unmarshal(data, &p); err != nil {
				return fmt.Errorf("invalid profile JSON: %w", err)
			}
			if err := p.Validate(); err != nil {
				return err
			}
			repo := persistience.NewProfiles()
			_, err = repo.Get(p.Name)
			if err == nil && !cmd.Bool("force") {
				return fmt.Errorf("profile %q already exists (use --force to overwrite)", p.Name)
			}
			if err := repo.Save(&p); err != nil {
				return fmt.Errorf("failed to save profile: %w", err)
			}
			fmt.Printf("✅ Imported profile %q\n", p.Name)
			return nil
		},
	}
}

func profileExport(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "export",
		Aliases: []string{"save"},
		Usage:   "Export a profile to JSON (stdout or file)",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "output",
				Aliases: []string{"o"},
				Usage:   "Output file (default: stdout)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().First()
			if name == "" {
				return fmt.Errorf("profile name is required")
			}
			repo := persistience.NewProfiles()
			p, err := repo.Get(name)
			if err != nil {
				return fmt.Errorf("profile %q not found: %w", name, err)
			}
			data, err := json.MarshalIndent(p, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal profile: %w", err)
			}
			if out := cmd.String("output"); out != "" {
				if err := os.WriteFile(out, data, 0644); err != nil {
					return fmt.Errorf("failed to write output: %w", err)
				}
				fmt.Printf("✅ Exported profile %q to %s\n", name, out)
			} else {
				fmt.Println(string(data))
			}
			return nil
		},
	}
}

func profileEdit(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "edit",
		Aliases: []string{"e"},
		Usage:   "Edit a profile in your default editor",
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().First()
			if name == "" {
				return fmt.Errorf("profile name is required")
			}
			repo := persistience.NewProfiles()
			p, err := repo.Get(name)
			if err != nil {
				return fmt.Errorf("profile %q not found: %w", name, err)
			}

			// Create a temporary file
			tmpFile, err := os.CreateTemp("", "gsheets-profile-*.json")
			if err != nil {
				return fmt.Errorf("failed to create temp file: %w", err)
			}
			tmpPath := tmpFile.Name()
			defer os.Remove(tmpPath)

			data, err := json.MarshalIndent(p, "", "  ")
			if err != nil {
				return fmt.Errorf("failed to marshal profile: %w", err)
			}
			if _, err := tmpFile.Write(data); err != nil {
				return fmt.Errorf("failed to write temp file: %w", err)
			}
			tmpFile.Close()

			// Determine editor
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = os.Getenv("VISUAL")
			}
			if editor == "" {
				editor = "vi" // fallback
			}

			// Open editor
			editorCmd := exec.Command(editor, tmpPath)
			editorCmd.Stdin = os.Stdin
			editorCmd.Stdout = os.Stdout
			editorCmd.Stderr = os.Stderr
			if err := editorCmd.Run(); err != nil {
				return fmt.Errorf("editor failed: %w", err)
			}

			// Read modified file
			modifiedData, err := os.ReadFile(tmpPath)
			if err != nil {
				return fmt.Errorf("failed to read modified profile: %w", err)
			}
			var modified profile.Profile
			if err := json.Unmarshal(modifiedData, &modified); err != nil {
				return fmt.Errorf("invalid modified JSON: %w", err)
			}
			if err := modified.Validate(); err != nil {
				return err
			}
			// Ensure name is unchanged (or allow rename? For now, keep original name)
			modified.Name = p.Name

			if err := repo.Save(&modified); err != nil {
				return fmt.Errorf("failed to save profile: %w", err)
			}
			fmt.Printf("✅ Saved profile %q\n", modified.Name)
			return nil
		},
	}
}

func profileDelete(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "delete",
		Aliases: []string{"rm"},
		Usage:   "Delete a profile",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "yes",
				Aliases: []string{"y"},
				Usage:   "Skip confirmation",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().First()
			if name == "" {
				return fmt.Errorf("profile name is required")
			}
			if !cmd.Bool("yes") {
				fmt.Printf("Are you sure you want to delete profile %q? [y/N] ", name)
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Println("Cancelled.")
					return nil
				}
			}
			repo := persistience.NewProfiles()
			if err := repo.Delete(name); err != nil {
				return fmt.Errorf("failed to delete profile: %w", err)
			}
			fmt.Printf("✅ Deleted profile %q\n", name)
			return nil
		},
	}
}

func profileUse(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "use",
		Aliases: []string{"bind"},
		Usage:   "Bind the current table (from flags or last used) to a profile",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "spreadsheet",
				Aliases: []string{"s"},
				Usage:   "Spreadsheet ID or URL (overrides last used)",
			},
			&cli.StringFlag{
				Name:    "table",
				Aliases: []string{"t"},
				Usage:   "Sheet/table name (overrides last used)",
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			name := cmd.Args().First()
			if name == "" {
				return fmt.Errorf("profile name is required")
			}
			// Resolve table ID and sheet name
			tableID := cmd.String("spreadsheet")
			if tableID == "" {
				tableID = cfg.Sheets.LastUsed.TableID
			}
			sheetName := cmd.String("table")
			if sheetName == "" {
				sheetName = cfg.Sheets.LastUsed.SheetName
			}
			if tableID == "" || sheetName == "" {
				return fmt.Errorf("no table specified; use --spreadsheet and --table, or run fetch/update first")
			}
			// Normalize spreadsheet ID (extract from URL if necessary)
			actualID := sheet.ExtractSheetID(tableID)

			// Verify profile exists
			repo := persistience.NewProfiles()
			if _, err := repo.Get(name); err != nil {
				return fmt.Errorf("profile %q does not exist", name)
			}

			// Update binding
			tableKey := actualID + "---" + sheetName
			if err := config.SetProfileBinding(actualID, sheetName, name); err != nil {
				return fmt.Errorf("failed to update binding: %w", err)
			}
			fmt.Printf("✅ Bound table %q to profile %q\n", tableKey, name)
			return nil
		},
	}
}
