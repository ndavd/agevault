package archive

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/ndavd/agevault/internal/utils"
)

func ZipDirectory(inputSource string, destinationWriter io.Writer) error {
	exists, isDir := utils.Exists(inputSource)
	if !exists || !isDir {
		return errors.New("source does not exist or is not a directory")
	}
	writer := zip.NewWriter(destinationWriter)
	defer writer.Close()
	return filepath.Walk(inputSource, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			path = fmt.Sprintf("%s%c", path, os.PathSeparator)
			_, err = writer.Create(path)
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()
		f, err := writer.Create(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(f, file)
		return err
	})
}

func UnZip(inputReader bytes.Reader, outputDestination string) error {
	reader, err := zip.NewReader(&inputReader, inputReader.Size())
	if err != nil {
		return err
	}
	destination, err := filepath.Abs(outputDestination)
	if err != nil {
		return err
	}
	for _, f := range reader.File {
		err := unzipFile(f, destination)
		if err != nil {
			return err
		}
	}
	return err
}

func unzipFile(f *zip.File, destination string) error {
	path := filepath.Join(destination, f.Name)
	if !strings.HasPrefix(path, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", path)
	}
	if f.FileInfo().IsDir() {
		if err := os.MkdirAll(path, os.ModePerm); err != nil {
			return err
		}
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	destinationFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()
	zippedFile, err := f.Open()
	if err != nil {
		return err
	}
	defer zippedFile.Close()
	_, err = io.Copy(destinationFile, zippedFile)
	return err
}
