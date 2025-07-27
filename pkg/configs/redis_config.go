package configs

type RedisConfig struct {
	Host string `env:"host"`
	Port int16  `env:"port"`
}
