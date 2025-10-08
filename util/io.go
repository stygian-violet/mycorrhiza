package util

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type FileReader interface {
	ReadFile(path string) ([]byte, error)
}

type FileServer interface {
	ServeFile(w http.ResponseWriter, rq *http.Request, path string)
}

type FileReadServer interface {
	FileReader
	FileServer
}

func WriteFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), os.ModeDir|0770); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0660)
}

func CopyFile(path string, file io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(path), os.ModeDir|0770); err != nil {
		return err
	}
	dst, err := os.OpenFile(path, os.O_WRONLY | os.O_CREATE | os.O_TRUNC, 0660)
	if err != nil {
		return err
	}
	_, err = io.Copy(dst, file)
	dst.Close()
	return err
}

func FileSize(file io.Seeker) (int64, error) {
	size, err := file.Seek(0, io.SeekEnd)
	if err != nil {
		return 0, err
	}
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return 0, err
	}
	return size, nil
}

// HTTP404Page writes a 404 error in the status, needed when no content is found on the page.
// TODO: demolish
func HTTP404Page(w http.ResponseWriter, page string) {
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	w.WriteHeader(http.StatusNotFound)
	_, _ = w.Write([]byte(page))
}

// HTTP200Page wraps some frequently used things for successful 200 responses.
// TODO: demolish
func HTTP200Page(w http.ResponseWriter, page string) {
	w.Header().Set("Content-Type", "text/html;charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(page))
}
