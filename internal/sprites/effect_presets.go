// Package sprites provides preset effect pipelines for common visual configurations.
package sprites

// Preset pipelines for common NPC and entity visual configurations.
var (
	// StandardNPCPipeline - default for all visible NPCs.
	// Applies 18x18 border wrap + black outline for visibility.
	StandardNPCPipeline = Effect().AddBorder().AddOutline(BlackOutline).Build()

	// BorderOnlyPipeline - just the 18x18 border wrap, no outline.
	// Use when you want the border for rendering but no visual outline.
	BorderOnlyPipeline = Effect().AddBorder().Build()
)
