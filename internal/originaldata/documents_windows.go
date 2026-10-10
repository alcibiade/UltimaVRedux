//go:build windows

package originaldata

import "golang.org/x/sys/windows"

func platformDocumentsDirectory(_ string) (string, error) {
	return windows.KnownFolderPath(windows.FOLDERID_Documents, windows.KF_FLAG_DEFAULT)
}
