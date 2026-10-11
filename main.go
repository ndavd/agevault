package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"github.com/ndavd/agevault/internal/archive"
	"github.com/ndavd/agevault/internal/crypt"
	"github.com/ndavd/agevault/internal/recipients"
	"github.com/ndavd/agevault/internal/shredder"
	"github.com/ndavd/agevault/internal/utils"
)

func Version() string {
	return "v1.2.0"
}

func Usage() {
	fmt.Printf("agevault %s", Version())
	fmt.Println()
	fmt.Println("lock/unlock directory with passphrase-protected identity file")
	fmt.Println("usage: agevault [directory-name] lock|unlock|keygen")
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

// NOTE: Ensure backwards compatibility with versions before v1.2.0, where the recipient was in the identity filename
func migrateLegacyRecipient(vaultName string, trimmedVaultName string) error {
	missingErr := fmt.Errorf("missing %s, run keygen first", filepath.Join(vaultName, recipients.FileName))
	identityFilename, err := getIdentityFilename(trimmedVaultName)
	if err != nil {
		return missingErr
	}
	recipient, err := age.ParseX25519Recipient(strings.Split(identityFilename, ".")[1])
	if err != nil {
		return missingErr
	}
	if err = recipients.Append(vaultName, trimmedVaultName, recipient); err != nil {
		return fmt.Errorf("could not add recipient: %s", err.Error())
	}
	fmt.Printf("found deprecated recipient format, migrated to %s...\n", filepath.Join(vaultName, recipients.FileName))
	return nil
}

func Keygen(vaultName string, trimmedVaultName string, passphrase string) (string, string, error) {
	encryptedExists, encryptedIsDir := utils.Exists(fmt.Sprintf("%s.age", vaultName))
	if encryptedExists && !encryptedIsDir {
		return "", "", errors.New("vault is locked, unlock it first")
	}
	if err := os.MkdirAll(vaultName, 0o700); err != nil {
		return "", "", fmt.Errorf("could not create %s: %s", vaultName, err.Error())
	}
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", err
	}
	recipient := identity.Recipient()
	fingerprint := crypt.Fingerprint(recipient)
	identityFilename := fmt.Sprintf(".%s.%s.key.age", fingerprint, trimmedVaultName)
	scryptRecipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return "", "", err
	}
	if err = crypt.EncryptToFile(identityFilename, []byte(identity.String()), scryptRecipient); err != nil {
		return "", "", err
	}
	if err = recipients.Append(vaultName, trimmedVaultName, recipient); err != nil {
		return "", "", fmt.Errorf("could not add recipient: %s", err.Error())
	}
	return identityFilename, fingerprint, nil
}

func Lock(vaultName string, trimmedVaultName string) ([]age.Recipient, error) {
	encryptedFilename := fmt.Sprintf("%s.age", vaultName)
	encryptedExists, encryptedIsDir := utils.Exists(encryptedFilename)
	if encryptedExists && !encryptedIsDir {
		return nil, errors.New("already locked")
	}
	vaultExists, vaultIsDir := utils.Exists(vaultName)
	if !vaultExists || !vaultIsDir {
		return nil, fmt.Errorf("missing %s", vaultName)
	}
	recipientList, err := recipients.Read(vaultName)
	if errors.Is(err, os.ErrNotExist) {
		if err = migrateLegacyRecipient(vaultName, trimmedVaultName); err != nil {
			return nil, err
		}
		recipientList, err = recipients.Read(vaultName)
	}
	if err != nil {
		return nil, fmt.Errorf("could not read recipients: %s", err.Error())
	}
	var tarBuffer bytes.Buffer
	if err = archive.TarDirectory(vaultName, &tarBuffer); err != nil {
		return nil, fmt.Errorf("could not tar: %s", err.Error())
	}
	if err = crypt.EncryptToFile(encryptedFilename, tarBuffer.Bytes(), recipientList...); err != nil {
		return nil, fmt.Errorf("could not encrypt: %s", err.Error())
	}
	if err = shredder.ShredDir(vaultName, 3); err != nil {
		return nil, fmt.Errorf("could not shred %s: %s", vaultName, err.Error())
	}
	return recipientList, nil
}

func Unlock(vaultName string, trimmedVaultName string, getPassphrase func(identityFilename string) (string, error)) error {
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
	defer encryptedVault.Close()
	encryptedIdentity, err := os.Open(identityFilename)
	if err != nil {
		return fmt.Errorf("could not read identity file: %s", err.Error())
	}
	defer encryptedIdentity.Close()
	passphrase, err := getPassphrase(identityFilename)
	if err != nil {
		return err
	}
	scryptIdentity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return err
	}
	var identityBuffer bytes.Buffer
	if err = crypt.DecryptToWriter(&identityBuffer, encryptedIdentity, scryptIdentity); err != nil {
		return fmt.Errorf("bad passphrase: %s", err.Error())
	}
	identity, err := age.ParseX25519Identity(strings.TrimSpace(identityBuffer.String()))
	if err != nil {
		return fmt.Errorf("could not parse decrypted identity: %s", err.Error())
	}
	var tarBuffer bytes.Buffer
	err = crypt.DecryptToWriter(&tarBuffer, encryptedVault, identity)
	// NOTE: Close before shredding, Windows can't remove an open file
	encryptedVault.Close()
	if err != nil {
		return fmt.Errorf("could not decrypt %s: %s", vaultName, err.Error())
	}
	tarReader := bytes.NewReader(tarBuffer.Bytes())
	if archive.IsZip(tarReader) {
		// NOTE: Ensure backwards compatibility with v1.0.0
		fmt.Println("found deprecated archiving format...")
		zipReader := bytes.NewReader(tarBuffer.Bytes())
		if err = archive.UnZip(*zipReader, ".", vaultName); err != nil {
			return fmt.Errorf("could not unzip zipped %s: %s", vaultName, err.Error())
		}
	} else {
		if err = archive.UnTar(tarBuffer, ".", vaultName); err != nil {
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
	vaultName := filepath.Clean(args[0])

	// NOTE: Useful for supporting dot-directories
	trimmedVaultName := strings.Trim(vaultName, ". ")

	badArgsError := errors.New("bad args")

	if trimmedVaultName == "" {
		errMsg(badArgsError)
	}

	if action == "keygen" {
		pw, err := crypt.ReadSecret("identity passphrase", true)
		if err != nil {
			errMsg(err)
		}
		identityFilename, fingerprint, err := Keygen(vaultName, trimmedVaultName, pw)
		if err != nil {
			errMsg(err)
		}
		fmt.Printf("%s CREATED\n", identityFilename)
		fmt.Printf("%s ADDED to %s\n", fingerprint, filepath.Join(vaultName, recipients.FileName))
		return
	}

	if action == "lock" {
		recipientList, err := Lock(vaultName, trimmedVaultName)
		if err != nil {
			errMsg(err)
		}
		fmt.Printf("%s LOCKED with:\n", vaultName)
		for _, recipient := range recipientList {
			fmt.Printf("  %s\n", crypt.Fingerprint(recipient.(fmt.Stringer)))
		}
		return
	}

	if action == "unlock" {
		err := Unlock(vaultName, trimmedVaultName, func(identityFilename string) (string, error) {
			return crypt.ReadSecret(
				fmt.Sprintf("enter passphrase for identity file \"%s\"", identityFilename),
				false,
			)
		})
		if err != nil {
			errMsg(err)
		}
		fmt.Printf("%s UNLOCKED\n", vaultName)
		return
	}

	errMsg(badArgsError)
}
