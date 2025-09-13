package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server Server `yaml:"server"`
	Data   Data   `yaml:"data"`
	Upload Upload `yaml:"upload"`
}

type Server struct {
	Port int `yaml:"port"`
}

type Data struct {
	Dir string `yaml:"dir"`
}

type Upload struct {
	MaxSize    int64    `yaml:"maxSize"`
	AllowedTypes []string `yaml:"allowedTypes"`
}

var cfg *Config

func Load(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	cfg = &Config{}
	err = yaml.Unmarshal(data, cfg)
	return err
}

func Get() *Config {
	return cfg
}
