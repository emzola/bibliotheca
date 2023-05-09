package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

const (
	ScopeCover = "cover"
	ScopeBook  = "book"
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

func (app *application) decodeJSON(w http.ResponseWriter, r *http.Request, dst interface{}) error {
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError
		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at character %d)", syntaxError.Offset)
		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")
		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)
		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)
		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytes)
		case errors.As(err, &invalidUnmarshalError):
			panic(err)
		default:
			return err
		}
	}
	err = dec.Decode(&struct{}{})
	if err != io.EOF {
		return errors.New("body must only contain a single JSON value")
	}
	return nil
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
func (app *application) detectMimeType(file multipart.File, fileHeader *multipart.FileHeader, scope string) ([]byte, *mimetype.MIME, error) {
	size := fileHeader.Size
	buffer := make([]byte, size)
	file.Read(buffer)
	mtype := mimetype.Detect(buffer)
	var supportedMediaType []string
	switch scope {
	case ScopeCover:
		supportedMediaType = []string{
			"image/jpeg",
			"image/png",
		}
	case ScopeBook:
		supportedMediaType = []string{
			"application/pdf",
			"application/epub+zip",
			"application/x-ms-reader",
			"application/x-mobipocket-ebook",
			"application/vnd.oasis.opendocument.text",
			"text/rtf",
			"image/vnd.djvu",
		}
	}
	if v := validator.Mime(mtype, supportedMediaType...); !v {
		return nil, nil, ErrInvalidMimeType
	}
	return buffer, mtype, nil
}

// uploadFileToS3 saves a form file to aws bucket and returns the key to the s3 file or an error if any.
func (app *application) uploadFileToS3(client *s3.Client, buffer []byte, mtype *mimetype.MIME, fileHeader *multipart.FileHeader, scope string) (string, error) {
	randomBytes := make([]byte, 16)
	_, err := rand.Read(randomBytes)
	if err != nil {
		return "", err
	}
	var uniqueFileName string
	uploader := manager.NewUploader(client)
	// TODO! Set uniqueFileName to include user id in the path e.g books/1/abc.pdf
	switch scope {
	case ScopeCover:
		uniqueFileName = "bookcovers/" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)) + filepath.Ext(fileHeader.Filename)
		_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
			Bucket:        aws.String(os.Getenv("AWS_S3_BUCKET")),
			Key:           aws.String(uniqueFileName),
			Body:          bytes.NewReader(buffer),
			ContentLength: *aws.Int64(fileHeader.Size),
			ContentType:   aws.String(mtype.String()),
		})
		uniqueFileName = "https://" + os.Getenv("AWS_S3_BUCKET") + ".s3." + os.Getenv("AWS_S3_REGION") + ".amazonaws.com/" + uniqueFileName
	case ScopeBook:
		uniqueFileName = "books/" + strings.ToLower(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(randomBytes)) + filepath.Ext(fileHeader.Filename)
		_, err = uploader.Upload(context.TODO(), &s3.PutObjectInput{
			Bucket:             aws.String(os.Getenv("AWS_S3_BUCKET")),
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
