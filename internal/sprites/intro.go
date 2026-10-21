//nolint:mnd // Verified original archive dimensions, entry counts, and animation cadence.
package sprites

import (
	"bytes"
	_ "embed"
	"fmt"
	"image/png"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/bradhannah/UltimaVRedux/internal/originaldata/introarchive"
	"github.com/bradhannah/UltimaVRedux/pkg/graphics/legacy"
)

var (
	//go:embed assets/ReduxLogo.png
	reduxLogo []byte
)

type IntroSprites struct {
	OriginFrames     []*ebiten.Image
	Presents         *ebiten.Image
	ArticleA         *ebiten.Image
	Production       *ebiten.Image
	BritishSignature *ebiten.Image
	BritishPaths     [][]legacy.PathCommand
	Ultima16Logo     *ebiten.Image
	FlameFizzBase    *ebiten.Image
	ReduxLogo        *ebiten.Image
	FlameAnimation   *ImageAnimation
}

// LoadIntroSprites decodes original title graphics directly from the configured game data.
func LoadIntroSprites(dataDir string) (*IntroSprites, error) {
	archives, err := introarchive.LoadAndValidate(dataDir)
	if err != nil {
		return nil, fmt.Errorf("loading original intro archives: %w", err)
	}

	return newIntroSprites(archives.Title, archives.British, archives.Paths, archives.Ultima, archives.FlameMask)
}

func newIntroSprites(title, british, paths, ultima, flameMask legacy.Archive) (*IntroSprites, error) {
	archives := introarchive.Set{
		Title: title, British: british, Paths: paths, Ultima: ultima, FlameMask: flameMask,
	}
	if err := introarchive.Validate(archives); err != nil {
		return nil, fmt.Errorf("validating intro archives: %w", err)
	}

	return buildIntroSprites(archives)
}

func buildIntroSprites(archives introarchive.Set) (*IntroSprites, error) {
	title, british, paths := archives.Title, archives.British, archives.Paths
	ultima, flameMask := archives.Ultima, archives.FlameMask

	reduxImage, err := png.Decode(bytes.NewReader(reduxLogo))
	if err != nil {
		return nil, fmt.Errorf("decoding embedded Redux logo: %w", err)
	}
	originFrames := make([]*ebiten.Image, 7)
	for index := range originFrames {
		originFrames[index] = ebiten.NewImageFromImage(title.Entries[index].Image)
	}
	flameFrames := make([]*ebiten.Image, 4)
	for index := range flameFrames {
		flameFrames[index] = ebiten.NewImageFromImage(ultima.Entries[index+1].Image)
	}
	pathCommands := make([][]legacy.PathCommand, len(paths.Entries))
	for index, entry := range paths.Entries {
		pathCommands[index] = append([]legacy.PathCommand(nil), entry.Path...)
	}

	return &IntroSprites{
		OriginFrames:     originFrames,
		Presents:         ebiten.NewImageFromImage(title.Entries[7].Image),
		ArticleA:         ebiten.NewImageFromImage(title.Entries[8].Image),
		Production:       ebiten.NewImageFromImage(title.Entries[9].Image),
		BritishSignature: ebiten.NewImageFromImage(british.Entries[0].Image),
		BritishPaths:     pathCommands,
		Ultima16Logo:     ebiten.NewImageFromImage(ultima.Entries[0].Image),
		FlameFizzBase:    ebiten.NewImageFromImage(flameMask.Entries[0].Image),
		ReduxLogo:        ebiten.NewImageFromImage(reduxImage),
		FlameAnimation:   NewImageAnimation(flameFrames, 100),
	}, nil
}
