package configs

import (
	"env_loader"
	"logger"
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
