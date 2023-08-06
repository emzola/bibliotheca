package config

// Config defines the app configuration.
type Config struct {
	Server struct {
		Port int    `yaml:"port" env:"PORT"`
		Env  string `yaml:"env" env:"ENV"`
	} `yaml:"server"`
	Database struct {
		DSN          string `yaml:"dsn" env:"DSN"`
		MaxOpenConns int    `yaml:"max_open_conns" env:"MAXOPENCONNS"`
		MaxIdleConns int    `yaml:"max_idle_conns" env:"MAXIDLECONNS"`
		MaxIdleTime  string `yaml:"max_idle_time" env:"MAXIDLETIME"`
	} `yaml:"database"`
	Smtp struct {
		Host     string `yaml:"host" env:"SMTPHOST"`
		Port     int    `yaml:"port" env:"SMTPPORT"`
		Username string `yaml:"username" env:"SMTPUSERNAME"`
		Password string `yaml:"password" env:"SMTPPASSWORD"`
		Sender   string `yaml:"sender" env:"SMTPSENDER"`
	} `yaml:"smtp"`
	S3 struct {
		AccessKeyID     string `yaml:"access_key_id" env:"ACCESSKEYID"`
		SecretAccessKey string `yaml:"secret_access_key" env:"SECRETACCESSKEY"`
		Region          string `yaml:"region" env:"REGION"`
		Bucket          string `yaml:"bucket" env:"BUCKET"`
	} `yaml:"s3"`
	Limiter struct {
		RPS     float64 `yaml:"rps" env:"RPS"`
		Burst   int     `yaml:"burst" env:"BURST"`
		Enabled bool    `yaml:"enabled" env:"LENABLED"`
	} `yaml:"limiter"`
	Cors struct {
		TrustedOrigins []string `yaml:"trusted_origins" env:"TRUSTEDORIGINS"`
	} `yaml:"cors"`
	Metrics struct {
		Enabled bool `yaml:"enabled" env:"MENABLED"`
	} `yaml:"metrics"`
	BasicAuth struct {
		Username string `yaml:"username" env:"USERNAME"`
		Password string `yaml:"password" env:"PASSWORD"`
	} `yaml:"basic_auth"`
}
