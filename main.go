package main

import (
	"flag"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emzola/bibliotheca/config"
	_ "github.com/emzola/bibliotheca/docs"
	"github.com/emzola/bibliotheca/handler"
	"github.com/emzola/bibliotheca/internal/jsonlog"
	"github.com/emzola/bibliotheca/repository"
	"github.com/emzola/bibliotheca/repository/postgres.go"
	"github.com/emzola/bibliotheca/service"
	"github.com/jellydator/ttlcache/v3"
)

// app defines the application's layers and shared resources.
type app struct {
	config  config.Config
	repo    repository.Repository
	service service.Service
	handler *handler.Handler
}

// @title  Bibliotheca API
// @version 1.0.0
// @description This is an API service for book uploads and downloads.
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email emma.idika@yahoo.com
// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html
// @host https://bibliotheca-api-dev.fl0.io
// @BasePath /
func main() {
	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// Initialize configuration
	var cfg config.Config
	flag.IntVar(&cfg.Port, "port", 4000, "API server port")
	flag.StringVar(&cfg.Env, "env", "development", "Environment(development|staging|production)")

	// Read the database connection pool settings into the config
	flag.StringVar(&cfg.Database.DSN, "db-dsn", os.Getenv("DSN"), "PostgreSQL DSN")
	flag.IntVar(&cfg.Database.MaxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.Database.MaxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.Database.MaxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	// Read the SMTP server settings into the config
	smtpport, err := strconv.Atoi(os.Getenv("SMTPPORT"))
	if err != nil {
		logger.PrintError(err, nil)
	}
	flag.StringVar(&cfg.SMTP.Host, "smtp-host", os.Getenv("SMTPHOST"), "SMTP host")
	flag.IntVar(&cfg.SMTP.Port, "smtp-port", smtpport, "SMTP port")
	flag.StringVar(&cfg.SMTP.Username, "smtp-username", os.Getenv("SMTPUSERNAME"), "SMTP username")
	flag.StringVar(&cfg.SMTP.Password, "smtp-password", os.Getenv("SMTPPASSWORD"), "SMTP password")
	flag.StringVar(&cfg.SMTP.Sender, "smtp-sender", "Bibliotheca <no-reply@bibliotheca-api-dev.fl0.io>", "SMTP sender")

	// Read AWS S3 settings into the config
	flag.StringVar(&cfg.S3.AccessKeyID, "s3-access-key", os.Getenv("AWSACCESSKEYID"), "S3 access key ID")
	flag.StringVar(&cfg.S3.SecretAccessKey, "s3-secret", os.Getenv("AWSSECRETACCESSKEY"), "S3 secret access key")
	flag.StringVar(&cfg.S3.Region, "s3-region", os.Getenv("AWSS3REGION"), "S3 Region")
	flag.StringVar(&cfg.S3.Bucket, "s3-bucket", os.Getenv("AWSS3BUCKET"), "S3 bucket")

	// Read the rate limter settings into the config
	flag.Float64Var(&cfg.Limiter.Rps, "limiter-rps", 4, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.Limiter.Burst, "limiter-burst", 8, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.Limiter.Enabled, "limiter-enabled", true, "Enable rate limiter")

	// Process the -cors-trusted-origins command line flag
	flag.Func("cors-trusted-origin", "Trusted CORS origin (space separated)", func(s string) error {
		cfg.Cors.TrustedOrigins = strings.Fields(s)
		return nil
	})

	flag.Parse()

	// Initialize database connection
	db, err := postgres.OpenDBConn(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()
	logger.PrintInfo("database connection pool established", nil)

	// Other shared resources: waitgroup and in-memory cache
	var wg sync.WaitGroup
	cache := ttlcache.New(ttlcache.WithTTL[string, int64](30 * time.Minute))
	go cache.Start()

	// Application layers
	repo := repository.New(db)
	service := service.New(cfg, &wg, logger, repo)
	handler := handler.New(cfg, logger, cache, service)

	// Instantiate application
	app := &app{
		config:  cfg,
		repo:    repo,
		service: service,
		handler: handler,
	}

	// Start HTTP server
	err = app.serve(&wg, logger)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
}
