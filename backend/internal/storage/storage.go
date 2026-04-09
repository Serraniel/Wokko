package storage

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

type Storage struct {
	BasePath string
}

func New(basePath string) *Storage {
	if err := os.MkdirAll(basePath, 0755); err != nil {
		panic(fmt.Sprintf("storage: mkdir %s: %v", basePath, err))
	}
	return &Storage{BasePath: basePath}
}

func (s *Storage) SavePhoto(file *multipart.FileHeader) (string, error) {
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".webp": true}
	if !allowed[ext] {
		return "", fmt.Errorf("unsupported file type: %s", ext)
	}
	if file.Size > 10*1024*1024 { // 10MB limit
		return "", fmt.Errorf("file too large: max 10MB")
	}
	filename := uuid.NewString() + ext
	dst, err := os.Create(filepath.Join(s.BasePath, filename))
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("open upload: %w", err)
	}
	defer src.Close()
	if _, err := io.Copy(dst, src); err != nil {
		return "", fmt.Errorf("save file: %w", err)
	}
	return filename, nil
}

func (s *Storage) DeletePhoto(filename string) {
	path := filepath.Join(s.BasePath, filepath.Base(filename))
	_ = os.Remove(path)
}
