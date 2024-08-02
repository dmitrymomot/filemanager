package filemanager

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"path"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"golang.org/x/sync/errgroup"
)

const (
	// DefaultACL - default access control list for new objects
	DefaultACL = "public-read"
	// DefaultMaxFileSize - default max file size for multipart form upload 64MB
	DefaultMaxFileSize = 64 << 20 // 64MB
)

type (
	// FileManager represents a file manager that interacts with an S3 bucket.
	FileManager struct {
		s3          S3Client
		httpClient  httpClient
		cdnURL      string
		bucket      string
		basePath    string
		maxFileSize int64
	}

	// Config represents a storage client config
	Config struct {
		// FileManagerKey is the access key for the S3 client.
		FileManagerKey string

		// FileManagerSecret is the secret key for the S3 client.
		FileManagerSecret string

		// CDNURL is the URL of the content delivery network (CDN) for the S3 client.
		CDNURL string

		// BasePath is the base path for the S3 client.
		BasePath string

		// Endpoint is the endpoint URL for the S3 client.
		Endpoint string

		// Region is the AWS region for the S3 client.
		Region string

		// Bucket is the name of the S3 bucket for the client.
		Bucket string

		// MaxFileSize is the maximum allowed file size for the S3 client.
		MaxFileSize int64
	}

	// S3Client S3-compatible storage client interface
	S3Client interface {
		PutObjectWithContext(ctx aws.Context, input *s3.PutObjectInput, opts ...request.Option) (
			*s3.PutObjectOutput, error,
		)
		ListObjectsV2WithContext(
			ctx aws.Context,
			input *s3.ListObjectsV2Input,
			opts ...request.Option,
		) (*s3.ListObjectsV2Output, error)
		HeadObjectWithContext(ctx aws.Context, input *s3.HeadObjectInput, opts ...request.Option) (
			*s3.HeadObjectOutput, error,
		)
		DeleteObjectWithContext(
			ctx aws.Context,
			input *s3.DeleteObjectInput,
			opts ...request.Option,
		) (*s3.DeleteObjectOutput, error)
	}

	// httpClient interface
	httpClient interface {
		Get(url string) (resp *http.Response, err error)
	}

	// Option represents a file manager option function.
	Option func(*FileManager) error
)

// New creates a new instance of FileManager with the provided configuration.
// It initializes a storage session using the AWS SDK and returns a FileManager object.
// The FileManager object is used to interact with the specified S3 bucket.
func New(cnf Config) (*FileManager, error) {
	// create new storage session with the provided configuration
	newSession, err := session.NewSession(&aws.Config{
		Credentials:      credentials.NewStaticCredentials(cnf.FileManagerKey, cnf.FileManagerSecret, ""),
		Endpoint:         aws.String(cnf.Endpoint),
		Region:           aws.String(cnf.Region),
		S3ForcePathStyle: aws.Bool(false),
		DisableSSL:       aws.Bool(false),
	})
	if err != nil {
		return nil, errors.Join(ErrInvalidS3ClientConfig, err)
	}

	return NewWithOptions(
		WithS3Client(s3.New(newSession)),
		WithBucketName(cnf.Bucket),
		WithCDNURL(cnf.CDNURL),
		WithBasePath(cnf.BasePath),
		WithMaxFileSize(cnf.MaxFileSize),
	)
}

// NewWithOptions creates a new instance of FileManager with the provided options.
// It initializes a FileManager object with default values and then applies the provided options.
// The options are applied in the order they are provided.
// It returns the FileManager object and any error encountered during initialization and option application.
func NewWithOptions(opt ...Option) (*FileManager, error) {
	// create new file manager
	fm := &FileManager{
		httpClient:  http.DefaultClient,
		maxFileSize: DefaultMaxFileSize, // 64MB
		basePath:    "uploads",
	}

	// apply options
	for _, o := range opt {
		if err := o(fm); err != nil {
			return nil, errors.Join(ErrInvalidS3ClientConfig, err)
		}
	}

	// validate configuration
	if fm.bucket == "" {
		return nil, errors.Join(ErrInvalidS3ClientConfig, ErrMissedBucketName)
	}
	if fm.s3 == nil {
		return nil, errors.Join(ErrInvalidS3ClientConfig, ErrMissedS3Client)
	}
	if fm.cdnURL == "" {
		return nil, errors.Join(ErrInvalidS3ClientConfig, ErrMissedCDNURL)
	}

	return fm, nil
}

// Upload uploads a file to the S3 bucket.
// It takes the file content as a byte slice, the filename, and the content type as input parameters.
// It returns the URL of the uploaded file and any error encountered during the upload process.
func (fm *FileManager) Upload(ctx context.Context, file io.ReadSeeker, filename, contentType string) (string, error) {
	_, err := fm.s3.PutObjectWithContext(ctx, &s3.PutObjectInput{
		ACL:         aws.String(DefaultACL),
		Body:        file,
		ContentType: aws.String(contentType),
		Bucket:      aws.String(fm.bucket),
		Key:         aws.String(filename),
	})
	if err != nil {
		return "", errors.Join(ErrFailedToUploadFile, err)
	}

	return fm.fileAbsolutePath(filename), nil
}

// UploadFromMultipartForm uploads a file from a multipart form to the S3 bucket.
// It parses the multipart form, retrieves the file from the form data, and then
// uploads it to the S3 bucket. The file size is limited to 64MB.
//
// Parameters:
// - r: The HTTP request containing the multipart form data.
// - fieldName: The name of the field in the multipart form that contains the file.
//
// Returns:
// - string: The URL of the uploaded file in the S3 bucket.
// - error: An error if any occurred during the upload process.
func (fm *FileManager) UploadFromMultipartForm(r *http.Request, fieldName string) (string, error) {
	// Parse the multipart form
	// Limit the file size to 64MB
	if err := r.ParseMultipartForm(fm.maxFileSize); err != nil {
		return "", errors.Join(ErrFailedToUploadFileFromMultipartForm, err)
	}

	// Retrieve the file from form data
	file, header, err := r.FormFile(fieldName)
	if err != nil {
		return "", errors.Join(ErrFailedToUploadFileFromMultipartForm, err)
	}
	defer func(file multipart.File) {
		if err := file.Close(); err != nil {
			slog.ErrorContext(r.Context(), "failed to close file", "error", err)
		}
	}(file)

	// Upload the file to the S3 bucket
	result, err := fm.Upload(
		r.Context(),
		file,
		filepath.Base(header.Filename),
		header.Header.Get("Content-Type"),
	)
	if err != nil {
		return "", errors.Join(ErrFailedToUploadFileFromMultipartForm, err)
	}

	return result, nil
}

// UploadFromURL uploads a file from a URL to the S3 bucket.
// It takes the URL of the file as input and returns the URL of the uploaded file and any error encountered during the upload process.
func (fm *FileManager) UploadFromURL(ctx context.Context, fileURL string) (string, error) {
	// get file from URL
	resp, err := fm.httpClient.Get(fileURL)
	if err != nil {
		return "", errors.Join(ErrFailedToUploadFileFromURL, err)
	}
	defer func(Body io.ReadCloser) {
		if err := Body.Close(); err != nil {
			slog.ErrorContext(ctx, "failed to close response body", "error", err)
		}
	}(resp.Body)

	// read file to buffer
	buf := make([]byte, resp.ContentLength)
	if _, err := resp.Body.Read(buf); err != nil {
		return "", errors.Join(ErrFailedToUploadFileFromURL, err)
	}

	// upload file to storage
	result, err := fm.Upload(
		ctx,
		bytes.NewReader(buf),
		path.Base(fileURL),
		resp.Header.Get("Content-Type"),
	)
	if err != nil {
		return "", errors.Join(ErrFailedToUploadFileFromURL, err)
	}
	return result, nil
}

// Remove removes a file from the storage.
// It takes the fileURL as a parameter and returns an error if any.
// The fileURL is the URL of the file to be removed.
func (fm *FileManager) Remove(ctx context.Context, fileURL string) error {
	// remove file from storage
	return fm.remove(ctx, filenameFromURL(fm.cdnURL, fileURL))
}

// RemoveFilesFromDirectory removes all files from the specified directory in the storage.
// It retrieves all files from the storage, and then removes each file individually in parallel.
// If the directory does not exist or there are no files in the directory, it returns nil.
// If any error occurs during the removal process, it returns an error indicating the failure.
func (fm *FileManager) RemoveFilesFromDirectory(ctx context.Context, dir string) error {
	// get all files from storage
	resp, err := fm.s3.ListObjectsV2WithContext(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(fm.bucket),
		Prefix: aws.String(strings.Trim(dir, "/")),
	})
	if err := handleS3Error(err); err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil // directory does not exist, nothing to do
		}
		return errors.Join(ErrFailedToRemoveFiles, err)
	}

	// Create a new errgroup
	eg, _ := errgroup.WithContext(ctx)

	// remove all files from storage
	for _, file := range resp.Contents {
		// remove file from storage
		eg.Go(func(key string) func() error {
			return func() error {
				return fm.remove(ctx, key)
			}
		}(*file.Key))
	}

	// Wait for all the goroutines to finish
	if err := eg.Wait(); err != nil {
		return errors.Join(ErrFailedToRemoveFiles, err)
	}

	return nil
}

// remove removes a file from the storage.
// If the file does not exist, it returns nil.
// It returns an error if there was a problem removing the file.
func (fm *FileManager) remove(ctx context.Context, key string) error {
	// check if file exists
	if exists, _ := fm.fileExists(ctx, key); !exists {
		return nil // file does not exist, nothing to do
	}

	// remove file from storage
	if _, err := fm.s3.DeleteObjectWithContext(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(fm.bucket),
		Key:    aws.String(key),
	}); err != nil {
		return errors.Join(ErrFailedToRemoveFile, err)
	}

	return nil
}

// fileExists checks if a file exists in the S3 bucket.
// It takes a filepath as input and returns a boolean value indicating whether the file exists or not.
// If there is an error while checking the file existence, it returns an error.
func (fm *FileManager) fileExists(ctx context.Context, filepath string) (bool, error) {
	_, err := fm.s3.HeadObjectWithContext(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(fm.bucket),
		Key:    aws.String(filepath),
	})
	if err := handleS3Error(err); err != nil {
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		return false, errors.Join(ErrFailedToCheckIfFileExists, err)
	}
	return true, nil
}

// fileAbsolutePath returns the absolute path of a file in the S3 bucket.
// It takes the filename as input and returns the absolute path of the file.
func (fm *FileManager) fileAbsolutePath(filename string) string {
	return fmt.Sprintf("%s/%s/%s", fm.cdnURL, fm.basePath, strings.Trim(filename, "/"))
}
