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

	storage := os.Getenv("STORAGE_PATH")
	if storage == "" {
		storage = "/mnt/files"
	}
	os.MkdirAll(storage, os.ModePerm)

	fileBase := strings.TrimSuffix(handler.Filename, filepath.Ext(handler.Filename))
	fileExt := filepath.Ext(handler.Filename)
	compressedName := fmt.Sprintf("%s_compressed%s.gz", fileBase, fileExt)
	u.deleteBeforeUpload(storage, fileBase)

	path := filepath.Join(storage, compressedName)
	dst, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	// 🧩 Gzip writer
	gzipWriter := gzip.NewWriter(dst)
	defer gzipWriter.Close()

	// Optional: store original filename in header
	gzipWriter.Name = handler.Filename

	// Compress contents
	if _, err := io.Copy(gzipWriter, file); err != nil {
		return path, err
	}

	return path, nil
}
