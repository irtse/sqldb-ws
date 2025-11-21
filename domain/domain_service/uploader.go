package domain_service

import (
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"regexp"
	"sqldb-ws/domain/schema"
	"sqldb-ws/domain/utils"
	"strings"
)

type Uploader struct {
	Domain utils.DomainITF
}

func NewUploader(d utils.DomainITF) *Uploader {
	return &Uploader{
		Domain: d,
	}
}

// add compression
func (u *Uploader) ApplyUpload(file multipart.File, handler *multipart.FileHeader) (string, error) {
	tableName := u.Domain.GetTable()
	if columnName, ok := u.Domain.GetParams().Get(utils.RootColumnsParam); !ok && len(strings.Split(columnName, ",")) > 0 {
		return "", errors.New("must have only one column field")
	} else {
		if path, err := u.upload(file, handler); err == nil {
			if sch, err := schema.GetSchema(schema.GetTablename(tableName)); err == nil && sch.HasField(columnName) {
				if f, _ := sch.GetField(columnName); !strings.Contains(f.Type, "upload") {
					return "", errors.New("must be a field of upload type")
				}
			}
			return path, nil
		} else {
			return "", err
		}
	}
}

func (u *Uploader) deleteBeforeUpload(storage string, fileName string) {
	pattern := `^` + fileName + `.*`
	regex, err := regexp.Compile(pattern)
	if err != nil {
		return
	}

	err = filepath.Walk(storage, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Match file name with regex
		if !info.IsDir() && regex.MatchString(info.Name()) {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to delete %s: %w", path, err)
			}
		}
		return nil
	})

	if err != nil {
		fmt.Println("Error walking directory:", err)
	}
}

func (u *Uploader) upload(file multipart.File, handler *multipart.FileHeader) (string, error) {
	defer file.Close()

	// Determine storage path
	storage := os.Getenv("STORAGE_PATH")
	if storage == "" {
		storage = "/mnt/files"
	}
	os.MkdirAll(storage, os.ModePerm)

	// Remove existing versions before uploading
	fileNameParts := strings.Split(handler.Filename, ".")
	baseName := fileNameParts[0]
	u.deleteBeforeUpload(storage, baseName)

	// Define compressed filename
	compressedPath := filepath.Join(storage, strings.Trim(handler.Filename, " ")+".gz")

	// Create destination file (compressed)
	dst, err := os.Create(compressedPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination: %w", err)
	}
	defer dst.Close()

	// Create gzip writer
	gw := gzip.NewWriter(dst)
	gw.Name = strings.Trim(handler.Filename, " ") // keep original name metadata
	fmt.Println("<"+compressedPath+">", "<"+strings.Trim(handler.Filename, " ")+">")
	defer gw.Close()

	// Copy uploaded content → gzip writer → file
	if _, err := io.Copy(gw, file); err != nil {
		return "", fmt.Errorf("failed to compress: %w", err)
	}

	return compressedPath, nil
}

func UncompressGzip(uncompressedPath string) (string, error) {
	// Ensure the file exists
	inFile, err := os.Open(fmt.Sprintf("%v.gz", strings.Trim(uncompressedPath, " ")))
	if err != nil {
		return "", fmt.Errorf("failed to open gzip file: %w", err)
	}
	defer inFile.Close()
	// Create a gzip reader
	gzipReader, err := gzip.NewReader(inFile)
	if err != nil {
		return "", fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()
	// Create destination file
	outFile, err := os.Create(uncompressedPath)
	if err != nil {
		return "", fmt.Errorf("failed to create uncompressed file: %w", err)
	}
	defer outFile.Close()

	// Copy data from gzip -> destination file
	if _, err := io.Copy(outFile, gzipReader); err != nil {
		return "", fmt.Errorf("failed to decompress: %w", err)
	}

	return uncompressedPath, nil
}

func DeleteUncompressed(path string) error {
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to delete temp file: %w", err)
	}
	return nil
}
