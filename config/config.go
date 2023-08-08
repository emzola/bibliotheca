package config

import "github.com/aws/aws-sdk-go-v2/service/s3"

// Config defines the app configuration.
type Config struct {
	Port int
	Env  string
	S3   struct {
		AccessKeyID     string
		SecretAccessKey string
		Region          string
		Bucket          string
		Client          *s3.Client
	}
	Database struct {
		DSN          string
		MaxOpenConns int
		MaxIdleConns int
		MaxIdleTime  string
	}
	SMTP struct {
		Host     string
		Port     int
		Username string
		Password string
		Sender   string
	}
	Limiter struct {
		Rps     float64
		Burst   int
		Enabled bool
	}
	Cors struct {
		TrustedOrigins []string
	}
	Metrics struct {
		Enabled bool
	}
	BasicAuth struct {
		Username string
		Password string
	}
}
