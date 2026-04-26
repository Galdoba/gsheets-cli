package infrastructure

import (
	"fmt"
	"gsheets-cli/internal/application"
	"gsheets-cli/internal/infrastructure/config"

	"github.com/Galdoba/appcontext/configmanager"
)

type Infrastructure struct {
	Config config.Config
}

func Initalize() (*Infrastructure, error) {
	inf := Infrastructure{}
	// cm, err := configmanager.New(application.AppName, config.Default(), configmanager.WithSerializationFormat(configmanager.TOML))
	// if err != nil {
	// 	return nil, fmt.Errorf("failed to create config manager: %w", err)
	// }
	// if err := cm.Load(); err != nil {
	// 	if err := cm.Save(); err != nil {
	// 		return nil, fmt.Errorf("failed to save config: %w", err)
	// 	}
	// 	if err := cm.Load(); err != nil {
	// 		return nil, fmt.Errorf("failed to load config: %w", err)
	// 	}
	// }
	// inf.Config = cm.Config()
	cfg, err := configmanager.LazyInit(application.AppName, config.Default())
	if err != nil {
		return nil, fmt.Errorf("failed to init configuration")
	}
	inf.Config = cfg
	return &inf, nil
}
