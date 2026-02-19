package configs

import "fmt"

type LlmConfig struct {
	Host  string `mapstructure:"LLM_HOST"`
	Port  int    `mapstructure:"LLM_PORT"`
	Model string `mapstructure:"LLM_MODEL"`
}

func (cfg *LlmConfig) GetBaseUrl() string {
	return fmt.Sprintf(
		"http://%s:%d",
		cfg.Host,
		cfg.Port,
	)
}
