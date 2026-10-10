// Package conversationpatchdata exposes tracked conversation patches to the runtime.
package conversationpatchdata

import (
	"embed"
	"io/fs"
)

// Files contains the cumulative semantic patches. Non-patch source and scaffold
// files are ignored by the catalog loader.
//
//go:embed all:*
var Files embed.FS

var _ fs.FS = Files
