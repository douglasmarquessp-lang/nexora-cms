package storage

import (
	"io/ioutil"
	"os"
	"path/filepath"
)

func mkdirAll(dir string, perm os.FileMode) error {
	return os.MkdirAll(dir, perm)
}

func writeFile(path string, data []byte, perm os.FileMode) error {
	return ioutil.WriteFile(path, data, perm)
}

func readFile(path string) ([]byte, error) {
	return ioutil.ReadFile(path)
}

func removeFile(path string) error {
	return os.Remove(path)
}

func statFile(path string) (os.FileInfo, error) {
	return os.Stat(path)
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func ensureDir(path string) error {
	dir := filepath.Dir(path)
	return os.MkdirAll(dir, 0755)
}
