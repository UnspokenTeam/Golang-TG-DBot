package configs

type RateLimiterConfig struct {
	SpamCooldown int64 `mapstructure:"SPAM_COOLDOWN"`
	MuteCooldown int   `mapstructure:"MUTE_COOLDOWN"`
}
