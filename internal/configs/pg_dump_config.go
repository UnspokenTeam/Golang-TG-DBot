package configs

type PgDumpConfig struct {
	Host     string `mapstructure:"POSTGRES_INTERNAL_HOST"`
	Username string `mapstructure:"POSTGRES_PGUSER"`
	DbName   string `mapstructure:"POSTGRES_PGDB"`
}
