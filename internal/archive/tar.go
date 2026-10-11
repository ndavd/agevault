package archive

import (
	"archive/tar"
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

func TarDirectory(inputSource string, destinationWriter io.Writer) error {
	exists, isDir := utils.Exists(inputSource)
	if !exists || !isDir {
		return errors.New("source does not exist or is not a directory")
	}
	writer := tar.NewWriter(destinationWriter)
	defer writer.Close()
	return filepath.Walk(inputSource, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if hasHardLinks(info) {
			return fmt.Errorf("%s has hard links", path)
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = path
		if err := writer.WriteHeader(header); err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		fileBytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = writer.Write(fileBytes)
		return err
	})
}

func UnTar(inputBuffer bytes.Buffer, outputDestination string, vaultName string) error {
	reader := tar.NewReader(&inputBuffer)
	destination, err := filepath.Abs(outputDestination)
	if err != nil {
		return err
	}
	for {
		header, err := reader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		err = untarFile(reader, header, destination, vaultName)
		if err != nil {
			return err
		}
	}
	return nil
}

func untarFile(r *tar.Reader, h *tar.Header, destination string, vaultName string) error {
	path := filepath.Join(destination, h.Name)
	vaultDir := filepath.Join(destination, vaultName)
	if path != vaultDir && !strings.HasPrefix(path, vaultDir+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", path)
	}
	if h.FileInfo().IsDir() {
		return os.MkdirAll(path, 0o700)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	destinationFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, h.FileInfo().Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()
	if _, err = io.CopyN(destinationFile, r, h.Size); err != nil {
		return err
	}
	if err = destinationFile.Sync(); err != nil {
		return err
	}
	return destinationFile.Close()
}
