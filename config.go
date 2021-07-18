package main

import (
	"os"
	"path"

	"gopkg.in/yaml.v2"
)

type Config struct {
	Token    string `yaml:"token"`
	ChatId   string `yaml:"chat_id"`
	Timezone string `yaml:"timezone"`
}

func ConfigDirectory(filename string) string {
	return path.Join(os.Getenv("BOT_CONFIG_DIRECTORY"), filename)
}

func LoadConfig() (Config, error) {
	f, err := os.Open(ConfigDirectory("config.yml"))
	if err != nil {
		return Config{}, err
	}
	defer f.Close()

	// TODO: validate config
	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}
