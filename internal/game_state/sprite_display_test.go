package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestGetDisplaySpriteAtPosition_LadderLogic tests the core ladder sprite logic
// that was causing both Bug #1 (avatar sprite replacement) and Bug #2 (tile persistence).
func TestGetDisplaySpriteAtPosition_LadderLogic(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(15, 15). // Start away from ladder
		WithSystemCallbacks().
		Build()

	if gameState == nil {
		t.Skip("Skipping test - real game data not available")
		return
	}

	// Test positions
	ladderUpPos := references.Position{X: 1, Y: 26}   // Known ladder position in Britain
	avatarAwayPos := references.Position{X: 1, Y: 27} // Away from ladder

	t.Run("AvatarOnLadderUp_ShowsKlimbLadderUp", func(t *testing.T) {
		// Position avatar on the ladder
		gameState.MapState.PlayerLocation.Position = ladderUpPos

		sprite := gameState.GetDisplaySpriteAtPosition(ladderUpPos)
		if sprite != indexes.KlimbLadderUp {
			t.Errorf("Expected KlimbLadderUp (%d), got %d", indexes.KlimbLadderUp, sprite)
		}
		t.Logf("✅ Avatar on ladder shows KlimbLadderUp sprite")
	})

	t.Run("AvatarAwayFromLadder_ShowsBaseTile", func(t *testing.T) {
		// Position avatar away from the ladder
		gameState.MapState.PlayerLocation.Position = avatarAwayPos

		sprite := gameState.GetDisplaySpriteAtPosition(ladderUpPos)
		// Should show base ladder tile, not climbing sprite
		if sprite == indexes.KlimbLadderUp || sprite == indexes.KlimbLadderDown {
			t.Errorf("Expected base ladder tile, got KlimbLadder sprite %d", sprite)
		}
		t.Logf("✅ Ladder without avatar shows base tile: %d", sprite)
	})

	t.Run("AvatarMovementTransition", func(t *testing.T) {
		// Test the exact scenario that was broken before

		// 1. Avatar starts on ladder
		gameState.MapState.PlayerLocation.Position = ladderUpPos
		spriteOnLadder := gameState.GetDisplaySpriteAtPosition(ladderUpPos)

		// 2. Avatar moves away
		gameState.MapState.PlayerLocation.Position = avatarAwayPos
		spriteAfterMove := gameState.GetDisplaySpriteAtPosition(ladderUpPos)

		// Validate the transition
		if spriteOnLadder != indexes.KlimbLadderUp {
			t.Errorf("Step 1 failed: Expected KlimbLadderUp when avatar on ladder")
		}

		if spriteAfterMove == indexes.KlimbLadderUp || spriteAfterMove == indexes.KlimbLadderDown {
			t.Errorf("Step 2 failed: Ladder still shows KlimbLadder (%d) after avatar moved away", spriteAfterMove)
		}

		t.Logf("✅ Transition works: ladder → %d, after move → %d", spriteOnLadder, spriteAfterMove)
	})
}

// TestGetAvatarSpriteAtPosition tests avatar-specific sprite replacement logic.
func TestGetAvatarSpriteAtPosition(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	if gameState == nil {
		t.Skip("Skipping test - real game data not available")
		return
	}

	testCases := []struct {
		name           string
		baseTileIndex  indexes.SpriteIndex
		expectedSprite indexes.SpriteIndex
		description    string
	}{
		{
			name:           "LadderUp",
			baseTileIndex:  indexes.LadderUp,
			expectedSprite: indexes.KlimbLadderUp,
			description:    "Avatar on up ladder should show climbing up sprite",
		},
		{
			name:           "LadderDown",
			baseTileIndex:  indexes.LadderDown,
			expectedSprite: indexes.KlimbLadderDown,
			description:    "Avatar on down ladder should show climbing down sprite",
		},
		{
			name:           "LeftBed",
			baseTileIndex:  indexes.LeftBed,
			expectedSprite: indexes.SleepingInBed,
			description:    "Avatar on bed should show sleeping sprite",
		},
		{
			name:           "Manacles",
			baseTileIndex:  indexes.Manacles,
			expectedSprite: indexes.ManaclesPrisoner,
			description:    "Avatar on manacles should show prisoner sprite",
		},
		{
			name:           "Stocks",
			baseTileIndex:  indexes.Stocks,
			expectedSprite: indexes.PersonStocks1,
			description:    "Avatar on stocks should show person in stocks sprite",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a mock tile for testing
			pos := references.Position{X: 10, Y: 10}

			// Test the sprite replacement logic directly
			result := gameState.GetNPCSpriteReplacement(tc.baseTileIndex, indexes.AvatarKeyIndex, pos).Sprite

			if result != tc.expectedSprite {
				t.Errorf("%s: Expected %d, got %d", tc.description, tc.expectedSprite, result)
			} else {
				t.Logf("✅ %s", tc.description)
			}
		})
	}
}

func TestCarpetMountedAvatarUsesTerrainPresentation(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		Build()
	position := gameState.MapState.PlayerLocation.Position
	layer := gameState.GetLayeredMapByCurrentLocation()

	for _, direction := range []references.Direction{references.Left, references.Right} {
		t.Run(direction.GetDirectionCompassName(), func(t *testing.T) {
			carpet := map_units.NewNPCFriendlyVehiceNewRef(references.CarpetVehicle, position, 0)
			carpet.GetVehicleDetails().SetPartyVehicleDirection(direction)
			gameState.PartyVehicle = *carpet
			vehicleBefore := gameState.PartyVehicle.GetVehicleDetails().Snapshot()
			locationBefore := gameState.MapState.PlayerLocation
			rngBefore := gameState.GetRNG().Calls()

			for tile, want := range map[indexes.SpriteIndex]indexes.SpriteIndex{
				indexes.LeftBed:    indexes.SleepingInBed,
				indexes.LadderUp:   indexes.KlimbLadderUp,
				indexes.LadderDown: indexes.KlimbLadderDown,
			} {
				layer.SetTileByLayer(map_state.MapOverrideLayer, &position, tile)
				presentation := gameState.GetAvatarSpriteAtPosition(position)
				if presentation.Sprite != want || !presentation.BackgroundIntegrated {
					t.Fatalf("tile %d: got %+v, want sprite %d with integrated background", tile, presentation, want)
				}
			}

			wantCarpet := indexes.SpriteIndex(indexes.RidingMagicCarpetRight)
			if direction == references.Left {
				wantCarpet = indexes.RidingMagicCarpetLeft
			}
			for _, tile := range []indexes.SpriteIndex{indexes.RightBed, indexes.BrickFloor} {
				layer.SetTileByLayer(map_state.MapOverrideLayer, &position, tile)
				if got := gameState.GetAvatarSpriteAtPosition(position); got.Sprite != wantCarpet || got.BackgroundIntegrated {
					t.Fatalf("tile %d: got %+v, want carpet %d", tile, got, wantCarpet)
				}
			}
			if vehicleBefore != gameState.PartyVehicle.GetVehicleDetails().Snapshot() ||
				locationBefore != gameState.MapState.PlayerLocation || rngBefore != gameState.GetRNG().Calls() {
				t.Fatal("sprite selection mutated gameplay state")
			}
		})
	}
}

func TestDroppedCarpetDoesNotUseBedReplacement(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	position := references.Position{X: 10, Y: 10}
	for _, tile := range []indexes.SpriteIndex{indexes.LeftBed, indexes.RightBed, indexes.BrickFloor} {
		presentation := gameState.GetNPCSpriteReplacement(tile, indexes.Carpet2, position)
		if presentation.Sprite != indexes.Carpet2 || presentation.BackgroundIntegrated || presentation.SuppressTerrainUnderlay {
			t.Fatalf("tile %d: dropped carpet presentation = %+v", tile, presentation)
		}
	}
	if got := gameState.GetNPCSpriteReplacement(indexes.LeftBed, indexes.AvatarKeyIndex, position); got.Sprite != indexes.SleepingInBed {
		t.Fatalf("eligible actor on left bed = %+v", got)
	}
}

// TestChairSpriteReplacement tests chair-specific avatar sprite logic in GetNPCSpriteReplacement.
func TestChairSpriteReplacement(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	if gameState == nil {
		t.Skip("Skipping test - real game data not available")
		return
	}

	// ChairBack naming refers to which direction the BACK of the chair faces
	// So ChairBackRight has its back facing right, meaning the person faces LEFT
	// And ChairBackLeft has its back facing left, meaning the person faces RIGHT
	testCases := []struct {
		chairType      indexes.SpriteIndex
		expectedSprite indexes.SpriteIndex
		description    string
	}{
		{
			chairType:      indexes.ChairBackRight,
			expectedSprite: indexes.SitChairLeft, // Back faces right → person faces left
			description:    "Chair with back facing right should show left-facing sitting sprite",
		},
		{
			chairType:      indexes.ChairBackLeft,
			expectedSprite: indexes.SitChairRight, // Back faces left → person faces right
			description:    "Chair with back facing left should show right-facing sitting sprite",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			pos := references.Position{X: 10, Y: 10}
			// Test the chair logic via GetNPCSpriteReplacement (where it's now located)
			result := gameState.GetNPCSpriteReplacement(tc.chairType, indexes.AvatarKeyIndex, pos).Sprite

			if result != tc.expectedSprite {
				t.Errorf("Expected %d, got %d", tc.expectedSprite, result)
			} else {
				t.Logf("✅ %s", tc.description)
			}
		})
	}
}

// TestSpriteLogic_PositionAwareness tests that sprite logic correctly considers position.
func TestSpriteLogic_PositionAwareness(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(10, 10). // Start at neutral position
		WithSystemCallbacks().
		Build()

	if gameState == nil {
		t.Skip("Skipping test - real game data not available")
		return
	}

	ladderPos := references.Position{X: 1, Y: 26}
	otherPos := references.Position{X: 5, Y: 5}

	t.Run("SameTileDifferentResults", func(t *testing.T) {
		// Test that the same tile type gives different results based on avatar position

		// Avatar NOT at ladder position
		gameState.MapState.PlayerLocation.Position = otherPos
		spriteWhenAway := gameState.GetDisplaySpriteAtPosition(ladderPos)

		// Avatar AT ladder position
		gameState.MapState.PlayerLocation.Position = ladderPos
		spriteWhenPresent := gameState.GetDisplaySpriteAtPosition(ladderPos)

		// Results should be different
		if spriteWhenAway == spriteWhenPresent {
			t.Errorf("Position awareness failed: same sprite (%d) regardless of avatar position", spriteWhenAway)
		} else {
			t.Logf("✅ Position awareness works: away=%d, present=%d", spriteWhenAway, spriteWhenPresent)
		}

		// Avatar present should show KlimbLadder
		if spriteWhenPresent != indexes.KlimbLadderUp && spriteWhenPresent != indexes.KlimbLadderDown {
			t.Errorf("Avatar at ladder should show KlimbLadder sprite, got %d", spriteWhenPresent)
		}
	})
}

// TestSpriteLogic_Regression validates that refactor doesn't break existing functionality.
func TestSpriteLogic_Regression(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	if gameState == nil {
		t.Skip("Skipping test - real game data not available")
		return
	}

	t.Run("RegularTiles_NoTransformation", func(t *testing.T) {
		// Regular tiles should not be transformed
		pos := references.Position{X: 10, Y: 10}
		gameState.MapState.PlayerLocation.Position = pos

		// Get base tile
		baseTile := gameState.GetLayeredMapByCurrentLocation().GetTileTopMapOnlyTile(&pos)
		if baseTile == nil {
			t.Skip("No tile at test position")
			return
		}

		sprite := gameState.GetDisplaySpriteAtPosition(pos)

		// For regular tiles, should get base tile or animation variant
		t.Logf("Regular tile %s (%d) → display sprite %d", baseTile.Name, baseTile.Index, sprite)

		// Should not get avatar replacement sprites for non-replacement tiles
		nonReplacementSprites := []indexes.SpriteIndex{
			indexes.KlimbLadderUp, indexes.KlimbLadderDown,
			indexes.SleepingInBed, indexes.ManaclesPrisoner,
			indexes.SitChairRight, indexes.SitChairLeft,
		}

		for _, forbiddenSprite := range nonReplacementSprites {
			if sprite == forbiddenSprite {
				t.Errorf("Regular tile should not show avatar replacement sprite %d", forbiddenSprite)
			}
		}
	})
}

// TestBardSpriteReplacement tests bard-specific sprite logic in GetNPCSpriteReplacement.
// This validates the BANJO_PLAYER sprite switching per original world-state rules.
// See ref:TILE-0008.
func TestBardSpriteReplacement(t *testing.T) {
	gameState, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()

	if gameState == nil {
		t.Skip("Skipping test - real game data not available")
		return
	}

	pos := references.Position{X: 10, Y: 10}

	t.Run("BardOnSouthChair_ShowsBardPlaying", func(t *testing.T) {
		// When a bard (BardPlaying sprite) is on a south-facing chair (ChairBackBack),
		// they should show the BardPlaying sprite (actively playing music)
		presentation := gameState.GetNPCSpriteReplacement(indexes.ChairBackBack, indexes.BardPlaying1, pos)
		result := presentation.Sprite
		if result != indexes.BardPlaying1 {
			t.Errorf("Expected BardPlaying1 (%d) on south chair, got %d", indexes.BardPlaying1, result)
		}
		if !presentation.SuppressTerrainUnderlay {
			t.Error("performing bard should suppress the separate chair underlay")
		}
		t.Logf("✅ Bard on south-facing chair shows BardPlaying sprite")
	})

	t.Run("BardNotOnChair_ShowsMerchant", func(t *testing.T) {
		// When a bard is NOT on a south-facing chair, they should show Merchant sprite
		// (preserving animation frame offset)
		result := gameState.GetNPCSpriteReplacement(indexes.Grass, indexes.BardPlaying1, pos).Sprite
		if result != indexes.MerchantKeyIndex {
			t.Errorf("Expected Merchant_KeyIndex (%d) when bard not on chair, got %d", indexes.MerchantKeyIndex, result)
		}
		t.Logf("✅ Bard not on south chair converts to Merchant sprite")
	})

	t.Run("BardAnimationFramePreservation", func(t *testing.T) {
		// Test that animation frame offset is preserved when converting to Merchant
		testCases := []struct {
			bardSprite    indexes.SpriteIndex
			expectedMerch indexes.SpriteIndex
		}{
			{indexes.BardPlaying1, indexes.MerchantKeyIndex},
			{indexes.BardPlaying2, indexes.MerchantKeyIndex + 1},
			{indexes.BardPlaying3, indexes.MerchantKeyIndex + 2},
			{indexes.BardPlaying4, indexes.MerchantKeyIndex + 3},
		}

		for _, tc := range testCases {
			result := gameState.GetNPCSpriteReplacement(indexes.Grass, tc.bardSprite, pos).Sprite
			if result != tc.expectedMerch {
				t.Errorf("BardPlaying frame %d: expected Merchant %d, got %d",
					tc.bardSprite-indexes.BardPlayingKeyIndex, tc.expectedMerch, result)
			}
		}
		t.Logf("✅ Animation frame offset preserved when converting bard to merchant")
	})

	t.Run("BardOnOtherChairs_ShowsMerchant", func(t *testing.T) {
		// Bard on non-south chairs should still show Merchant (not BardPlaying)
		otherChairs := []indexes.SpriteIndex{
			indexes.ChairBackRight,
			indexes.ChairBackLeft,
			indexes.ChairBackForward,
		}

		for _, chairType := range otherChairs {
			result := gameState.GetNPCSpriteReplacement(chairType, indexes.BardPlaying1, pos).Sprite
			if result == indexes.BardPlaying1 {
				t.Errorf("Bard on chair %d should NOT show BardPlaying", chairType)
			}
		}
		t.Logf("✅ Bard on non-south chairs shows Merchant sprite")
	})

	t.Run("BardOnBed_ShowsSleepingInBed", func(t *testing.T) {
		// Per original world-state rules behavior: When bard is on a bed tile, they should show
		// SleepingInBed sprite, NOT standing merchant. The original converts bard to
		// merchant FIRST, then tile-context sprite projection applies bed transformation.
		// See ref:TILE-0008.
		result := gameState.GetNPCSpriteReplacement(indexes.LeftBed, indexes.BardPlaying1, pos).Sprite
		if result != indexes.SleepingInBed {
			t.Errorf("Expected SleepingInBed (%d) when bard on bed, got %d (merchant would be %d)",
				indexes.SleepingInBed, result, indexes.MerchantKeyIndex)
		}
		t.Logf("✅ Bard on bed shows SleepingInBed sprite")
	})

	t.Run("BardOnLadder_ShowsClimbing", func(t *testing.T) {
		// Bard on ladder should show climbing sprite, not standing merchant
		result := gameState.GetNPCSpriteReplacement(indexes.LadderUp, indexes.BardPlaying1, pos).Sprite
		if result != indexes.KlimbLadderUp {
			t.Errorf("Expected KlimbLadderUp (%d) when bard on ladder, got %d",
				indexes.KlimbLadderUp, result)
		}
		t.Logf("✅ Bard on ladder shows climbing sprite")
	})

	t.Run("BardInManacles_ShowsPrisoner", func(t *testing.T) {
		// Bard in manacles should show prisoner sprite
		result := gameState.GetNPCSpriteReplacement(indexes.Manacles, indexes.BardPlaying1, pos).Sprite
		if result != indexes.ManaclesPrisoner {
			t.Errorf("Expected ManaclesPrisoner (%d) when bard in manacles, got %d",
				indexes.ManaclesPrisoner, result)
		}
		t.Logf("✅ Bard in manacles shows prisoner sprite")
	})
}

// GetDisplaySpriteAtPosition returns the complete sprite that would be visible at a position.
// This includes avatar sprites, NPC sprites with replacements, and tiles from GetEffectiveTile.
// Used primarily by tests to validate the complete visual state at any position.
func (g *GameState) GetDisplaySpriteAtPosition(pos references.Position) indexes.SpriteIndex {
	// Get the base tile at this position
	currentMap := g.GetLayeredMapByCurrentLocation()
	baseTile := currentMap.GetTileTopMapOnlyTile(&pos)
	if baseTile == nil {
		// Handle out-of-bounds positions
		if g.IsOutOfBounds(pos) {
			return g.GetCurrentSmallLocationReference().GetOuterTile()
		}
		return indexes.NoSprites
	}

	// If avatar is at this position, get avatar sprite replacement
	if g.IsAvatarAtPosition(&pos) {
		return g.GetAvatarSpriteAtPosition(pos).Sprite
	}

	// Check for NPCs or objects at this position
	npcs := g.CurrentNPCAIController.GetNpcs()
	if g.CurrentNPCAIController == nil {
		panic("NPCAIController is nil")
	}
	if npcs != nil {
		if mapUnit := npcs.GetMapUnitAtPositionOrNil(pos); mapUnit != nil {
			// Quest item conditional visibility - hide quest item NPCs when corresponding item is in inventory
			if g.isQuestItemNPCHidden(*mapUnit) {
				// Quest item NPC should be hidden, fall through to show underlying tile
			} else {
				unitDetails := (*mapUnit).MapUnitDetails()
				// Only show NPCs on the current floor
				if unitDetails.Floor == g.MapState.PlayerLocation.Floor {
					// Get sprite from the map unit and apply sprite replacement
					currentMap := g.GetLayeredMapByCurrentLocation()
					mapUnitTile := currentMap.GetTileByLayer(map_state.MapUnitLayer, &pos)
					if mapUnitTile != nil {
						return g.GetNPCSpriteReplacement(baseTile.Index, mapUnitTile.Index, pos).Sprite
					}
				}
			}
		}
	}

	// Use unified tile override system for all tile rendering (equipment, special tiles like portcullis/mirrors/drawbridges)
	return g.GetEffectiveTile(&pos)
}
