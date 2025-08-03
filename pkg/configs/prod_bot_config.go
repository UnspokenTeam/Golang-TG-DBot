package configs

import (
	"env_loader"
	"log"
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
	err := envLoader.LoadDataIntoStruct(prodConfig)
	if err != nil {
		log.Fatal(err)
	}
	return prodConfig
}
