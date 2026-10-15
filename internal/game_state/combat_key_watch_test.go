package game_state

import (
	"fmt"
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/require"
)

func TestCombatSkullKeyUsesActingCharacterInsteadOfExplorationPosition(t *testing.T) {
	for _, territory := range []references.CombatMapTerritory{
		references.BritanniaCombatTerritory, references.DungeonCombatTerritory,
	} {
		for _, actorIndex := range []int{0, 1} {
			for _, door := range []struct {
				locked   indexes.SpriteIndex
				unlocked indexes.SpriteIndex
			}{
				{indexes.MagicLockDoor, indexes.RegularDoor},
				{indexes.MagicLockDoorWithView, indexes.RegularDoorView},
			} {
				t.Run(fmt.Sprintf("%v/actor%d/%v", territory, actorIndex, door.locked), func(t *testing.T) {
					state, callbacks := NewIntegrationTestBuilder(t).WithLocation(references.Britain).
						WithSystemCallbacks().WithRandomSeed(1).Build()
					require.NoError(t, state.TransitionToCombatMap(0, territory, CombatEntryNorth))
					actor := CombatTurnIdentity{IsPartyMember: true, EntityIndex: actorIndex}
					state.CombatScene.CurrentTurnIdentity = actor
					state.CombatScene.currentTurnIdentitySet = true
					state.CombatScene.CurrentTurnIndex = actorIndex
					state.CombatScene.SetPartyMemberPosition(actorIndex, references.Position{X: 5, Y: 5})
					state.MapState.PlayerLocation.Position = references.Position{X: 1, Y: 1}
					target := references.Position{X: 5, Y: 4}
					decoy := state.MapState.PlayerLocation.Position.Up()
					state.setCombatTerrain(target, door.locked)
					state.setCombatTerrain(decoy, indexes.MagicLockDoor)
					state.PartyState.Inventory.Provisions.SkullKeys.Set(2)
					callbacks.Reset()
					beforeTime, beforeRNG := state.DateTime, state.GetRNG().Calls()
					beforeCommits := state.CastUseCommitSequence()
					commit, review := state.BeginSkullKeyUse()
					require.Equal(t, SkullKeyUseReady, review.Result)

					result, paid := state.CommitCombatSkullKeyUse(commit, references.Up)

					require.True(t, paid)
					require.Equal(t, SkullKeyUnlockSuccess, result.Result)
					require.Equal(t, target, result.Target)
					require.Equal(t, door.unlocked, state.CombatScene.CombatMapRef.GetTileNumber(target))
					require.Equal(t, door.unlocked, state.GetCurrentLayeredMap().GetTileTopMapOnlyTile(&target).Index)
					require.EqualValues(t, indexes.MagicLockDoor, state.CombatScene.CombatMapRef.GetTileNumber(decoy))
					require.EqualValues(t, 1, state.PartyState.Inventory.Provisions.SkullKeys.Get())
					require.Equal(t, beforeCommits+1, state.CastUseCommitSequence())
					require.Equal(t, beforeTime, state.DateTime)
					require.Equal(t, beforeRNG, state.GetRNG().Calls())
					require.Empty(t, callbacks.TimeAdvanced(), "combat completion belongs to the caller")
					_, paid = state.CommitCombatSkullKeyUse(commit, references.Up)
					require.False(t, paid, "one direction cannot pay twice")
				})
			}
		}
	}
}

func TestCombatSkullKeyUsesExactClonePosition(t *testing.T) {
	state, source, clone := newSyntheticExactIdentityCastUseState(t)
	state.CombatScene.CurrentTurnIdentity = clone
	state.CombatScene.setCombatantPosition(source, references.Position{X: 2, Y: 2})
	state.CombatScene.setCombatantPosition(clone, references.Position{X: 7, Y: 7})
	state.MapState.PlayerLocation.Position = references.Position{X: 1, Y: 1}
	target := references.Position{X: 7, Y: 6}
	decoy := references.Position{X: 2, Y: 1}
	state.setCombatTerrain(target, indexes.MagicLockDoorWithView)
	state.setCombatTerrain(decoy, indexes.MagicLockDoor)
	state.PartyState.Inventory.Provisions.SkullKeys.Set(1)
	commit, review := state.BeginSkullKeyUse()
	require.Equal(t, SkullKeyUseReady, review.Result)

	result, paid := state.CommitCombatSkullKeyUse(commit, references.Up)

	require.True(t, paid)
	require.Equal(t, SkullKeyUnlockSuccess, result.Result)
	require.Equal(t, target, result.Target)
	require.EqualValues(t, indexes.RegularDoorView, state.CombatScene.CombatMapRef.GetTileNumber(target))
	require.EqualValues(t, indexes.MagicLockDoor, state.CombatScene.CombatMapRef.GetTileNumber(decoy))
}

func TestCombatSkullKeyPaidNoEffectAndBoundaries(t *testing.T) {
	for _, test := range []struct {
		name      string
		origin    references.Position
		direction references.Direction
		tile      indexes.SpriteIndex
	}{
		{"ordinary lock", references.Position{X: 5, Y: 5}, references.Up, indexes.LockedDoor},
		{"ordinary view lock", references.Position{X: 5, Y: 5}, references.Up, indexes.LockedDoorView},
		{"floor", references.Position{X: 5, Y: 5}, references.Up, indexes.BrickFloor},
		{"north edge", references.Position{}, references.Up, indexes.MagicLockDoor},
		{"west edge", references.Position{}, references.Left, indexes.MagicLockDoor},
		{"south edge", references.Position{X: 10, Y: 10}, references.Down, indexes.MagicLockDoor},
		{"east edge", references.Position{X: 10, Y: 10}, references.Right, indexes.MagicLockDoor},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, source, _ := newSyntheticExactIdentityCastUseState(t)
			state.CombatScene.setCombatantPosition(source, test.origin)
			state.PartyState.Inventory.Provisions.SkullKeys.Set(1)
			target := test.direction.GetNewPositionInDirection(&test.origin)
			if target.X < 0 || target.Y < 0 || target.X >= 11 || target.Y >= 11 {
				// A magical door on the opposite edge detects accidental wrapping.
				target = &references.Position{X: (target.X + 11) % 11, Y: (target.Y + 11) % 11}
			}
			state.setCombatTerrain(*target, test.tile)
			beforeRNG, beforeTime := state.GetRNG().Calls(), state.DateTime
			beforeCommits := state.CastUseCommitSequence()
			commit, _ := state.BeginSkullKeyUse()

			result, paid := state.CommitCombatSkullKeyUse(commit, test.direction)

			require.True(t, paid)
			require.Equal(t, SkullKeyUnlockNoEffect, result.Result)
			require.Zero(t, state.PartyState.Inventory.Provisions.SkullKeys.Get())
			require.Equal(t, beforeCommits+1, state.CastUseCommitSequence())
			require.Equal(t, test.tile, state.CombatScene.CombatMapRef.GetTileNumber(*target))
			require.Equal(t, beforeRNG, state.GetRNG().Calls())
			require.Equal(t, beforeTime, state.DateTime)
		})
	}
}

func TestCombatKeyAndWatchRejectRetainedOwnerChanges(t *testing.T) {
	for _, item := range []references.Item{references.SkullKeys, references.PocketWatch} {
		for _, change := range []string{
			"quantity", "removed", "actor", "clone", "same slot", "provider", "character",
			"later dispatch", "encounter", "moved", "map", "terrain map", "layered map",
		} {
			t.Run(fmt.Sprintf("%v/%s", item, change), func(t *testing.T) {
				state, source, clone := newSyntheticExactIdentityCastUseState(t)
				state.PartyState.Inventory.Provisions.SkullKeys.Set(2)
				state.PartyState.Inventory.QuestItems.Set(references.PocketWatch, 1)
				state.CombatScene.setCombatantPosition(source, references.Position{X: 5, Y: 5})
				target := references.Position{X: 5, Y: 4}
				state.setCombatTerrain(target, indexes.MagicLockDoor)
				key, _ := state.BeginSkullKeyUse()
				owner := state.CaptureCombatUseOwner()
				watch := state.BeginCombatPocketWatchUse(owner)
				require.NotNil(t, key)
				require.NotNil(t, watch)
				changeCombatKeyWatchOwner(t, state, source, clone, item, change)
				beforeInventory := state.PartyState.Inventory.LockerSnapshot()
				beforeRNG, beforeTime := state.GetRNG().Calls(), state.DateTime
				beforeCommits := state.CastUseCommitSequence()
				if item == references.SkullKeys {
					_, paid := state.CommitCombatSkullKeyUse(key, references.Up)
					require.False(t, paid)
				} else {
					_, paid := state.CommitCombatPocketWatchUse(watch)
					require.False(t, paid)
				}
				require.Equal(t, beforeInventory, state.PartyState.Inventory.LockerSnapshot())
				require.Equal(t, beforeRNG, state.GetRNG().Calls())
				require.Equal(t, beforeTime, state.DateTime)
				require.Equal(t, beforeCommits, state.CastUseCommitSequence())
				require.EqualValues(t, indexes.MagicLockDoor, state.CombatScene.CombatMapRef.GetTileNumber(target))
			})
		}
	}
}

func changeCombatKeyWatchOwner(
	t *testing.T, state *GameState, source, clone CombatTurnIdentity, item references.Item, change string,
) {
	t.Helper()
	combat := state.CombatScene
	entry := combat.TurnQueue.GetEntryByIdentity(source)
	switch change {
	case "quantity", "removed":
		quantity := uint16(3)
		if change == "removed" {
			quantity = 0
		}
		if item == references.SkullKeys {
			state.PartyState.Inventory.Provisions.SkullKeys.Set(quantity)
		} else {
			state.PartyState.Inventory.QuestItems.Set(references.PocketWatch, quantity)
		}
	case "actor":
		combat.CurrentTurnIdentity = CombatTurnIdentity{IsPartyMember: true, EntityIndex: 1}
	case "clone":
		combat.CurrentTurnIdentity = clone
	case "same slot":
		for index, candidate := range combat.TurnQueue.entries {
			if candidate == entry {
				replacement := *entry
				combat.TurnQueue.entries[index] = &replacement
			}
		}
	case "provider":
		replacement := *entry.Combatant.(*CombatPCProvider)
		entry.Combatant = &replacement
	case "character":
		provider := entry.Combatant.(*CombatPCProvider)
		replacement := *provider.Character()
		provider.character = &replacement
	case "later dispatch":
		combat.nextPossessionTurn++
	case "encounter":
		replacement := *combat
		state.CombatScene = &replacement
	case "moved":
		combat.setCombatantPosition(source, references.Position{X: 6, Y: 5})
	case "map":
		state.MapState.PlayerLocation.Location = references.Britain
	case "terrain map":
		replacement := *combat.CombatMapRef
		combat.CombatMapRef = &replacement
	case "layered map":
		state.MapState.LayeredMaps.ResetAndCreateCombatMap(combat.CombatMapRef, state.GameReferences.TileReferences, 19, 13)
	default:
		t.Fatalf("unknown owner change %q", change)
	}
}

func TestCombatPocketWatchReadsExecutionTimeAndRetainsInventory(t *testing.T) {
	state, _, _ := newSyntheticExactIdentityCastUseState(t)
	state.PartyState.Inventory.QuestItems.Set(references.PocketWatch, 1)
	commit := state.BeginCombatPocketWatchUse(state.CaptureCombatUseOwner())
	require.NotNil(t, commit)
	state.DateTime.Hour, state.DateTime.Minute = 23, 47
	beforeTime, beforeRNG := state.DateTime, state.GetRNG().Calls()
	beforeCommits := state.CastUseCommitSequence()
	timeString, paid := state.CommitCombatPocketWatchUse(commit)
	require.True(t, paid)
	require.Equal(t, beforeTime.GetTimeAsString(), timeString)
	require.Equal(t, beforeCommits+1, state.CastUseCommitSequence())
	require.EqualValues(t, 1, state.PartyState.Inventory.QuestItems.Get(references.PocketWatch))
	require.Equal(t, beforeTime, state.DateTime)
	require.Equal(t, beforeRNG, state.GetRNG().Calls())
	_, paid = state.CommitCombatPocketWatchUse(commit)
	require.False(t, paid)
	require.Equal(t, beforeCommits+1, state.CastUseCommitSequence())
}

func TestCombatSkullKeyRejectsCancelledDirectionAndStaleUnlock(t *testing.T) {
	state, actor, _ := newSyntheticExactIdentityCastUseState(t)
	state.CombatScene.setCombatantPosition(actor, references.Position{X: 5, Y: 5})
	state.PartyState.Inventory.Provisions.SkullKeys.Set(1)
	target := references.Position{X: 5, Y: 4}
	state.setCombatTerrain(target, indexes.MagicLockDoor)
	commit, _ := state.BeginSkullKeyUse()
	beforeCommits := state.CastUseCommitSequence()
	_, paid := state.CommitCombatSkullKeyUse(commit, references.NoneDirection)
	require.False(t, paid)
	require.EqualValues(t, 1, state.PartyState.Inventory.Provisions.SkullKeys.Get())
	require.Equal(t, beforeCommits, state.CastUseCommitSequence())
	require.Equal(t, SkullKeyUseConsumed, state.CommitSkullKeyUse(commit))
	review := state.ReviewSkullKeyUnlock(commit, references.Up)
	require.Equal(t, SkullKeyUnlockReady, review.Result)
	state.CombatScene.setCombatantPosition(actor, references.Position{X: 6, Y: 5})
	require.Equal(t, SkullKeyUnlockStale, state.CommitSkullKeyUnlock(review))
	require.EqualValues(t, indexes.MagicLockDoor, state.CombatScene.CombatMapRef.GetTileNumber(target))
	require.EqualValues(t, indexes.MagicLockDoor,
		state.GetCurrentLayeredMap().SpriteIndexByLayer(map_state.MapLayer, target))
}
