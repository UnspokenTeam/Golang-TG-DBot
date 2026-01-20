package configs

import (
	"github.com/unspokenteam/golang-tg-dbot/pkg/env_loader"
	"github.com/unspokenteam/golang-tg-dbot/pkg/logger"
)

type BotCommandsConfig struct {
	HelpCommand string `env:"HELP_COMMAND"`
}

var BotCommands BotCommandsConfig

func LoadBotCommands() {
	envLoader := env_loader.CreateLoaderFromEnv()
	if err := envLoader.LoadDataIntoStruct(&BotCommands); err != nil {
		logger.LogFatal(err.Error(), "configuring", nil)
	}
}

func ReloadCommandConfigs() {}
