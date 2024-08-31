package archive

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

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
	return nil
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

func IsZip(r io.Reader) bool {
	h := make([]byte, 4)
	if _, err := io.ReadFull(r, h); err != nil {
		return false
	}
	const zipMagicNumber = "PK\x03\x04"
	return bytes.Equal(h, []byte(zipMagicNumber))
}
