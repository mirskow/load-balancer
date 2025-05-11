// Config представляет конфигурацию всего приложения.
// Она включает настройки для HTTP сервера, балансировщика нагрузки, лимитера запросов и Redis.
//
// Для парсинга конфигурационного файла используется библиотека viper.
package config

import (
	"time"

	"github.com/spf13/viper"
)

type (
	Config struct {
		HTTP     HTTPConfig     `yaml:"http"`
		Balancer BalancerConfig `yaml:"balancer"`
		Limiter  LimiterConfig  `yaml:"limiter"`
		Redis    RedisConfig    `yaml:"redis"`
	}

	HTTPConfig struct {
		Port               string        `yaml:"port"`
		MaxHeaderMegabytes int           `yaml:"maxHeaderMegabytes"`
		ReadTimeout        time.Duration `yaml:"readTimeout"`
		WriteTimeout       time.Duration `yaml:"writeTimeout"`
	}

	BalancerConfig struct {
		Backends        []string      `yaml:"backends"`
		HealthCheckTime time.Duration `yaml:"healthCheckTime"`
	}

	LimiterConfig struct {
		Capacity   int           `yaml:"capacity"`
		RatePerSec int           `yaml:"ratePerSec"`
		TTL        int64         `yaml:"ttl"`
		RefillTime time.Duration `yaml:"refillTime"`
	}

	RedisConfig struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
	}
)

func Init(configsDir string) (*Config, error) {
	if err := parseConfigFile(configsDir); err != nil {
		return nil, err
	}

	var cfg Config
	if err := unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func parseConfigFile(configsDir string) error {
	viper.AddConfigPath(configsDir)
	viper.SetConfigName("config")

	if err := viper.ReadInConfig(); err != nil {
		return err
	}

	return nil
}

func unmarshal(cfg *Config) error {
	if err := viper.UnmarshalKey("http", &cfg.HTTP); err != nil {
		return err
	}

	if err := viper.UnmarshalKey("balancer", &cfg.Balancer); err != nil {
		return err
	}

	if err := viper.UnmarshalKey("limiter", &cfg.Limiter); err != nil {
		return err
	}

	if err := viper.UnmarshalKey("redis", &cfg.Redis); err != nil {
		return err
	}

	return nil
}
