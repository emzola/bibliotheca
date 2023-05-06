package main

import (
	"context"
	"flag"
	"os"

	s3Config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/emzola/bibliotheca/internal/jsonlog"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	s3   struct {
		client *s3.Client
	}
}

type application struct {
	config config
	logger *jsonlog.Logger
}

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment(development|staging|production)")
	flag.Parse()

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	err := aws3Config(&cfg)
	if err != nil {
		logger.PrintError(err, nil)
	}

	app := &application{
		config: cfg,
		logger: logger,
	}

	err = app.serve()
	if err != nil {
		app.logger.PrintFatal(err, nil)
	}
}

func aws3Config(cfg *config) error {
	creds := credentials.NewStaticCredentialsProvider(os.Getenv("AWS_ACCESS_KEY_ID"), os.Getenv("AWS_SECRET_ACCESS_KEY"), "")
	awsCfg, err := s3Config.LoadDefaultConfig(context.TODO(), s3Config.WithCredentialsProvider(creds), s3Config.WithRegion(os.Getenv("AWS_S3_REGION")))
	if err != nil {
		return err
	}
	cfg.s3.client = s3.NewFromConfig(awsCfg)
	return nil
}
