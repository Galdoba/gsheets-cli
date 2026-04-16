package flags

import "github.com/urfave/cli/v3"

const (
	Credentials = "credentials"
	Spreadsheet = "spreadsheet"
	Sheet       = "sheeet"
)

var GlobalCredentials = cli.StringFlag{
	Name:    "credentials",
	Aliases: []string{"c"},
	Usage:   "Path to Google Service Account JSON file",
}

var GlobalSpreadsheet = cli.StringFlag{
	Name:    "spreadsheet",
	Aliases: []string{"s"},
	Usage:   "Spreadsheet ID or full URL",
}

var GlobalTable = cli.StringFlag{
	Name:    "table",
	Aliases: []string{"t"},
	Usage:   "Table (sheet) name",
}
