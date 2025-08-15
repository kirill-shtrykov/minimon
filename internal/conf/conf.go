package conf

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type Metric struct {
	Key      string `yaml:"key"`
	Method   string `yaml:"method"`
	Interval int    `yaml:"interval"`
	Type     string `yaml:"type,omitempty"`
}

type Graph struct {
	Metrics []string `yaml:"metrics"`
}

type Dashboard struct {
	Graphs []Graph `yaml:"graphs"`
}

type Config struct {
	DB      SQLiteConfig `yaml:"db"`
	Metrics []Metric     `yaml:"metrics"`
}

func LoadConfig(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}

	return &cfg, nil
}
