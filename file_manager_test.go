package filemanager_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/dmitrymomot/filemanager"
)

// default access control list for new objects
const defaultACL = "public-read"

// RoundTripFunc is a helper type for creating a custom http.RoundTripper.
type RoundTripFunc func(req *http.Request) (*http.Response, error)

// RoundTrip implements the http.RoundTripper interface.
func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// mockS3Client is a mock implementation of the S3Client interface.
type mockS3Client struct {
	mock.Mock
}

func (m *mockS3Client) PutObjectWithContext(
	ctx aws.Context,
	input *s3.PutObjectInput,
	opts ...request.Option,
) (*s3.PutObjectOutput, error) {
	args := m.Called(ctx, input, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

func (m *mockS3Client) ListObjectsV2WithContext(
	ctx aws.Context,
	input *s3.ListObjectsV2Input,
	opts ...request.Option,
) (*s3.ListObjectsV2Output, error) {
	args := m.Called(ctx, input, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.ListObjectsV2Output), args.Error(1)
}

func (m *mockS3Client) HeadObjectWithContext(
	ctx aws.Context,
	input *s3.HeadObjectInput,
	opts ...request.Option,
) (*s3.HeadObjectOutput, error) {
	args := m.Called(ctx, input, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.HeadObjectOutput), args.Error(1)
}

func (m *mockS3Client) DeleteObjectWithContext(
	ctx aws.Context,
	input *s3.DeleteObjectInput,
	opts ...request.Option,
) (*s3.DeleteObjectOutput, error) {
	args := m.Called(ctx, input, opts)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*s3.DeleteObjectOutput), args.Error(1)
}

func TestUpload(t *testing.T) {
	fileContent := bytes.NewReader([]byte("test content"))
	filename := "testfile.txt"
	contentType := "text/plain"
	bucket := "test-bucket"
	cdn := "https://cdn.example.com"
	baseURL := "/uploads"
	maxFileSize := int64(32 << 20) // 32 MB

	mockS3 := new(mockS3Client)
	mockS3.On("PutObjectWithContext", mock.Anything, &s3.PutObjectInput{
		ACL:         aws.String(defaultACL),
		Body:        fileContent,
		ContentType: aws.String(contentType),
		Bucket:      aws.String(bucket),
		Key:         aws.String(filename),
	}, mock.Anything).Return(&s3.PutObjectOutput{}, nil)

	// Create a new FileManager instance and inject the mock.
	fm, err := filemanager.NewWithOptions(
		filemanager.WithS3Client(mockS3),
		filemanager.WithBucketName(bucket),
		filemanager.WithCDNURL(cdn),
		filemanager.WithBasePath(baseURL),
		filemanager.WithMaxFileSize(maxFileSize),
	)
	require.NoError(t, err)

	url, err := fm.Upload(context.Background(), fileContent, filename, contentType)
	require.NoError(t, err)
	require.Contains(t, url, filename)
	require.Equal(t, "https://cdn.example.com/uploads/"+filename, url)

	// Assert that the expectations were met.
	mockS3.AssertExpectations(t)
}

// ... More test cases ...
func TestUploadFromMultipartForm(t *testing.T) {
	fileContent := []byte("test content")
	filename := "testfile.txt"
	fieldName := "file"
	bucket := "test-bucket"
	cdn := "https://cdn.example.com"
	baseURL := "/uploads"
	maxFileSize := int64(32 << 20) // 32 MB

	// Create a mock HTTP request with a multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, err := writer.CreateFormFile(fieldName, filename)
	require.NoError(t, err)
	_, err = file.Write(fileContent)
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	req, err := http.NewRequest("POST", "/upload", body)
	require.NoError(t, err)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	mockS3 := new(mockS3Client)
	mockS3.On("PutObjectWithContext", mock.Anything, mock.AnythingOfType("*s3.PutObjectInput"), mock.Anything).
		Return(&s3.PutObjectOutput{}, nil)

	// Create a new FileManager instance and inject the mock.
	fm, err := filemanager.NewWithOptions(
		filemanager.WithS3Client(mockS3),
		filemanager.WithBucketName(bucket),
		filemanager.WithCDNURL(cdn),
		filemanager.WithBasePath(baseURL),
		filemanager.WithMaxFileSize(maxFileSize),
	)
	require.NoError(t, err)

	// Call the UploadFromMultipartForm function
	url, err := fm.UploadFromMultipartForm(req, "file")
	require.NoError(t, err)
	require.Equal(t, "https://cdn.example.com/uploads/testfile.txt", url)

	// Assert that the expectations were met
	mockS3.AssertExpectations(t)
}

func TestUploadFromURL(t *testing.T) {
	bucket := "test-bucket"
	cdn := "https://cdn.example.com"
	baseURL := "/uploads"
	maxFileSize := int64(32 << 20) // 32 MB

	fileURL := "https://example.com/testfile.txt"
	ctx := context.Background()

	// Create a mock HTTP response with file content
	fileContent := []byte("test content")
	resp := &http.Response{
		Body:       io.NopCloser(bytes.NewReader(fileContent)),
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		StatusCode: http.StatusOK,
	}

	// Create a mock HTTP client that returns the response
	mockHTTPClient := &http.Client{
		Transport: RoundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.String() == fileURL {
				return resp, nil
			}
			return nil, errors.New("unexpected URL")
		}),
	}

	mockS3 := new(mockS3Client)
	mockS3.On("PutObjectWithContext", mock.Anything, mock.AnythingOfType("*s3.PutObjectInput"), mock.Anything).
		Return(&s3.PutObjectOutput{}, nil)

	// Create a new FileManager instance and inject the mock.
	fm, err := filemanager.NewWithOptions(
		filemanager.WithS3Client(mockS3),
		filemanager.WithBucketName(bucket),
		filemanager.WithCDNURL(cdn),
		filemanager.WithBasePath(baseURL),
		filemanager.WithMaxFileSize(maxFileSize),
		filemanager.WithCustomHTTPClient(mockHTTPClient),
	)
	require.NoError(t, err)

	// Call the UploadFromURL function
	result, err := fm.UploadFromURL(ctx, fileURL)
	require.NoError(t, err)
	require.Equal(t, "https://cdn.example.com/uploads/testfile.txt", result)

	// Assert that the expectations were met
	mockS3.AssertExpectations(t)
}
