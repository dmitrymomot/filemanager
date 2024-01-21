package filemanager

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
)

// filenameFromURL returns the filename from the URL.
// The url is the URL of the file.
func filenameFromURL(cdnURL, fileURL string) string {
	return strings.TrimPrefix(fileURL, cdnURL)
}

// handleS3Error handles S3 errors.
// It returns an error if the error is not nil.
// The err is the error to handle.
func handleS3Error(err error) error {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		case "NotFound": // s3.ErrCodeNoSuchKey does not work, aws is missing this error code so we hardwire a string
			return ErrNotFound
		default:
			return errors.Join(ErrUnexpected, err)
		}
	}
	return err
}
