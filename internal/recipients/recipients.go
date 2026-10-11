package recipients

import (
	"fmt"
	"os"
	"path/filepath"

	"filippo.io/age"
	"github.com/ndavd/agevault/internal/crypt"
	"github.com/ndavd/agevault/internal/utils"
)

const FileName = ".agevault-recipients"

// NOTE: Both *age.X25519Recipient and *age.HybridRecipient satisfy this
type Recipient interface {
	age.Recipient
	fmt.Stringer
}

func Append(vaultName string, trimmedVaultName string, recipient Recipient) error {
	path := filepath.Join(vaultName, FileName)
	exists, _ := utils.Exists(path)
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()
	if !exists {
		header := fmt.Sprintf("# agevault recipients for %s\n# lines starting with # are ignored\n", trimmedVaultName)
		if _, err = file.WriteString(header); err != nil {
			return err
		}
	}
	if _, err = fmt.Fprintf(file, "\n# fingerprint: %s\n%s\n", crypt.Fingerprint(recipient), recipient.String()); err != nil {
		return err
	}
	if err = file.Sync(); err != nil {
		return err
	}
	return file.Close()
}

func Read(vaultName string) ([]age.Recipient, error) {
	file, err := os.Open(filepath.Join(vaultName, FileName))
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return age.ParseRecipients(file)
}
