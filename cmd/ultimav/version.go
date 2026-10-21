package main

import "runtime/debug"

// version is stamped into release builds by scripts/package_release.py with
// -ldflags "-X main.version=..."; source builds fall back to module build info.
var version = ""

func runtimeGameVersion() string {
	if version != "" {
		return version
	}
	if buildInfo, ok := debug.ReadBuildInfo(); ok && buildInfo.Main.Version != "" && buildInfo.Main.Version != "(devel)" {
		return buildInfo.Main.Version
	}

	return developmentGameVersion
}
