package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config defines the app configuration.
type Config struct {
	Server struct {
		Port int    `yaml:"port"`
		Env  string `yaml:"env"`
	} `yaml:"server"`
	Database struct {
		DSN          string `yaml:"dsn"`
		MaxOpenConns int    `yaml:"max_open_conns"`
		MaxIdleConns int    `yaml:"max_idle_conns"`
		MaxIdleTime  string `yaml:"max_idle_time"`
	} `yaml:"database"`
	Smtp struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
		Sender   string `yaml:"sender"`
	} `yaml:"smtp"`
	S3 struct {
		AccessKeyID     string `yaml:"access_key_id"`
		SecretAccessKey string `yaml:"secret_access_key"`
		Region          string `yaml:"region"`
		Bucket          string `yaml:"bucket"`
	} `yaml:"s3"`
	Limiter struct {
		RPS     float64 `yaml:"rps"`
		Burst   int     `yaml:"burst"`
		Enabled bool    `yaml:"enabled"`
	} `yaml:"limiter"`
	Cors struct {
		TrustedOrigins []string `yaml:"trusted_origins"`
	} `yaml:"cors"`
	Metrics struct {
		Enabled bool `yaml:"enabled"`
	} `yaml:"metrics"`
	BasicAuth struct {
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"basic_auth"`
}

// Decode de-serializes the config.yml file into Go types.
func Decode() (Config, error) {
	f, err := os.Open("config.yml")
	if err != nil {
		return Config{}, err
	}
	defer f.Close()
	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		return Config{}, err
	}
	return cfg, nil
}
