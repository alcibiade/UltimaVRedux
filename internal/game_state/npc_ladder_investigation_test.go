package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// TestNPCAtLadderPosition investigates the guard at ladder position at 3:42pm
func TestNPCAtLadderPosition(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(1, 26). // At ladder
		WithSystemCallbacks().
		Build()
	ladderPos := references.Position{X: 1, Y: 26}

	t.Run("InvestigateGuardAt3_42PM", func(t *testing.T) {
		// Set time to 3:42 PM
		gs.DateTime.Hour = byte(15)
		gs.DateTime.Minute = byte(42)

		t.Logf("Time set to %d:%02d", gs.DateTime.Hour, gs.DateTime.Minute)
		t.Logf("Current floor: %d", gs.MapState.PlayerLocation.Floor)

		// Check what's at the ladder position
		currentMap := gs.GetLayeredMapByCurrentLocation()

		// Check base tile
		baseTile := currentMap.GetTileTopMapOnlyTile(&ladderPos)
		if baseTile != nil {
			t.Logf("Base tile at ladder: %s (index=%d)", baseTile.Name, baseTile.Index)
		}

		// Check MapUnitLayer
		mapUnitTile := currentMap.GetTileByLayer(map_state.MapUnitLayer, &ladderPos)
		if mapUnitTile != nil {
			t.Logf("MapUnitLayer at ladder: %s (index=%d)", mapUnitTile.Name, mapUnitTile.Index)
			if mapUnitTile.Index >= 332 && mapUnitTile.Index <= 335 {
				t.Logf("  WARNING: This is an avatar sprite!")
			}
		} else {
			t.Logf("MapUnitLayer at ladder: empty")
		}

		// Check if there's an NPC at this position
		if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
			npcs := gs.CurrentNPCAIController.GetNpcs()

			if mapUnit := npcs.GetMapUnitAtPositionOrNil(ladderPos); mapUnit != nil {
				unitDetails := (*mapUnit).MapUnitDetails()
				t.Logf("\n=== NPC FOUND AT LADDER ===")
				t.Logf("NPC Floor: %d", unitDetails.Floor)
				t.Logf("NPC Position: (%d, %d)", unitDetails.Position.X, unitDetails.Position.Y)

				switch npc := (*mapUnit).(type) {
				case *map_units.NPCFriendly:
					t.Logf("NPC Type: Friendly")
					// t.Logf("NPC Dialog Index: %d", npc.NPCReference.DialogIndex)
					t.Logf("NPC Type: %d", npc.NPCReference.NpcType)
					t.Logf("NPC Sprite Index: %d", npc.NPCReference.GetSpriteIndex())

					// Check schedule
					schedule := npc.NPCReference.Schedule
					behaviour := schedule.GetIndividualNPCBehaviourByUltimaDate(gs.DateTime)
					t.Logf("\nSchedule Info:")
					t.Logf("  AI Type: %d", behaviour.Ai)
					t.Logf("  Position: (%d, %d)", behaviour.Position.X, behaviour.Position.Y)
					t.Logf("  Floor: %d", behaviour.Floor)

					// Check if this is a guard
					if npc.NPCReference.NpcType == references.Guard {
						t.Logf("\n*** THIS IS A GUARD! ***")
					}
				}
			} else {
				t.Logf("No NPC found at ladder position via GetMapUnitAtPositionOrNil")
			}
		}

		// Check all NPCs on ALL floors and log their details
		t.Logf("\n=== All NPCs in Britain ===")
		if gs.CurrentNPCAIController != nil && gs.CurrentNPCAIController.GetNpcs() != nil {
			npcs := gs.CurrentNPCAIController.GetNpcs()
			guardCount := 0
			totalNPCs := 0

			for x := 0; x < 32; x++ {
				for y := 0; y < 32; y++ {
					pos := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
					if mapUnit := npcs.GetMapUnitAtPositionOrNil(pos); mapUnit != nil {
						unitDetails := (*mapUnit).MapUnitDetails()
						totalNPCs++

						switch npc := (*mapUnit).(type) {
						case *map_units.NPCFriendly:
							t.Logf("NPC #%d: Type=%d, at (%d,%d) floor %d, sprite=%d",
								totalNPCs, npc.NPCReference.NpcType, x, y, unitDetails.Floor, npc.NPCReference.GetSpriteIndex())

							// Check for guards by sprite index (Guard1-4: 368-371)
							spriteIndex := npc.NPCReference.GetSpriteIndex()
							if spriteIndex >= 368 && spriteIndex <= 371 {
								guardCount++
								t.Logf("  ^^ GUARD #%d", guardCount)

								// Check schedule info
								schedule := npc.NPCReference.Schedule
								behaviour := schedule.GetIndividualNPCBehaviourByUltimaDate(gs.DateTime)
								t.Logf("    Schedule: AI=%d, Position=(%d,%d), Floor=%d",
									behaviour.Ai, behaviour.Position.X, behaviour.Position.Y, behaviour.Floor)

								// Check if near ladder
								if (x == 1 && y == 26) || (x <= 2 && y >= 25 && y <= 27) {
									t.Logf("    *** GUARD NEAR LADDER! ***")
								}
							}
						default:
							t.Logf("NPC #%d: Unknown type at (%d,%d) floor %d", totalNPCs, x, y, unitDetails.Floor)
						}
					}
				}
			}
			t.Logf("Total NPCs found: %d", totalNPCs)
			t.Logf("Total guards found: %d", guardCount)
		}

		// Test what "Look" would see
		t.Logf("\n=== What 'Look' sees at ladder ===")
		displaySprite := gs.GetDisplaySpriteAtPosition(ladderPos)
		t.Logf("Display sprite: index=%d", displaySprite)

		// Check what Look would say about this sprite
		lookDescription := gs.GameReferences.LookReferences.GetTileLookDescription(displaySprite)
		t.Logf("Look description for sprite %d: '%s'", displaySprite, lookDescription)

		// Also check what guard sprites would say
		t.Logf("\n=== Guard sprite look descriptions ===")
		for guardSprite := 368; guardSprite <= 371; guardSprite++ {
			guardDescription := gs.GameReferences.LookReferences.GetTileLookDescription(indexes.SpriteIndex(guardSprite))
			t.Logf("Guard sprite %d: '%s'", guardSprite, guardDescription)
		}

		// Check if it matches various sprite ranges
		if displaySprite >= 332 && displaySprite <= 335 {
			t.Logf("Display sprite is an AVATAR sprite (332-335)")
		} else if displaySprite >= 368 && displaySprite <= 371 {
			t.Logf("Display sprite is a GUARD sprite (368-371)")
		} else if displaySprite == 279 {
			t.Logf("Display sprite is KlimbLadderUp (279)")
		}
	})
}

// TODO: TestGuardSchedulesInBritain - will implement after fixing field names
