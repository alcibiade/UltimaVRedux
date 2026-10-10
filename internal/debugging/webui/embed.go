// Package webui provides the embedded production assets for the debug workbench.
package webui

import (
	"embed"
	"io/fs"
)

//go:embed dist/*
var embeddedAssets embed.FS

// Assets returns the root of the embedded Vite production build.
func Assets() fs.FS {
	assets, err := fs.Sub(embeddedAssets, "dist")
	if err != nil {
		panic("opening embedded debug workbench assets: " + err.Error())
	}

	return assets
}
