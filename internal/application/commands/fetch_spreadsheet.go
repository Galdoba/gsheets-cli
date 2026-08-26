package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Galdoba/gsheets-cli/internal/domain/rowmatch"
	"github.com/Galdoba/gsheets-cli/internal/domain/sheet"
	"github.com/Galdoba/gsheets-cli/internal/infrastructure/config"
	"github.com/Galdoba/gsheets-cli/internal/service"
	"github.com/urfave/cli/v3"
)

// fetch_spreadsheet.go
// Добавляем флаг --match-config и передачу конфигурации
func Fetch(cfg config.Config) *cli.Command {
	return &cli.Command{
		Name:    "fetch",
		Aliases: []string{"f"},
		Usage:   "Fetch remote spreadsheet data and save to local storage",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "credentials", Aliases: []string{"c"}, Usage: "Path to credentials JSON (overrides config)"},
			&cli.StringFlag{Name: "spreadsheet", Aliases: []string{"s"}, Usage: "Spreadsheet ID or URL"},
			&cli.StringFlag{Name: "table", Aliases: []string{"t"}, Usage: "Sheet name"},
			&cli.StringFlag{Name: "match-config", Usage: "Path to JSON file with rowmatch.Config"},
		},
		Action: fetchAction(cfg),
	}
}

// fetch_spreadsheet.go
func fetchAction(cfg config.Config) cli.ActionFunc {
	return func(ctx context.Context, cmd *cli.Command) error {
		parameters, err := getSpreadsheetData(cmd, cfg)
		if err != nil {
			return fmt.Errorf("failed to collect spreadsheet data: %w", err)
		}

		params := service.SpreadsheetServiceParams{
			SpreadsheetID:  parameters[dataSheetID],
			SheetName:      parameters[dataSheetName],
			CredentialFile: parameters[dataCredFile],
		}

		svc := service.NewSpreadsheetService()

		// Проверяем наличие локального кэша
		existingCache, loadErr := svc.LoadCache(params)
		if loadErr == nil && existingCache != nil {
			// Кэш существует: игнорируем флаг --match-config и используем сохранённую конфигурацию
			fmt.Println("Кэш существует: игнорируем флаг --match-config и используем сохранённую конфигурацию")
			existingConfig := existingCache.MatchRules

			if existingConfig != nil && len(existingConfig.Rules) > 0 {
				params.MatchConfig = existingConfig
			} else {
				// Если в существующем кэше нет конфигурации (например, старый кэш до введения safeguard),
				// то используем дефолт, чтобы не блокировать обновление после fetch.
				fmt.Println("Если в существующем кэше нет конфигурации (например, старый кэш до введения safeguard)то используем дефолт, чтобы не блокировать обновление после fetch.")

				defCfg := rowmatch.DefaultConfig()
				params.MatchConfig = &defCfg
			}
		} else {
			// Кэша нет – первичный fetch
			fmt.Println("Кэша нет – первичный fetch")
			if cfgPath := cmd.String("match-config"); cfgPath != "" {
				fmt.Println("cfgPath = match-config != ''")
				data, err := os.ReadFile(cfgPath)
				if err != nil {
					return fmt.Errorf("failed to read match config: %w", err)
				}
				var mc rowmatch.Config
				if err := json.Unmarshal(data, &mc); err != nil {
					return fmt.Errorf("invalid match config JSON: %w", err)
				}
				params.MatchConfig = &mc
			} else {
				// Нет ни файла, ни кэша – используем дефолтную конфигурацию
				defCfg := rowmatch.DefaultConfig()
				params.MatchConfig = &defCfg
				fmt.Println("use defrault conf")
			}
		}

		fmt.Println("Fetching remote data...")
		fmt.Println("передаем", params.MatchConfig)
		fetched, err := svc.Fetch(ctx, params)
		if err != nil {
			return err
		}

		fmt.Println("Saving to local cache...")
		if err := svc.SetCache(params, fetched); err != nil {
			return err
		}

		actualID := sheet.ExtractSheetID(parameters[dataSheetID])
		fmt.Printf("✅ Successfully synced %d rows to local storage\n", fetched.Rows)
		if err := config.UpdateLastUsed(actualID, parameters[dataSheetName]); err != nil {
			return fmt.Errorf("failed to update last used table: %w", err)
		}
		return nil
	}
}
