// Package sprites provides preset effect pipelines for common visual configurations.
package sprites

// StandardNPCPipeline outlines the already-bordered artwork supplied by SpriteSheet.
//
//nolint:gochecknoglobals // Shared immutable map-unit preset; renderers only read its effects.
var StandardNPCPipeline = Effect().AddOutline(BlackOutline).Build()
