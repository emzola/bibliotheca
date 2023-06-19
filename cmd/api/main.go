package main

import (
	"context"
	"database/sql"
	"flag"
	"os"
	"strconv"
	"sync"
	"time"

	s3Config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/emzola/bibliotheca/internal/data"
	"github.com/emzola/bibliotheca/internal/jsonlog"
	"github.com/emzola/bibliotheca/internal/mailer"
	"github.com/jellydator/ttlcache/v3"
	_ "github.com/lib/pq"
)

const version = "1.0.0"

// The config struct holds all the configuration settings for the application.
type config struct {
	port int
	env  string
	s3   struct {
		client *s3.Client
	}
	db struct {
		dsn          string
		maxOpenConns int
		maxIdleConns int
		maxIdleTime  string
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
}

// The application struct holds the dependencies for our HTTP handlers,
// helpers and middleware.
type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
	mailer mailer.Mailer
	wg     sync.WaitGroup
	cache  *ttlcache.Cache[string, int64]
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment(development|staging|production)")

	// Read the database connection pool settings into the config struct
	flag.StringVar(&cfg.db.dsn, "db-dsn", "", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	// Read the rate limter settings into the config struct
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum requests per second")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// Read the SMTP server configuration settings into the config struct
	smtpport, err := strconv.Atoi(os.Getenv("SMTPPORT"))
	if err != nil {
		logger.PrintError(err, nil)
	}
	flag.StringVar(&cfg.smtp.host, "smtp-host", os.Getenv("SMTPHOST"), "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", smtpport, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", os.Getenv("SMTPUSERNAME"), "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", os.Getenv("SMTPPASSWORD"), "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Bibliotheca <no-reply@bibliotheca.com>", "SMTP sender")

	flag.Parse()

	// Open database connection
	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()
	logger.PrintInfo("database connection pool established", nil)

	// Initialize AWS S3 client
	err = aws3Config(&cfg)
	if err != nil {
		logger.PrintError(err, nil)
	}

	// In-memory caching with a ttl of 30 minutes
	cache := ttlcache.New(ttlcache.WithTTL[string, int64](30 * time.Minute))
	go cache.Start()

	app := &application{
		config: cfg,
		logger: logger,
		models: *data.NewModels(db),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
		cache:  cache,
	}

	// Start the HTTP server
	err = app.serve()
	if err != nil {
		app.logger.PrintFatal(err, nil)
	}
}

// openDB configures a PostgreSQL database connection pool.
func openDB(cfg config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.db.dsn)
	if err != nil {
		return nil, err
	}
	duration, err := time.ParseDuration(cfg.db.maxIdleTime)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(cfg.db.maxOpenConns)
	db.SetMaxIdleConns(cfg.db.maxIdleConns)
	db.SetConnMaxIdleTime(duration)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = db.PingContext(ctx)
	if err != nil {
		return nil, err
	}
	return db, nil
}

// aws3Config configures AWS S3 object storage.
func aws3Config(cfg *config) error {
	creds := credentials.NewStaticCredentialsProvider(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), "")
	awsCfg, err := s3Config.LoadDefaultConfig(context.TODO(), s3Config.WithCredentialsProvider(creds), s3Config.WithRegion(os.Getenv("AWS_S3_REGION")))
	if err != nil {
		return err
	}
	cfg.s3.client = s3.NewFromConfig(awsCfg)
	return nil
}
