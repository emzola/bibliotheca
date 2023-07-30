package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/emzola/bibliotheca/data"
	"github.com/gabriel-vasile/mimetype"
)

// detectMimeType detects and validates the content type of a multipart file to ensure it is supported.
// This method is a workaround to the problem encountered when trying to detect content type directly
// inside createbookHandler (i.e. the multipart file becomes corrupted once it's parsed to detect its mime type).
func (s *service) detectMimeType(file multipart.File, fileHeader *multipart.FileHeader) ([]byte, *mimetype.MIME, error) {
	size := fileHeader.Size
	buffer := make([]byte, size)
	_, err := file.Read(buffer)
	if err != nil {
		return nil, nil, err
	}
	mtype := mimetype.Detect(buffer)
	return buffer, mtype, nil
}

// uploadFileToS3 saves a form file to aws bucket and returns the key to the s3 file or an error if any.
func (s *service) uploadFileToS3(client *s3.Client, buffer []byte, mtype *mimetype.MIME, fileHeader *multipart.FileHeader, scope string) (string, error) {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	var uniqueFileName string
	uploader := manager.NewUploader(client)
	// TODO! Set uniqueFileName to include user id in the path e.g books/1/abc.pdf
	switch scope {
	case data.ScopeCover:
		uniqueFileName = "bookcovers/" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)) + filepath.Ext(fileHeader.Filename)
		_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
			Bucket:        aws.String(s.config.S3.Bucket),
			Key:           aws.String(uniqueFileName),
			Body:          bytes.NewReader(buffer),
			ContentLength: *aws.Int64(fileHeader.Size),
			ContentType:   aws.String(mtype.String()),
		})
		uniqueFileName = "https://" + s.config.S3.Bucket + ".s3." + s.config.S3.Region + ".amazonaws.com/" + uniqueFileName
	case data.ScopeBook:
		uniqueFileName = "books/" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)) + filepath.Ext(fileHeader.Filename)
		_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
			Bucket:             aws.String(s.config.S3.Bucket),
			Key:                aws.String(uniqueFileName),
			Body:               bytes.NewReader(buffer),
			ContentLength:      *aws.Int64(fileHeader.Size),
			ContentType:        aws.String(mtype.String()),
			ContentDisposition: aws.String("attachment"),
		})
	}
	if err != nil {
		return "", err
	}
	return uniqueFileName, nil
}

// downloadFileFromS3 downloads a file from the aws s3 bucket.
func (s *service) downloadFileFromS3(client *s3.Client, book *data.Book) error {
	// Set file name to follow the format: title (author[s]) ext
	// e.g Animal Farm (George Orwell).pdf
	author := strings.Join(book.Author, " ")
	filename := book.Title + " (" + author + ")" + "." + strings.ToLower(book.Extension)

	newFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newFile.Close()
	downloader := manager.NewDownloader(client)
	_, err = downloader.Download(context.TODO(), newFile, &s3.GetObjectInput{
		Bucket: aws.String(s.config.S3.Bucket),
		Key:    aws.String(book.S3FileKey),
	})
	if err != nil {
		return err
	}
	return nil
}

// background launches a background goroutine and recovers from panics inside
// the goroutine. It accepts an arbitrary function as a parameter and executes
// the function parameter inside the goroutine.
func (s *service) background(fn func()) {
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		defer func() {
			if err := recover(); err != nil {
				s.logger.PrintError(fmt.Errorf("%s", err), nil)
			}
		}()
		fn()
	}()
}

// fetchRemoteResource fetches data from a remote resource using a HTTP client
func (s *service) fetchRemoteResource(client *http.Client, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	r, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()
	return io.ReadAll(r.Body)

}
