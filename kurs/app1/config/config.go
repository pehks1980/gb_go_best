package config

import (
	"fmt"
	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Port string `envconfig:"SERVICE_PORT" required:"true"`
	Host string `envconfig:"SERVICE_HOST" required:"true"`
	Timeout int `envconfig:"TIMEOUT" required:"true"`
}

func New(cfgFile string) (*Config, error) {
	err := godotenv.Load(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("load config file %q %w", cfgFile, err)
	}

	config := new(Config)
	err = envconfig.Process("", config)
	if err != nil {
		return nil, fmt.Errorf("get config from env variables: %w", err)
	}

	return config, nil
}

