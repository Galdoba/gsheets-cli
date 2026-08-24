package config

import (
	"fmt"

	"github.com/Galdoba/appcontext/configmanager"
	"github.com/Galdoba/gsheets-cli/internal/application"
)

type Config struct {
	Credentials Credentials  `toml:"credentials"`
	Sheets      Spreadsheets `toml:"sheets"`
}

type Credentials struct {
	ActiveAccount   string            `toml:"active_account"`
	ServiceAccounts map[string]string `toml:"service_accounts"`
}

type Spreadsheets struct {
	LastUsed LastUse          `toml:"last_used"`
	Tables   map[string]Table `toml:"tables"`
}

type Table struct {
	Address     string   `toml:"address"`
	SheetsNames []string `toml:"sheets_names"`
}

type LastUse struct {
	TableID   string `toml:"table_id"`
	SheetName string `toml:"sheet_name"`
	ProfileID string `toml:"profile_id"`
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
			LastUsed: LastUse{
				TableID:   "",
				SheetName: "",
				ProfileID: "default",
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
	}
}

func (cfg Config) LastUsedTable() (string, string, string) {
	return cfg.Sheets.LastUsed.TableID, cfg.Sheets.LastUsed.SheetName, cfg.Sheets.LastUsed.ProfileID
}

func UpdateUsage(tableID, sheetName, profileID string) error {
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
		c.Sheets.LastUsed.TableID = updateNew(c.Sheets.LastUsed.TableID, tableID)
		c.Sheets.LastUsed.SheetName = updateNew(c.Sheets.LastUsed.SheetName, sheetName)
		c.Sheets.LastUsed.ProfileID = updateNew(c.Sheets.LastUsed.ProfileID, profileID)
	})

	return nil
}

func updateNew(old, new string) string {
	if new == "" {
		return old
	}
	return new
}
