//go:build unix

package archive

import (
	"io/fs"
	"syscall"
)

func hasHardLinks(info fs.FileInfo) bool {
	st, ok := info.Sys().(*syscall.Stat_t)
	return ok && info.Mode().IsRegular() && st.Nlink > 1
}
