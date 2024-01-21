# FileManager Package for S3-Compatible Storage

[![GitHub tag (latest SemVer)](https://img.shields.io/github/tag/dmitrymomot/filemanager)](https://github.com/dmitrymomot/filemanager)
[![Go Reference](https://pkg.go.dev/badge/github.com/dmitrymomot/filemanager.svg)](https://pkg.go.dev/github.com/dmitrymomot/filemanager)
[![License](https://img.shields.io/github/license/dmitrymomot/filemanager)](https://github.com/dmitrymomot/filemanager/blob/main/LICENSE)

[![Tests](https://github.com/dmitrymomot/filemanager/actions/workflows/tests.yml/badge.svg)](https://github.com/dmitrymomot/filemanager/actions/workflows/tests.yml)
[![CodeQL Analysis](https://github.com/dmitrymomot/filemanager/actions/workflows/codeql-analysis.yml/badge.svg)](https://github.com/dmitrymomot/filemanager/actions/workflows/codeql-analysis.yml)
[![GolangCI Lint](https://github.com/dmitrymomot/filemanager/actions/workflows/golangci-lint.yml/badge.svg)](https://github.com/dmitrymomot/filemanager/actions/workflows/golangci-lint.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/dmitrymomot/filemanager)](https://goreportcard.com/report/github.com/dmitrymomot/filemanager)

The `filemanager` package provides a convenient and efficient way to interact with AWS S3 or S3-compatible storage services in Go. It simplifies common operations such as uploading files from various sources, removing files, and managing content within S3 buckets.

## Features

- **File Uploads:** Upload files directly from byte slices, multipart forms, or URLs.
- **File Removal:** Remove individual files or all files within a directory.
- **S3 Integration:** Seamlessly integrates with AWS S3 and other S3-compatible services.
- **Content-Type Detection:** Automatically detects and sets the MIME type for uploaded files.
- **Customizable Settings:** Configurable for different bucket names, paths, and size limits.

## Installation

To install the package, use the following go get command:
```bash
go get github.com/dmitrymomot/filemanager
```

## Configuration

Before using the FileManager, set up your configuration with AWS credentials, bucket details, and other preferences:

```go
import "github.com/dmitrymomot/filemanager"

config := filemanager.Config{
    FileManagerKey:    "your-access-key-id",
    FileManagerSecret: "your-secret-access-key",
    CDNURL:            "your-cdn-url",
    BasePath:          "your-base-path",
    Endpoint:          "your-s3-endpoint",
    Region:            "your-region",
    Bucket:            "your-bucket-name",
    MaxFileSize:       64 << 20, // e.g., 64MB
}

fm, err := filemanager.New(config)
if err != nil {
    log.Fatal(err)
}
```

## Usage

### Uploading Files

Upload a file from an HTTP request:

```go
http.HandleFunc("/upload", func(w http.ResponseWriter, r *http.Request) {
    url, err := fm.UploadFromMultipartForm(r, "fileFieldName")
    if err != nil {
        // handle error
    }

    fmt.Fprintf(w, "File uploaded successfully: %s", url)
})
```

Upload a file from a URL:

```go
url, err := fm.UploadFromURL(context.Background(), "https://example.com/path/to/file")
if err != nil {
    // handle error
}
```

### Removing Files

Remove a specific file:

```go
err := fm.Remove(context.Background(), "fileURL")
if err != nil {
    // handle error
}
```

Remove all files in a directory:

```go
err := fm.RemoveFilesFromDirectory(context.Background(), "directoryPath")
if err != nil {
    // handle error
}
```

## Contributing

Contributions to the `filemanager` package are welcome! Here are some ways you can contribute:

- Reporting bugs
- Additional tests cases
- Suggesting enhancements
- Submitting pull requests
- Sharing the love by telling others about this project

## License

This package is licensed under the [Apache 2.0](LICENSE) - see the `LICENSE` file for details.