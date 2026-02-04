package configs

import "fmt"

type PostgresConfig struct {
	Host     string `mapstructure:"POSTGRES_HOST"`
	Port     int    `mapstructure:"POSTGRES_PORT"`
	Username string `mapstructure:"POSTGRES_PGUSER"`
	Password string `mapstructure:"POSTGRES_PGPASS"`
	DbName   string `mapstructure:"POSTGRES_PGDB"`
}

func (cfg *PostgresConfig) GetConnectionString() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s",
		cfg.Host,
		cfg.Port,
		cfg.Username,
		cfg.Password,
		cfg.DbName,
	)
}
