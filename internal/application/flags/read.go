package flags

import "github.com/urfave/cli/v3"

const (
	Output = "output"
)

var ReadCmdOutput = cli.StringFlag{
	Name:    "output",
	Aliases: []string{"o"},
	Usage:   "Path to save CSV file",
	Value:   "output.csv",
}
