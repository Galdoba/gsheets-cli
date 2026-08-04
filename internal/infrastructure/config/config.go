package config

import (
	"strings"
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
	LastUsedTable string           `toml:"last_used_table"`
	Tables        map[string]Table `toml:"tables"`
}

type Table struct {
	Address     string   `toml:"address"`
	SheetsNames []string `toml:"sheets_names"`
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
			LastUsedTable: "{spreadsheet_alias}::{table_name}",
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

func (cfg Config) LastUsedTable() (string, string) {
	lut := cfg.Sheets.LastUsedTable
	data := strings.Split(lut, "::")
	if len(data) != 2 {
		return "", ""
	}
	return data[0], data[1]
}
