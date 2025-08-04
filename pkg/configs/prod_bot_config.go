package configs

import (
	"env_loader"
	"logger"
)

type ProdBotConfig struct {
	ProdToken   string `env:"PROD_TOKEN"`
	CaddyDomain string `env:"CADDY_DOMAIN"`
	AppPort     int16  `env:"APP_PORT"`
	BufferSize  uint   `env:"BUFFER_SIZE"`
}

func GetProdConfig() *ProdBotConfig {
	envLoader := env_loader.CreateLoaderFromEnv()
	prodConfig := &ProdBotConfig{}
	if err := envLoader.LoadDataIntoStruct(prodConfig); err != nil {
		logger.LogFatal(err.Error(), "configuring", nil)
	}
	return prodConfig
}
