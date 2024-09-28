package app

import (
	"fmt"
	"github/zixrend/env_loader"
)

type Data struct {
	D1 string
	D2 string
}

func Run() {
	data := Data{}
	envl := env_loader.Env{}
	err := envl.LoadData(env_loader.DEVELOPMENT)
	if err != nil {
		return
	}
	fmt.Println(fmt.Sprintf("%s %s", data.D1, data.D2))
	fmt.Println("Bot started on port 8080")
}
