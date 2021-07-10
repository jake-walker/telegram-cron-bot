package main

import (
	"os"

	"gopkg.in/yaml.v2"
)

type Job struct {
	Name    string   `yaml:"name"`
	Command []string `yaml:"command"`
}

type Config struct {
	Token    string `yaml:"token"`
	Jobs     []Job  `yaml:"jobs"`
	ChatId   string `yaml:"chat_id"`
	Timezone string `yaml:"timezone"`
}

func LoadConfig() (Config, error) {
	f, err := os.Open("config.yml")
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
