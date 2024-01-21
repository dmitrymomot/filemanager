package filemanager

import "strings"

// WithS3Client sets the S3 client.
func WithS3Client(client S3Client) FileManagerOption {
	return func(f *FileManager) error {
		if client == nil {
			return ErrMissedS3Client
		}
		f.s3 = client
		return nil
	}
}

// WithCustomHTTPClient sets the custom HTTP client.
func WithCustomHTTPClient(client httpClient) FileManagerOption {
	return func(f *FileManager) error {
		if client == nil {
			return ErrMissedHTTPClient
		}
		f.httpClient = client
		return nil
	}
}

// WithBucketName sets the bucket name.
func WithBucketName(bucketName string) FileManagerOption {
	return func(f *FileManager) error {
		if bucketName == "" {
			return ErrMissedBucketName
		}
		f.bucket = bucketName
		return nil
	}
}

// WithCDNURL sets the CDN URL.
func WithCDNURL(cdnURL string) FileManagerOption {
	return func(f *FileManager) error {
		cdnURL = strings.Trim(cdnURL, "/")
		if cdnURL == "" {
			return ErrMissedCDNURL
		} else if !strings.HasPrefix(cdnURL, "http://") && !strings.HasPrefix(cdnURL, "https://") {
			return ErrInvalidCDNURL
		}
		f.cdnURL = cdnURL
		return nil
	}
}

// WithBasePath sets the base path.
func WithBasePath(basePath string) FileManagerOption {
	return func(f *FileManager) error {
		f.basePath = strings.Trim(basePath, "/")
		return nil
	}
}

// WithMaxFileSize sets the max file size.
func WithMaxFileSize(maxFileSize int64) FileManagerOption {
	return func(f *FileManager) error {
		if maxFileSize <= 0 {
			maxFileSize = DefaultMaxFileSize
		}
		f.maxFileSize = maxFileSize
		return nil
	}
}
