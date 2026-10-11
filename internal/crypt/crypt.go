package crypt

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"syscall"

	"filippo.io/age"
	"golang.org/x/term"
)

func EncryptToFile(destinationFilename string, data []byte, recipients ...age.Recipient) error {
	file, err := os.OpenFile(destinationFilename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	writeCloser, err := age.Encrypt(file, recipients...)
	if err != nil {
		return err
	}
	if _, err = writeCloser.Write(data); err != nil {
		return err
	}
	if err = writeCloser.Close(); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	return file.Close()
}

func DecryptToWriter(destinationWriter io.Writer, encryptedDataReader io.Reader, identity age.Identity) error {
	reader, err := age.Decrypt(encryptedDataReader, identity)
	if err != nil {
		return err
	}
	_, err = io.Copy(destinationWriter, reader)
	return err
}

func ReadSecret(label string, confirm bool) (string, error) {
	prefix := ""
	if confirm {
		prefix = "create "
	}
	fmt.Printf("%s%s: ", prefix, label)
	secretBytes, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	if len(secretBytes) == 0 {
		return "", errors.New("passphrase cannot be empty")
	}
	if confirm {
		fmt.Printf("confirm %s: ", label)
		confirmSecretBytes, err := term.ReadPassword(int(syscall.Stdin))
		fmt.Println()
		if err != nil {
			return "", err
		}
		if string(secretBytes) != string(confirmSecretBytes) {
			return "", fmt.Errorf("%s not matching", label)
		}
	}
	return string(secretBytes), nil
}

func Fingerprint(recipient fmt.Stringer) string {
	sum := sha256.Sum256([]byte(recipient.String()))
	return hex.EncodeToString(sum[:16])
}
