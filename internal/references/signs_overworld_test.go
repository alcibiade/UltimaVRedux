package references

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/test/originaldata"
	"github.com/stretchr/testify/require"
)

// TestSignReferences_CheckLargeMapSignTiles verifies that each world uses its
// own floor when correlating SIGNS.DAT records with readable terrain.
func TestSignReferences_CheckLargeMapSignTiles(t *testing.T) {
	cfg, err := config.LoadUltimaVConfiguration("", originaldata.Directory(t, "DATA.OVL", "SIGNS.DAT"))
	require.NoError(t, err)

	// Load signs
	signsPath := cfg.GetSignsDataFilePath()
	signRefs, err := NewSignReferences(signsPath)
	if err != nil {
		t.Fatalf("Error loading signs: %v", err)
	}

	signs := signRefs.GetSignsAtLocation(BritanniaUnderworld)
	for _, test := range []struct {
		name  string
		world World
		floor int
	}{
		{name: "overworld", world: OVERWORLD, floor: int(OVERWORLD)},
		{name: "underworld", world: UNDERWORLD, floor: int(UNDERWORLD)},
	} {
		t.Run(test.name, func(t *testing.T) {
			worldMap, loadErr := NewLargeMapReference(cfg, test.world)
			require.NoError(t, loadErr)

			checked := 0
			for _, sign := range signs {
				if sign.Floor != test.floor {
					continue
				}
				checked++
				tileIdx := worldMap.GetSpriteIndex(Coordinate(sign.X), Coordinate(sign.Y))
				require.Truef(t, tileIdx.IsSign(),
					"sign at (%d,%d) uses unreadable tile %d (0x%x)", sign.X, sign.Y, tileIdx, tileIdx)
			}
			require.Positive(t, checked)
		})
	}
}
