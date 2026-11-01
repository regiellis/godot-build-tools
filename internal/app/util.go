package app

import (
	"os"
	"path/filepath"
)

func filepathJoin(elem ...string) string {
	return filepath.Join(elem...)
}

func glob(pattern string) ([]string, error) {
	return filepath.Glob(pattern)
}

func getenvSimple(name string) string {
	return os.Getenv(name)
}
