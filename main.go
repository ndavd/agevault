package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"strings"

	"filippo.io/age"
	"github.com/ndavd/agevault/internal/archive"
	"github.com/ndavd/agevault/internal/crypt"
	"github.com/ndavd/agevault/internal/shredder"
	"github.com/ndavd/agevault/internal/utils"
)

func Version() string {
	return "v1.0.0"
}

func Usage() {
	fmt.Printf("agevault %s", Version())
	fmt.Println()
	fmt.Println("lock/unlock directory with passphrase-protected identity file")
	fmt.Println("usage: agevault [vault-name] lock|unlock|keygen")
	os.Exit(0)
}

func errMsg(err error) {
	fmt.Printf("error: %s\n", err.Error())
	os.Exit(1)
}

func getIdentityFilename(trimmedVaultName string) (string, error) {
	identityFilename, err := utils.FileMatchInCwd(func(filename string) bool {
		return strings.HasSuffix(filename, fmt.Sprintf(".%s.key.age", trimmedVaultName))
	})
	if err != nil {
		return "", err
	}
	if identityFilename == "" {
		return "", errors.New("missing identity file")
	}
	return identityFilename, nil
}

func Keygen(trimmedVaultName string) (string, error) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", err
	}
	identityFilename := fmt.Sprintf(".%s.%s.key.age", identity.Recipient().String(), trimmedVaultName)
	pw, err := crypt.ReadSecret("identity passphrase", true)
	if err != nil {
		return "", err
	}
	scryptRecipient, err := age.NewScryptRecipient(pw)
	if err != nil {
		return "", err
	}
	if err = crypt.EncryptToFile(identityFilename, []byte(identity.String()), scryptRecipient); err != nil {
		return "", err
	}
	return identityFilename, nil
}

func Lock(vaultName string, trimmedVaultName string) (string, error) {
	encryptedFilename := fmt.Sprintf("%s.age", vaultName)
	encryptedExists, encryptedIsDir := utils.Exists(encryptedFilename)
	if encryptedExists && !encryptedIsDir {
		return "", errors.New("already locked")
	}
	identityFilename, err := getIdentityFilename(trimmedVaultName)
	if err != nil {
		return "", err
	}
	recipientString := strings.Split(identityFilename, ".")[1]
	recipient, err := age.ParseX25519Recipient(recipientString)
	if err != nil {
		return "", fmt.Errorf("could not read recipient: %s", err.Error())
	}
	vaultExists, vaultIsDir := utils.Exists(vaultName)
	if !vaultExists || !vaultIsDir {
		return "", fmt.Errorf("missing %s", vaultName)
	}
	var tarBuffer bytes.Buffer
	if err = archive.TarDirectory(vaultName, &tarBuffer); err != nil {
		return "", fmt.Errorf("could not tar: %s", err.Error())
	}
	if err = crypt.EncryptToFile(encryptedFilename, tarBuffer.Bytes(), recipient); err != nil {
		return "", fmt.Errorf("could not encrypt: %s", err.Error())
	}
	if err = shredder.ShredDir(vaultName, 3); err != nil {
		return "", fmt.Errorf("could not shred %s: %s", vaultName, err.Error())
	}
	return recipientString, nil
}

func Unlock(vaultName string, trimmedVaultName string) error {
	identityFilename, err := getIdentityFilename(trimmedVaultName)
	if err != nil {
		return err
	}
	vaultExists, vaultIsDir := utils.Exists(vaultName)
	if vaultExists && vaultIsDir {
		return errors.New("already unlocked")
	}
	encryptedVaultFilename := fmt.Sprintf("%s.age", vaultName)
	encryptedVault, err := os.Open(encryptedVaultFilename)
	if err != nil {
		return fmt.Errorf("missing encrypted %s: %s", vaultName, err.Error())
	}
	encryptedIdentity, err := os.Open(identityFilename)
	if err != nil {
		return fmt.Errorf("could not read identity file: %s", err.Error())
	}
	pw, err := crypt.ReadSecret(
		fmt.Sprintf("enter passphrase for identity file \"%s\"", identityFilename),
		false,
	)
	if err != nil {
		return err
	}
	scryptIdentity, err := age.NewScryptIdentity(pw)
	var identityBuffer bytes.Buffer
	if err = crypt.DecryptToWriter(&identityBuffer, encryptedIdentity, scryptIdentity); err != nil {
		return fmt.Errorf("bad passphrase: %s", err.Error())
	}
	identity, err := age.ParseIdentities(&identityBuffer)
	if err != nil || len(identity) != 1 {
		return fmt.Errorf("could not parse decrypted identity: %s", err.Error())
	}
	var tarBuffer bytes.Buffer
	err = crypt.DecryptToWriter(&tarBuffer, encryptedVault, identity[0])
	if err != nil {
		return fmt.Errorf("could not decrypt %s: %s", vaultName, err.Error())
	}
	tarReader := bytes.NewReader(tarBuffer.Bytes())
	if archive.IsZip(tarReader) {
		// NOTE: Ensure backwards compatibility with v1.0.0
		fmt.Println("found deprecated archiving format...")
		zipReader := bytes.NewReader(tarBuffer.Bytes())
		if err = archive.UnZip(*zipReader, "."); err != nil {
			return fmt.Errorf("could not unzip zipped %s: %s", vaultName, err.Error())
		}
	} else {
		if err = archive.UnTar(tarBuffer, "."); err != nil {
			return fmt.Errorf("could not untar tarred %s: %s", vaultName, err.Error())
		}
	}
	if err = shredder.ShredFile(encryptedVaultFilename, 1); err != nil {
		return fmt.Errorf("could not shred %s: %s", encryptedVaultFilename, err.Error())
	}
	return nil
}

func main() {
	args := os.Args[1:]

	if len(args) == 1 && args[0] == "--version" {
		fmt.Println(Version())
		os.Exit(0)
	}

	if len(args) != 2 {
		Usage()
	}

	action := args[1]
	vaultName := args[0]

	// NOTE: Useful for supporting dot-directories
	trimmedVaultName := strings.Trim(vaultName, ". ")

	if trimmedVaultName != "" && action == "keygen" {
		identityFilename, err := Keygen(trimmedVaultName)
		if err != nil {
			errMsg(err)
		}
		fmt.Printf("%s CREATED (do not change the filename)\n", identityFilename)
		return
	}

	if trimmedVaultName != "" && action == "lock" {
		recipientString, err := Lock(vaultName, trimmedVaultName)
		if err != nil {
			errMsg(err)
		}
		fmt.Printf("%s SECURED with %s\n", vaultName, recipientString)
		return
	}

	if trimmedVaultName != "" && action == "unlock" {
		err := Unlock(vaultName, trimmedVaultName)
		if err != nil {
			errMsg(err)
		}
		fmt.Printf("%s DECRYPTED\n", vaultName)
		return
	}

	errMsg(errors.New("bad args"))
}
