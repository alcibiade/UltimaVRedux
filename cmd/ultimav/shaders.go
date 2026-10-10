package main

import "github.com/hajimehoshi/ebiten/v2"

//nolint:gochecknoglobals // Rendering shares lazily initialized immutable shaders.
var (
	invertShader         *ebiten.Shader
	dungeonPaletteShader *ebiten.Shader
	finalePaletteShader  *ebiten.Shader
	finaleMoongateShader *ebiten.Shader
)

// getDungeonPaletteShader returns the cached dungeon palette remapping shader (ref:DNG-0009).
func getDungeonPaletteShader() *ebiten.Shader {
	if dungeonPaletteShader == nil {
		var err error
		dungeonPaletteShader, err = ebiten.NewShader([]byte(dungeonPaletteShaderSrc))
		if err != nil {
			panic(err)
		}
	}

	return dungeonPaletteShader
}

// getInvertShader returns the cached color inversion shader
func getInvertShader() *ebiten.Shader {
	if invertShader == nil {
		var err error
		invertShader, err = ebiten.NewShader([]byte(invertShaderSrc))
		if err != nil {
			panic(err)
		}
	}

	return invertShader
}

// getFinalePaletteShader returns the cached finale palette remapping shader (ref:DNG-0015).
func getFinalePaletteShader() *ebiten.Shader {
	if finalePaletteShader == nil {
		var err error
		finalePaletteShader, err = ebiten.NewShader([]byte(finalePaletteShaderSrc))
		if err != nil {
			panic(err)
		}
	}

	return finalePaletteShader
}

// getFinaleMoongateShader returns the cached blue-to-red EGA palette shader.
func getFinaleMoongateShader() *ebiten.Shader {
	if finaleMoongateShader == nil {
		var err error
		finaleMoongateShader, err = ebiten.NewShader([]byte(finaleMoongateShaderSrc))
		if err != nil {
			panic(err)
		}
	}

	return finaleMoongateShader
}

// Kage shader for color inversion with intensity control
const invertShaderSrc = `package main

var Intensity float

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	// Sample the original color
	original := imageSrc0UnsafeAt(texCoord)

	// Invert the RGB channels: inverted = 1.0 - original
	inverted := vec4(1.0 - original.r, 1.0 - original.g, 1.0 - original.b, original.a)

	// Mix between original and inverted based on intensity
	// Intensity 0.0 = original, Intensity 1.0 = fully inverted
	return mix(original, inverted, Intensity)
}
`

// The dungeon EGA palette mapping swaps palette-index bits 1 and 2
// in selected tiles.
//
//nolint:dupword // Repeated vector components encode exact EGA colours.
const dungeonPaletteShaderSrc = `package main

func sameDungeonColor(actual vec3, expected vec3) bool {
	delta := abs(actual - expected)
	return delta.r < 0.01 && delta.g < 0.01 && delta.b < 0.01
}

func dungeonColor(original vec4) vec4 {
	third := 1.0 / 3.0
	twoThirds := 2.0 / 3.0
	if sameDungeonColor(original.rgb, vec3(0, twoThirds, 0)) {
		return vec4(twoThirds, 0, 0, original.a)
	}
	if sameDungeonColor(original.rgb, vec3(0, twoThirds, twoThirds)) {
		return vec4(twoThirds, 0, twoThirds, original.a)
	}
	if sameDungeonColor(original.rgb, vec3(twoThirds, 0, 0)) {
		return vec4(0, twoThirds, 0, original.a)
	}
	if sameDungeonColor(original.rgb, vec3(twoThirds, 0, twoThirds)) {
		return vec4(0, twoThirds, twoThirds, original.a)
	}
	if sameDungeonColor(original.rgb, vec3(third, 1, third)) {
		return vec4(1, third, third, original.a)
	}
	if sameDungeonColor(original.rgb, vec3(third, 1, 1)) {
		return vec4(1, third, 1, original.a)
	}
	if sameDungeonColor(original.rgb, vec3(1, third, third)) {
		return vec4(third, 1, third, original.a)
	}
	if sameDungeonColor(original.rgb, vec3(1, third, 1)) {
		return vec4(third, 1, 1, original.a)
	}

	return original
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	remapped := dungeonColor(imageSrc0UnsafeAt(texCoord))
	return remapped * color
}
`

// The finale EGA palette mapping remaps palette nibbles in selected tiles (ref:DNG-0015).
//
//nolint:dupword // Repeated vector components encode exact EGA colours.
const finalePaletteShaderSrc = `package main

func sameColor(actual vec3, expected vec3) bool {
	delta := abs(actual - expected)
	return delta.r < 0.01 && delta.g < 0.01 && delta.b < 0.01
}

func finaleColor(original vec4) vec4 {
	third := 1.0 / 3.0
	twoThirds := 2.0 / 3.0
	if sameColor(original.rgb, vec3(0, 0, twoThirds)) {
		return vec4(twoThirds, 0, twoThirds, original.a)
	}
	if sameColor(original.rgb, vec3(0, twoThirds, 0)) ||
		sameColor(original.rgb, vec3(0, twoThirds, twoThirds)) {
		return vec4(twoThirds, 0, 0, original.a)
	}
	if sameColor(original.rgb, vec3(twoThirds, 0, 0)) ||
		sameColor(original.rgb, vec3(twoThirds, third, 0)) {
		return vec4(0, twoThirds, 0, original.a)
	}
	if sameColor(original.rgb, vec3(twoThirds, 0, twoThirds)) {
		return vec4(0, 0, twoThirds, original.a)
	}
	if sameColor(original.rgb, vec3(third, third, 1)) ||
		sameColor(original.rgb, vec3(third, 1, third)) ||
		sameColor(original.rgb, vec3(third, 1, 1)) {
		return vec4(1, third, third, original.a)
	}
	if sameColor(original.rgb, vec3(1, third, third)) {
		return vec4(third, 1, third, original.a)
	}
	if sameColor(original.rgb, vec3(1, third, 1)) {
		return vec4(third, third, 1, original.a)
	}

	return original
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	remapped := finaleColor(imageSrc0UnsafeAt(texCoord))
	return remapped * color
}
`

//nolint:dupword // Repeated vector components encode exact EGA colours.
const finaleMoongateShaderSrc = `package main

func sameMoongateColor(actual vec3, expected vec3) bool {
	delta := abs(actual - expected)
	return delta.r < 0.01 && delta.g < 0.01 && delta.b < 0.01
}

func Fragment(position vec4, texCoord vec2, color vec4) vec4 {
	original := imageSrc0UnsafeAt(texCoord)
	third := 1.0 / 3.0
	twoThirds := 2.0 / 3.0
	if sameMoongateColor(original.rgb, vec3(0, 0, twoThirds)) {
		return vec4(twoThirds, 0, 0, original.a) * color
	}
	if sameMoongateColor(original.rgb, vec3(third, third, 1)) {
		return vec4(1, third, third, original.a) * color
	}

	return original * color
}
`
