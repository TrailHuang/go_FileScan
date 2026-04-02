package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type ScannerConfig struct {
	WatchDirectories  []string   `mapstructure:"watch_directories"`
	LearningTablePath string     `mapstructure:"learning_table_path"`
	Scan              ScanConfig `mapstructure:"scan"`
	Log               LogConfig  `mapstructure:"log"`
}

type ScanConfig struct {
	MaxConcurrentScans int           `mapstructure:"max_concurrent_scans"`
	FileSizeLimit      string        `mapstructure:"file_size_limit"`
	ScanTimeout        time.Duration `mapstructure:"scan_timeout"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	MaxSize    string `mapstructure:"max_size"`
	MaxBackups int    `mapstructure:"max_backups"`
	MaxAge     int    `mapstructure:"max_age"`
}

type OutputConfig struct {
	Format            string `mapstructure:"format"`
	File              string `mapstructure:"file"`
	IncludeCleanFiles bool   `mapstructure:"include_clean_files"`
}

type Config struct {
	Scanner ScannerConfig `mapstructure:"scanner"`
	Output  OutputConfig  `mapstructure:"output"`
}

func LoadConfig(configPath string) (*Config, error) {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &config, nil
}
