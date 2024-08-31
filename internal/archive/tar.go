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

func UnTar(inputBuffer bytes.Buffer, outputDestination string) error {
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
		err = untarFile(reader, header, destination)
		if err != nil {
			return err
		}
	}
	return nil
}

func untarFile(r *tar.Reader, h *tar.Header, destination string) error {
	path := filepath.Join(destination, h.Name)
	if !strings.HasPrefix(path, filepath.Clean(destination)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", path)
	}
	if h.FileInfo().IsDir() {
		return os.MkdirAll(path, os.ModePerm)
	}
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	destinationFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, h.FileInfo().Mode())
	if err != nil {
		return err
	}
	defer destinationFile.Close()
	_, err = io.CopyN(destinationFile, r, h.Size)
	return err
}

func IsTar(r io.Reader) bool {
	h := make([]byte, 262)
	if _, err := io.ReadFull(r, h); err != nil {
		return false
	}
	fmt.Println(h, []byte("ustar"))
	return bytes.Equal(h[257:], []byte("ustar"))
}
