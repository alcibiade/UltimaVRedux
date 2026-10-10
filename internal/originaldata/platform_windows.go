//go:build windows

package originaldata

import (
	"golang.org/x/sys/windows/registry"
)

func platformProviderSeeds() []Seed {
	var records []WindowsGOGRegistryRecord
	for _, root := range []registry.Key{registry.CURRENT_USER, registry.LOCAL_MACHINE} {
		for _, access := range []uint32{registry.WOW64_32KEY, registry.WOW64_64KEY} {
			record := WindowsGOGRegistryRecord{GamePath: "", UninstallPath: "", UninstallPublisher: ""}
			path := `SOFTWARE\GOG.com\Games\` + GOGGameID
			key, err := registry.OpenKey(root, path, registry.QUERY_VALUE|access)
			if err == nil {
				installPath, _, valueErr := key.GetStringValue("path")
				_ = key.Close()
				if valueErr == nil {
					record.GamePath = installPath
				}
			}
			uninstallPath := `SOFTWARE\Microsoft\Windows\CurrentVersion\Uninstall\` + GOGGameID + `_is1`
			key, err = registry.OpenKey(root, uninstallPath, registry.QUERY_VALUE|access)
			if err == nil {
				publisher, _, publisherErr := key.GetStringValue("Publisher")
				installPath, _, valueErr := key.GetStringValue("InstallLocation")
				_ = key.Close()
				if publisherErr == nil {
					record.UninstallPublisher = publisher
				}
				if valueErr == nil {
					record.UninstallPath = installPath
				}
			}
			records = append(records, record)
		}
	}

	return WindowsGOGSeeds(records)
}
