package shredder

import (
	"crypto/rand"
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ndavd/agevault/internal/utils"
)

func ShredFile(path string, iterations int) error {
	file, err := os.OpenFile(path, os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	info, err := file.Stat()
	if err != nil {
		return err
	}
	random := make([]byte, info.Size())
	for i := 0; i < iterations; i++ {
		if _, err = rand.Read(random); err != nil {
			return err
		}
		if _, err = file.WriteAt(random, 0); err != nil {
			return err
		}
		if err = file.Sync(); err != nil {
			return err
		}
	}
	if err = file.Close(); err != nil {
		return err
	}
	return os.Remove(path)
}

func ShredDir(path string, iterations int) error {
	exists, isDir := utils.Exists(path)
	if !exists || !isDir {
		return errors.New("is not a directory or does not exist")
	}
	err := filepath.WalkDir(path, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && d.Type().IsRegular() {
			ShredFile(path, iterations)
		}
		return nil
	})
	if err != nil {
		return err
	}
	return os.RemoveAll(path)
}
