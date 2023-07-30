package clients

import (
	"context"

	s3Config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/emzola/bibliotheca/config"
)

type S3Client *s3.Client

// NewS3Client configures a new AWS S3 object storage client.
func NewS3Client(cfg config.Config) (S3Client, error) {
	creds := credentials.NewStaticCredentialsProvider(cfg.S3.AccessKeyID, cfg.S3.SecretAccessKey, "")
	awsCfg, err := s3Config.LoadDefaultConfig(context.TODO(), s3Config.WithCredentialsProvider(creds), s3Config.WithRegion(cfg.S3.Region))
	if err != nil {
		return nil, err
	}
	s3Client := s3.NewFromConfig(awsCfg)
	return s3Client, nil
}
