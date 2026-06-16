package config

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/spf13/viper"
)

type Config struct {
	DB   DBConfig   `mapstructure:"db" validate:"required"`
	API  APIConfig  `mapstructure:"api" validate:"required"`
	LLM  LLMConfig  `mapstructure:"llm" validate:"required"`
	Log  LogConfig  `mapstructure:"log" validate:"required"`
}

type DBConfig struct {
	Path string `mapstructure:"path" validate:"required"`
}

type APIConfig struct {
	Port int `mapstructure:"port" validate:"required,min=1,max=65535"`
}

type LLMConfig struct {
	Endpoint string `mapstructure:"endpoint" validate:"required,url"`
}

type LogConfig struct {
	Level string `mapstructure:"level" validate:"required,oneof=debug info warn error"`
}

// Load reads configuration from environment variables, .env file, and defaults,
// then validates the resulting configuration struct.
func Load() (*Config, error) {
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	viper.SetEnvPrefix("PROWIKI")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// Default values
	viper.SetDefault("db.path", ".prowiki.db")
	viper.SetDefault("api.port", 8080)
	viper.SetDefault("llm.endpoint", "http://localhost:4000") // Mock LiteLLM default
	viper.SetDefault("log.level", "info")

	// Read in config file if it exists, ignore if it doesn't
	_ = viper.ReadInConfig()

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	validate := validator.New()
	if err := validate.Struct(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}
