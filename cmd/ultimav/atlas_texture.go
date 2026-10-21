package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/atlasterrain"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
	"github.com/hajimehoshi/ebiten/v2"
)

// atlasTextureOwner belongs to Game, not a GameScene or borrowed modal.
type atlasTextureOwner struct {
	identity string
	texture  *ebiten.Image
	root     string // Tests inject t.TempDir; production resolves the user-local cache.
}

func (o *atlasTextureOwner) load(terrain *references.LargeMapReference) (*ebiten.Image, error) {
	if terrain == nil {
		return nil, fmt.Errorf("atlas immutable terrain unavailable: %w", atlasterrain.ErrUnsupportedSource)
	}
	input, err := atlasterrain.Capture(func(x, y int) int {
		//nolint:gosec // Capture supplies only 0..255 for both axes.
		return int(terrain.GetSpriteIndex(references.Coordinate(x), references.Coordinate(y)))
	})
	if err != nil {
		return nil, fmt.Errorf("capture atlas terrain: %w", err)
	}
	artwork := sprites.TileAtlasPNG()
	identity := atlasterrain.Identity(input, artwork)
	if o.identity == identity && o.texture != nil {
		return o.texture, nil
	}
	if o.texture != nil {
		o.dispose()
	}
	root := o.root
	if root == "" {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return nil, fmt.Errorf("resolve atlas cache home: %w", homeErr)
		}
		root = filepath.Join(home, ".ultima_v_redux", "cache", "atlas")
	}
	result, err := atlasterrain.LoadOrGenerate(root, input, artwork)
	if err != nil {
		return nil, fmt.Errorf("load atlas geography: %w", err)
	}
	if result.Diagnostic != nil {
		logging.Warnf(logging.CategoryUI, "Atlas cache unavailable; retaining generated geography: %v", result.Diagnostic)
	}
	start := time.Now()
	o.texture = ebiten.NewImageFromImage(result.Image)
	o.identity = identity
	logging.Infof(logging.CategoryUI,
		"Atlas PNG %s (hit=%t, compose=%s, encode=%s, decode=%s, upload submission=%s; texture 64 MiB)",
		result.Path, result.CacheHit, result.ComposeTime, result.EncodeTime, result.DecodeTime, time.Since(start))
	// result and all CPU pixel/encoding buffers become unreachable on return.
	return o.texture, nil
}

func (o *atlasTextureOwner) dispose() {
	if o.texture != nil {
		o.texture.Deallocate()
		o.texture = nil
	}
	o.identity = ""
}
