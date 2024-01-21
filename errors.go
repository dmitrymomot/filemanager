package filemanager

import "errors"

// Predefined errors.
var (
	ErrInvalidS3ClientConfig               = errors.New("invalid S3 client config")
	ErrFailedToUploadFile                  = errors.New("failed to upload file")
	ErrFailedToRemoveFile                  = errors.New("failed to remove file")
	ErrFailedToCheckIfFileExists           = errors.New("failed to check if file exists")
	ErrFailedToRemoveFiles                 = errors.New("failed to remove files")
	ErrMissedBucketName                    = errors.New("missed bucket name")
	ErrMissedS3Client                      = errors.New("missed S3 client")
	ErrMissedCDNURL                        = errors.New("missed CDN URL")
	ErrInvalidCDNURL                       = errors.New("invalid CDN URL")
	ErrFailedToUploadFileFromURL           = errors.New("failed to upload file from URL")
	ErrFailedToUploadFileFromMultipartForm = errors.New("failed to upload file from multipart form")
	ErrNotFound                            = errors.New("not found")
	ErrUnexpected                          = errors.New("unexpected error")
	ErrMissedHTTPClient                    = errors.New("missed HTTP client")
)
