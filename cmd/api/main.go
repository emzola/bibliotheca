package main

import (
	"context"
	"database/sql"
	"flag"
	"os"
	"time"

	s3Config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/emzola/bibliotheca/internal/data"
	"github.com/emzola/bibliotheca/internal/jsonlog"
	_ "github.com/lib/pq"
)

const version = "1.0.0"

// A config holds all the configuration settings for the application.
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
}

// An application holds the dependencies for our HTTP handlers,
// helpers and middleware.
type application struct {
	config config
	logger *jsonlog.Logger
	models data.Models
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment(development|staging|production)")

	flag.StringVar(&cfg.db.dsn, "db-dsn", "", "PostgreSQL DSN")
	flag.IntVar(&cfg.db.maxOpenConns, "db-max-open-conns", 25, "PostgreSQL max open connections")
	flag.IntVar(&cfg.db.maxIdleConns, "db-max-idle-conns", 25, "PostgreSQL max idle connections")
	flag.StringVar(&cfg.db.maxIdleTime, "db-max-idle-time", "15m", "PostgreSQL max connection idle time")

	flag.Parse()

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	// Open database connection
	db, err := openDB(cfg)
	if err != nil {
		logger.PrintFatal(err, nil)
	}
	defer db.Close()

	// Initialize AWS S3 client
	err = aws3Config(&cfg)
	if err != nil {
		logger.PrintError(err, nil)
	}

	app := &application{
		config: cfg,
		logger: logger,
		models: *data.NewModels(db),
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
