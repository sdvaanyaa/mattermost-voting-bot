package config

import (
	"log"
	"os"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Env              string `yaml:"env" env-default:"local"`
	TarantoolAddress string `yaml:"tarantool_address" env-default:"tarantool:3301"`
	MattermostURL    string `yaml:"mattermost_url" env-default:"http://mattermost:8065"`
	BotToken         string `yaml:"bot_token" env-required:"true"`
}

func MustLoad() *Config {
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		log.Fatal("CONFIG_PATH environment variable not set")
	}

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		log.Fatalf("CONFIG_PATH does not exist: %s", configPath)
	}

	var cfg Config
	if err := cleanenv.ReadConfig(configPath, &cfg); err != nil {
		log.Fatalf("unable to read config: %v", err)
	}

	return &cfg
}
