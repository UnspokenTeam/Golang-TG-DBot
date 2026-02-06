package configs

import (
	"sync/atomic"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

type ConfigCache struct {
	cache atomic.Value
	viper *viper.Viper
}

func NewConfigCache(v *viper.Viper) *ConfigCache {
	cc := &ConfigCache{viper: v}
	cc.reload()

	v.WatchConfig()
	v.OnConfigChange(func(e fsnotify.Event) {
		cc.reload()
	})

	return cc
}

func (cc *ConfigCache) reload() {
	newCache := make(map[string]interface{})
	for key, value := range cc.viper.AllSettings() {
		newCache[key] = value
	}
	cc.cache.Store(newCache)
}

func (cc *ConfigCache) GetString(key string) string {
	cache := cc.cache.Load().(map[string]interface{})
	if v, ok := cache[key]; ok {
		if str, ok := v.(string); ok {
			return str
		}
	}
	return ""
}

func (cc *ConfigCache) GetInt(key string) int {
	cache := cc.cache.Load().(map[string]interface{})
	if v, ok := cache[key]; ok {
		if num, ok := v.(int); ok {
			return num
		}
	}
	return 0
}
