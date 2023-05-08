package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/emzola/bibliotheca/internal/validator"
	"github.com/gabriel-vasile/mimetype"
	"github.com/julienschmidt/httprouter"
)

type envelope map[string]interface{}

// readIDParam pulls the url id parameter from the request and returns it or an error if any.
func (a *application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())
	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil {
		return 0, err
	}
	return id, nil
}

// formatFileSize formats a bytes file size and returns a string with KB or MB suffix.
func (app *application) formatFileSize(size int64) string {
	var format string
	bytes := float64(size)
	kilobytes := bytes / 1024
	megabytes := kilobytes / 1024
	if bytes < 1_048_576 {
		format = fmt.Sprintf("%.1f KB", kilobytes)
	} else {
		format = fmt.Sprintf("%.1f MB", megabytes)
	}
	return format
}

// detectMimeType detects and validates the content type of a multipart file to ensure it is supported.
// This method is a workaround to the problem encountered when trying to detect content type directly
// inside createbookHandler (i.e. the multipart file becomes corrupted once it's parsed to detect its mime type).
func (app *application) detectMimeType(file multipart.File, fileHeader *multipart.FileHeader) ([]byte, *mimetype.MIME, error) {
	size := fileHeader.Size
	buffer := make([]byte, size)
	file.Read(buffer)
	mtype := mimetype.Detect(buffer)
	supportedMediaType := []string{
		"application/pdf",
		"application/epub+zip",
		"application/x-ms-reader",
		"application/x-mobipocket-ebook",
		"application/vnd.oasis.opendocument.text",
		"text/rtf",
		"image/vnd.djvu",
	}
	if v := validator.Mime(mtype, supportedMediaType...); !v {
		return nil, nil, ErrInvalidMimeType
	}
	return buffer, mtype, nil
}

// uploadFileToS3 saves a form file to aws bucket and returns the key to the s3 file or an error if any.
func (app *application) uploadFileToS3(client *s3.Client, buffer []byte, mtype *mimetype.MIME, fileHeader *multipart.FileHeader) (string, error) {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	uniqueFileName := "books/" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)) + filepath.Ext(fileHeader.Filename)
	uploader := manager.NewUploader(client)
	_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket:             aws.String(os.Getenv("AWS_S3_BUCKET")),
		Key:                aws.String(uniqueFileName),
		Body:               bytes.NewReader(buffer),
		ContentLength:      *aws.Int64(fileHeader.Size),
		ContentType:        aws.String(mtype.String()),
		ContentDisposition: aws.String("attachment"),
	})
	if err != nil {
		return "", err
	}
	return uniqueFileName, nil
}

// encodeJSON serializes data to JSON and writes the appropriate HTTP status code and headers if necessary.
func (app *application) encodeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.MarshalIndent(data, "", "\t")
	if err != nil {
		return err
	}
	js = append(js, '\n')
	for k, v := range headers {
		w.Header()[k] = v
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)
	return nil
}
