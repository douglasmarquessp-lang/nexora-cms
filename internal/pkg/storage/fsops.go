package storage

import (
	"io/ioutil"
	"os"
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
