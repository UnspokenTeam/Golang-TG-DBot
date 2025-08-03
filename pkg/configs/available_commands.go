package configs

import (
	"env_loader"
	"log"
)

type BotCommandsConfig struct {
	HelpCommand string `env:"HELP_COMMAND"`
}

var BotCommands BotCommandsConfig

func LoadBotCommands() {
	envLoader := env_loader.CreateLoaderFromEnv()
	err := envLoader.LoadDataIntoStruct(&BotCommands)
	if err != nil {
		log.Fatal(err)
	}
}
