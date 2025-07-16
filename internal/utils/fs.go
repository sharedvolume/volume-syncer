package utils

import "os"

// EnsureDir creates the directory if it does not exist
func EnsureDir(dir string) error {
	return os.MkdirAll(dir, 0755)
}
