package config

import (
	"fmt"

	"github.com/Galdoba/appcontext/configmanager"
	"github.com/Galdoba/gsheets-cli/internal/application"
)

type Config struct {
	Credentials Credentials     `toml:"credentials"`
	Sheets      Spreadsheets    `toml:"sheets"`
	Profiles    ProfileBindings `toml:"profiles"`
}

type Credentials struct {
	ActiveAccount   string            `toml:"active_account"`
	ServiceAccounts map[string]string `toml:"service_accounts"`
}

type Spreadsheets struct {
	LastUsed LastUsed         `toml:"last_used"`
	Tables   map[string]Table `toml:"tables"`
}

type LastUsed struct {
	TableID   string `toml:"table_id"`
	SheetName string `toml:"sheet_name"`
}

type Table struct {
	Address     string   `toml:"address"`
	SheetsNames []string `toml:"sheets_names"`
}

type ProfileBindings struct {
	Tables map[string][]string `toml:"tables"` // key: profileName, value: list of table identifiers
}

func Default() Config {
	return Config{
		Credentials: Credentials{
			ActiveAccount: "account",
			ServiceAccounts: map[string]string{
				"account": "path/to/credential file",
			},
		},
		Sheets: Spreadsheets{
			LastUsed: LastUsed{
				TableID:   "",
				SheetName: "",
			},
			Tables: map[string]Table{
				"{spreadsheet_alias}": {
					Address: "{https://example.com/path/to/spreadsheet}",
					SheetsNames: []string{
						"{table_name}",
					},
				},
			},
		},
		Profiles: ProfileBindings{
			Tables: map[string][]string{
				"default": {},
			},
		},
	}
}

func (cfg Config) ResolveProfileForTable(tableKey string) string {
	for profileName, tables := range cfg.Profiles.Tables {
		for _, t := range tables {
			if t == tableKey {
				return profileName
			}
		}
	}
	return "default"
}

// UpdateLastUsed persists the most recently used spreadsheet and table identifiers.
func UpdateLastUsed(tableID, sheetName string) error {
	cm, err := configmanager.New(application.AppName, Default(), configmanager.WithFormat(configmanager.TOML))
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}
	if err := cm.Load(); err != nil {
		if err := cm.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		if err := cm.Load(); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}
	cm.UpdateAndSave(func(c *Config) {
		if tableID != "" {
			c.Sheets.LastUsed.TableID = tableID
		}
		if sheetName != "" {
			c.Sheets.LastUsed.SheetName = sheetName
		}
	})
	return nil
}

// SetProfileBinding updates the profile binding for a given table.
// It removes the table from any existing profile bindings and adds it to the new profile.
func SetProfileBinding(spreadsheetID, sheetName, profileName string) error {
	cm, err := configmanager.New(application.AppName, Default(), configmanager.WithFormat(configmanager.TOML))
	if err != nil {
		return fmt.Errorf("failed to create config manager: %w", err)
	}
	if err := cm.Load(); err != nil {
		if err := cm.Save(); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}
		if err := cm.Load(); err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}
	}

	tableKey := spreadsheetID + "---" + sheetName

	cm.UpdateAndSave(func(c *Config) {
		// Ensure the map is initialized
		if c.Profiles.Tables == nil {
			c.Profiles.Tables = make(map[string][]string)
		}
		// Remove table from all profiles
		for prof, tables := range c.Profiles.Tables {
			newTables := make([]string, 0, len(tables))
			for _, t := range tables {
				if t != tableKey {
					newTables = append(newTables, t)
				}
			}
			c.Profiles.Tables[prof] = newTables
		}
		// Add table to the new profile
		c.Profiles.Tables[profileName] = append(c.Profiles.Tables[profileName], tableKey)
	})

	return nil
}
