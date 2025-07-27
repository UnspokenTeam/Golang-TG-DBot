package app

import (
	"fmt"
	"github.com/zixrend/env_loader"
	"github.com/zixrend/golang-tg-bot/pkg/configs"
	"os"
)

func Run() {
	env := os.Getenv("GO_ENV")
	if env == "DEVELOPMENT" {
		fmt.Println("DEVELOPMENT")
	}
	config := env_loader.GetFromEnv[configs.RedisConfig]()
	fmt.Println(fmt.Sprintf("%s %s", config.Host, config.Port))
	fmt.Println("Bot started on port 8080")
}
