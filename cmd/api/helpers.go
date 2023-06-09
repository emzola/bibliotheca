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
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/emzola/bibliotheca/internal/data"
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
func (a *application) readIDParam(r *http.Request, idParam string) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())
	id, err := strconv.ParseInt(params.ByName(idParam), 10, 64)
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

// decodeJSON de-serializes JSON data into Go types.
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
// func (app *application) formatFileSize(size int64) string {
// 	var format string
// 	bytes := float64(size)
// 	kilobytes := bytes / 1024
// 	megabytes := kilobytes / 1024
// 	if bytes < 1_048_576 {
// 		format = fmt.Sprintf("%.1f KB", kilobytes)
// 	} else {
// 		format = fmt.Sprintf("%.1f MB", megabytes)
// 	}
// 	return format
// }

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

// downloadFileFromS3 downloads a file from the aws s3 bucket.
func (app *application) downloadFileFromS3(client *s3.Client, book *data.Book) error {
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
		Bucket: aws.String(os.Getenv("AWS_S3_BUCKET")),
		Key:    aws.String(book.S3FileKey),
	})
	if err != nil {
		return err
	}
	return nil
}

// readString returns a string value from the query string, or the provided default value
// if no matching key could be found.
func (app *application) readString(qs url.Values, key, defaultValue string) string {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	return s
}

// readCSV reads a string value from the query string and then splits it into a slice
// on the comma character. If no matching key could be found, it returns the provided
// default value.
func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	csv := qs.Get(key)
	if csv == "" {
		return defaultValue
	}
	return strings.Split(csv, ",")
}

// readInt reads a string value from the query string and converts it to an
// integer before returning. If no matching key could be found it returns the provided
// default value. If the value couldn't be converted to an integer, then we record an
// error message in the provided Validator instance.
func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)
	if s == "" {
		return defaultValue
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}
	return i
}

// background launches a background goroutine and recovers from panics inside
// the goroutine. It accepts an arbitrary function as a parameter and executes
// the function parameter inside the goroutine.
func (app *application) background(fn func()) {
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()
		defer func() {
			if err := recover(); err != nil {
				app.logger.PrintError(fmt.Errorf("%s", err), nil)
			}
		}()
		fn()
	}()
}

// fetchRemoteResource fetches data from a remote resource using a HTTP client
func (app *application) fetchRemoteResource(client *http.Client, url string, data interface{}) error {
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	r, err := client.Do(req)
	if err != nil {
		return err
	}
	defer r.Body.Close()
	bytes, err := io.ReadAll(r.Body)
	if err != nil {
		return err
	}
	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return err
	}
	return nil
}
