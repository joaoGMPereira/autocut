package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
	Dir  string `yaml:"-"`
}

func Load(dir string) (*Config, error) {
	cfg := &Config{
		Host: "127.0.0.1",
		Port: 4070,
		Dir:  dir,
	}

	path := filepath.Join(dir, "config.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.Dir = dir
	return cfg, nil
}
