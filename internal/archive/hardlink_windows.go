//go:build windows

package archive

import (
	"io/fs"
)

func hasHardLinks(_ fs.FileInfo) bool { return false }
