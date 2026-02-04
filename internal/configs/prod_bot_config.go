package configs

type ProdBotConfig struct {
	ProdToken   string `mapstructure:"PROD_TOKEN"`
	CaddyDomain string `mapstructure:"CADDY_DOMAIN"`
	BufferSize  uint   `mapstructure:"BUFFER_SIZE"`
}
