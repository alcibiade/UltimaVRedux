package game_state

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/config"
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/files"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

type indexedDungeonTrigger struct {
	index   int
	trigger references.DungeonCombatMapTrigger
}

func TestCheckTrapSynchronizesRepresentationsAndRefreshesOnceForAllMatches(t *testing.T) {
	triggerPosition := references.Position{X: 0, Y: 0}
	destination := references.Position{X: 8, Y: 8}
	state, callbacks := newSyntheticDungeonTriggerCombatState(t,
		indexedDungeonTrigger{index: 0, trigger: references.DungeonCombatMapTrigger{
			ReplacementTile: indexes.BrickFloor,
			Trigger:         triggerPosition,
			Destinations:    [2]references.Position{destination, {X: 11, Y: 0}},
		}},
		indexedDungeonTrigger{index: 1, trigger: references.DungeonCombatMapTrigger{
			ReplacementTile: indexes.StoneBrickWall,
			Trigger:         triggerPosition,
			Destinations:    [2]references.Position{destination, {X: 0, Y: 11}},
		}},
		indexedDungeonTrigger{index: 2, trigger: references.DungeonCombatMapTrigger{
			ReplacementTile: indexes.BrickWallArchway,
			Trigger:         triggerPosition,
			Destinations:    [2]references.Position{{X: 11, Y: 11}, {X: 0xff, Y: 0xff}},
		}},
	)
	randomCallsBefore := state.GetRNG().Calls()
	hpBefore := state.PartyState.GetCharacter(0).GetCurrentHp()

	assert.True(t, state.CheckTrap(&triggerPosition))
	assert.Equal(t, 1, callbacks.DisplayUpdates, "all matching slots cause one display refresh")
	assert.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall),
		state.CombatScene.CombatMapRef.GetTileNumber(destination))
	assert.Equal(t, indexes.SpriteIndex(indexes.StoneBrickWall),
		state.GetCurrentLayeredMap().SpriteIndexByLayer(map_state.MapLayer, destination))
	assert.False(t, state.isCombatPositionValid(&destination, -1),
		"movement immediately observes the synchronized replacement terrain")
	launch := traceCombatProjectileLaunch(
		state.combatMissileTerrainAt,
		references.Position{X: 6, Y: 8},
		references.Position{X: 10, Y: 8},
		references.MissileArrow,
	)
	assert.True(t, launch.Blocked, "later projectile targeting observes the reference replacement")
	assert.Equal(t, destination, launch.Impact)
	assert.False(t, state.CheckTrap(&triggerPosition))
	assert.Equal(t, 1, callbacks.DisplayUpdates, "consumed slots do not refresh again")
	assert.Equal(t, randomCallsBefore, state.GetRNG().Calls())
	assert.Equal(t, hpBefore, state.PartyState.GetCharacter(0).GetCurrentHp())
	assert.Empty(t, callbacks.Messages)
	assert.Empty(t, callbacks.TimeAdvanced())
	assert.Empty(t, callbacks.SoundEffectsPlayed)

	slots, present := state.CombatScene.CombatMapRef.DungeonTriggerSlots()
	require.True(t, present)
	for slotIndex := range 3 {
		assert.Equal(t, references.Position{
			X: references.DungeonTriggerConsumedPosition,
			Y: references.DungeonTriggerConsumedPosition,
		}, slots[slotIndex].Trigger)
	}
}

func TestCommandedCombatMovementTriggerBoundaries(t *testing.T) {
	t.Run("party destination", func(t *testing.T) {
		trigger := references.Position{X: 3, Y: 2}
		state, callbacks := newSingleTriggerCombatState(t, trigger)
		state.CombatScene.SetPartyMemberPosition(0, references.Position{X: 2, Y: 2})

		assert.Equal(t, CombatMoveMoved, state.MoveCombatCharacter(references.Right))
		assertDungeonTriggerConsumed(t, state, 0)
		assert.Equal(t, 1, callbacks.DisplayUpdates)
	})

	t.Run("party blocked", func(t *testing.T) {
		trigger := references.Position{X: 3, Y: 2}
		state, callbacks := newSingleTriggerCombatState(t, trigger)
		state.CombatScene.SetPartyMemberPosition(0, references.Position{X: 2, Y: 2})
		state.setCombatTerrain(trigger, indexes.StoneBrickWall)

		assert.Equal(t, CombatMoveBlocked, state.MoveCombatCharacter(references.Right))
		assertDungeonTriggerUnconsumed(t, state, 0, trigger)
		assert.Zero(t, callbacks.DisplayUpdates)
	})

	t.Run("party flee", func(t *testing.T) {
		trigger := references.Position{X: 0, Y: 2}
		state, callbacks := newSingleTriggerCombatState(t, trigger)
		state.CombatScene.SetPartyMemberPosition(0, trigger)

		assert.Equal(t, CombatMoveFled, state.MoveCombatCharacter(references.Left))
		assertDungeonTriggerUnconsumed(t, state, 0, trigger)
		assert.Zero(t, callbacks.DisplayUpdates)
	})

	t.Run("party-loyal monster destination", func(t *testing.T) {
		trigger := references.Position{X: 3, Y: 2}
		state, callbacks := newSingleTriggerCombatState(t, trigger)
		state.CombatScene.SetPartyMemberPosition(0, references.Position{X: 9, Y: 9})
		monster, monsterIndex := addSyntheticTriggerMonster(
			state,
			references.Position{X: 2, Y: 2},
			CombatMonsterLoyaltyParty,
			references.MissileArrow,
		)
		authorizeControlledMonster(state, monsterIndex)
		callbacks.Reset()

		assert.Equal(t, CombatMoveMoved, state.MoveCombatCharacter(references.Right))
		assert.Equal(t, trigger, monster.Position())
		assertDungeonTriggerConsumed(t, state, 0)
		assert.Equal(t, 1, callbacks.DisplayUpdates)
	})

	t.Run("ordinary hostile AI movement", func(t *testing.T) {
		trigger := references.Position{X: 3, Y: 2}
		state, callbacks := newSingleTriggerCombatState(t, trigger)
		state.CombatScene.SetPartyMemberPosition(0, references.Position{X: 9, Y: 9})
		monster, monsterIndex := addSyntheticTriggerMonster(
			state,
			references.Position{X: 2, Y: 2},
			CombatMonsterLoyaltyMonster,
			references.MissileArrow,
		)
		callbacks.Reset()

		assert.True(t, state.tryMonsterMoveAxis(monster, monsterIndex, monster.Position(), 1, 0))
		assert.Equal(t, trigger, monster.Position())
		assertDungeonTriggerUnconsumed(t, state, 0, trigger)
		assert.Zero(t, callbacks.DisplayUpdates)
	})
}

func TestCombatKlimbSmallRockRelocatesThenTriggers(t *testing.T) {
	trigger := references.Position{X: 3, Y: 2}
	state, callbacks := newSingleTriggerCombatState(t, trigger)
	origin := references.Position{X: 2, Y: 2}
	state.CombatScene.SetPartyMemberPosition(0, origin)
	state.setCombatTerrain(trigger, indexes.SmallRockWall)
	callbacks.Reset()

	require.True(t, state.ActionKlimbCombatMap(references.Right))
	assert.Equal(t, trigger, state.CombatScene.GetPartyMemberPosition(0))
	assertDungeonTriggerConsumed(t, state, 0)
	assert.Equal(t, 1, callbacks.DisplayUpdates)
	require.Len(t, callbacks.CombatEvents, 1)
	assert.Equal(t, CombatEventMove, callbacks.CombatEvents[0].Kind)
	assert.Equal(t, origin, callbacks.CombatEvents[0].From)
	assert.Equal(t, trigger, callbacks.CombatEvents[0].To)
}

func TestCombatKlimbLadderExitDoesNotTrigger(t *testing.T) {
	position := references.Position{X: 2, Y: 2}
	state, callbacks := newSingleTriggerCombatState(t, position)
	state.CombatScene.SetPartyMemberPosition(0, position)
	state.setCombatTerrain(position, indexes.LadderUp)
	callbacks.Reset()

	require.True(t, state.ActionKlimbCombatMap(references.Right))
	assertDungeonTriggerUnconsumed(t, state, 0, position)
	assert.Zero(t, callbacks.DisplayUpdates)
}

func TestCombatPushResolvesTriggerBeforeOrdinaryPush(t *testing.T) {
	trigger := references.Position{X: 3, Y: 2}
	state, callbacks := newSingleTriggerCombatState(t, trigger)
	state.CombatScene.SetPartyMemberPosition(0, references.Position{X: 2, Y: 2})
	callbacks.Reset()

	assert.True(t, state.ActionPushCombatMap(references.Right))
	assert.Equal(t, references.Position{X: 2, Y: 2}, state.CombatScene.GetPartyMemberPosition(0))
	assertDungeonTriggerConsumed(t, state, 0)
	assert.Equal(t, 1, callbacks.DisplayUpdates)
	assert.Empty(t, callbacks.Messages, "ordinary Push feedback must not run after a trigger handles the target")
	assert.Empty(t, callbacks.TimeAdvanced())
}

func TestPhysicalMissileEndpointsTriggerForEveryCombatAllegiance(t *testing.T) {
	impact := references.Position{X: 6, Y: 5}

	t.Run("party empty endpoint", func(t *testing.T) {
		state, callbacks := newSingleTriggerCombatState(t, impact)
		installPendingTriggerAttack(state, CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}, CombatAttackSource{
			Slot: references.SlotWeapon, Equipment: references.Bow, MaxRange: 6, Missile: references.MissileArrow,
		}, CombatProjectileLaunch{
			From: references.Position{X: 2, Y: 5}, Aim: impact, Impact: impact,
			Missile: references.MissileArrow, Animate: true,
		})

		state.ResolvePendingCombatAttack()
		assertDungeonTriggerConsumed(t, state, 0)
		assert.Equal(t, 1, callbacks.DisplayUpdates)
	})

	t.Run("party-loyal controlled monster empty endpoint", func(t *testing.T) {
		state, callbacks := newSingleTriggerCombatState(t, impact)
		monster, monsterIndex := addSyntheticTriggerMonster(
			state, references.Position{X: 2, Y: 5}, CombatMonsterLoyaltyParty, references.MissileArrow,
		)
		installPendingTriggerAttack(state, CombatTurnIdentity{EntityIndex: monsterIndex}, CombatAttackSource{
			Slot: references.SlotNone, Equipment: references.NoEquipment, MaxRange: 6, Missile: references.MissileArrow,
		}, CombatProjectileLaunch{
			From: monster.Position(), Aim: impact, Impact: impact,
			Missile: references.MissileArrow, Animate: true,
		})
		callbacks.Reset()

		state.ResolvePendingCombatAttack()
		assertDungeonTriggerConsumed(t, state, 0)
		assert.Equal(t, 1, callbacks.DisplayUpdates)
	})

	t.Run("hostile monster empty endpoint", func(t *testing.T) {
		state, callbacks := newSingleTriggerCombatState(t, impact)
		monster, monsterIndex := addSyntheticTriggerMonster(
			state, references.Position{X: 2, Y: 5}, CombatMonsterLoyaltyMonster, references.MissileRock,
		)
		state.CombatScene.pendingMonsterMissile = &pendingMonsterMissile{
			attackerIndex: monsterIndex,
			launch: CombatProjectileLaunch{
				From: monster.Position(), Aim: impact, Impact: impact,
				Missile: references.MissileRock, Animate: true,
			},
			diagnostics: CombatDiagnostics{Missile: references.MissileRock},
		}
		callbacks.Reset()

		require.True(t, state.ResolvePendingMonsterMissile())
		assertDungeonTriggerConsumed(t, state, 0)
		assert.Equal(t, 1, callbacks.DisplayUpdates)
	})
}

func TestPhysicalMissileUsesRedirectedFinalEndpoint(t *testing.T) {
	primaryPosition := references.Position{X: 5, Y: 5}
	redirectedImpact := references.Position{X: 6, Y: 5}
	state, callbacks := newSingleTriggerCombatState(t, redirectedImpact)
	target, targetIndex := addSyntheticTriggerMonster(
		state, primaryPosition, CombatMonsterLoyaltyMonster, references.MissileArrow,
	)
	installPendingTriggerAttack(state, CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}, CombatAttackSource{
		Slot: references.SlotWeapon, Equipment: references.Bow, MaxRange: 6, Missile: references.MissileArrow,
	}, CombatProjectileLaunch{
		From: references.Position{X: 2, Y: 5}, Aim: redirectedImpact, Impact: redirectedImpact,
		Missile: references.MissileArrow, Animate: true,
	})
	state.CombatScene.pendingAttack.primarySet = true
	state.CombatScene.pendingAttack.primaryTarget = CombatTurnIdentity{EntityIndex: targetIndex}
	state.CombatScene.pendingAttack.diagnostics.Hit = CombatHitResult{Outcome: CombatHitMiss}
	hpBefore := target.CurrentHP()
	callbacks.Reset()

	state.ResolvePendingCombatAttack()
	assertDungeonTriggerConsumed(t, state, 0)
	assert.Equal(t, hpBefore, target.CurrentHP(), "redirected misses still do no target damage")
	assert.Equal(t, 1, callbacks.DisplayUpdates)
}

func TestHostilePhysicalMissileUsesRedirectedFinalEndpoint(t *testing.T) {
	primaryPosition := references.Position{X: 5, Y: 5}
	redirectedImpact := references.Position{X: 6, Y: 5}
	state, callbacks := newSingleTriggerCombatState(t, redirectedImpact)
	state.CombatScene.SetPartyMemberPosition(0, primaryPosition)
	monster, monsterIndex := addSyntheticTriggerMonster(
		state, references.Position{X: 2, Y: 5}, CombatMonsterLoyaltyMonster, references.MissileRock,
	)
	nearest := findNearestPC(monster.Position(), state.CombatScene)
	callbacks.Reset()

	launch, attacked := state.prepareMonsterAttackWithRNG(
		monster, monsterIndex, nearest, newSequenceRNG(0, 0, 2, 1),
	)

	require.True(t, attacked)
	require.NotNil(t, launch)
	assert.Equal(t, redirectedImpact, launch.Impact)
	require.True(t, state.ResolvePendingMonsterMissile())
	assertDungeonTriggerConsumed(t, state, 0)
	assert.Equal(t, 1, callbacks.DisplayUpdates)
}

func TestPhysicalMissileTriggerExclusions(t *testing.T) {
	impact := references.Position{X: 6, Y: 5}
	t.Run("unblocked traversal square", func(t *testing.T) {
		traversal := references.Position{X: 4, Y: 5}
		state, callbacks := newSingleTriggerCombatState(t, traversal)
		installPendingTriggerAttack(state, CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}, CombatAttackSource{
			Slot: references.SlotWeapon, Equipment: references.Bow, MaxRange: 6, Missile: references.MissileArrow,
		}, CombatProjectileLaunch{
			From: references.Position{X: 2, Y: 5}, Aim: impact, Impact: impact,
			Missile: references.MissileArrow, Animate: true,
		})

		state.ResolvePendingCombatAttack()
		assertDungeonTriggerUnconsumed(t, state, 0, traversal)
		assert.Zero(t, callbacks.DisplayUpdates)
	})

	t.Run("blocked impact", func(t *testing.T) {
		state, callbacks := newSingleTriggerCombatState(t, impact)
		installPendingTriggerAttack(state, CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}, CombatAttackSource{
			Slot: references.SlotWeapon, Equipment: references.Bow, MaxRange: 6, Missile: references.MissileArrow,
		}, CombatProjectileLaunch{
			From: references.Position{X: 2, Y: 5}, Aim: references.Position{X: 8, Y: 5}, Impact: impact,
			Missile: references.MissileArrow, Blocked: true, Animate: true,
		})

		state.ResolvePendingCombatAttack()
		assertDungeonTriggerUnconsumed(t, state, 0, impact)
		assert.Zero(t, callbacks.DisplayUpdates)
	})

	for _, equipment := range []references.Equipment{references.MorningStar, references.Halberd} {
		t.Run(equipment.String(), func(t *testing.T) {
			state, callbacks := newSingleTriggerCombatState(t, impact)
			installPendingTriggerAttack(state, CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}, CombatAttackSource{
				Slot: references.SlotWeapon, Equipment: equipment, MaxRange: 2, Missile: equipment.GetMissileType(),
			}, CombatProjectileLaunch{
				From: references.Position{X: 4, Y: 5}, Aim: impact, Impact: impact,
				Missile: equipment.GetMissileType(), Animate: false,
			})

			state.ResolvePendingCombatAttack()
			assertDungeonTriggerUnconsumed(t, state, 0, impact)
			assert.Zero(t, callbacks.DisplayUpdates)
		})
	}

	t.Run("Magic Axe outbound", func(t *testing.T) {
		state, callbacks := newSingleTriggerCombatState(t, impact)
		installPendingTriggerAttack(state, CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}, CombatAttackSource{
			Slot: references.SlotWeapon, Equipment: references.MagicAxe, MaxRange: 4, Missile: references.MissileAxe,
		}, CombatProjectileLaunch{
			From: references.Position{X: 2, Y: 5}, Aim: impact, Impact: impact,
			Missile: references.MissileAxe, Animate: true,
		})

		state.ResolvePendingCombatAttack()
		assertDungeonTriggerConsumed(t, state, 0)
		assert.Equal(t, 1, callbacks.DisplayUpdates)
	})

	t.Run("Magic Axe return", func(t *testing.T) {
		origin := references.Position{X: 2, Y: 5}
		state, callbacks := newSingleTriggerCombatState(t, origin)
		installPendingTriggerAttack(state, CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}, CombatAttackSource{
			Slot: references.SlotWeapon, Equipment: references.MagicAxe, MaxRange: 4, Missile: references.MissileAxe,
		}, CombatProjectileLaunch{
			From: origin, Aim: impact, Impact: impact,
			Missile: references.MissileAxe, Animate: true,
		})

		state.ResolvePendingCombatAttack()
		assertDungeonTriggerUnconsumed(t, state, 0, origin)
		assert.Zero(t, callbacks.DisplayUpdates, "presentation-only return flight is not a trigger endpoint")
	})
}

func TestSevenOriginalLaunchSpellsResolveUnblockedFinalEndpoints(t *testing.T) {
	impact := references.Position{X: 6, Y: 5}
	spells := []references.Spell{
		references.GravPor,
		references.VasFlam,
		references.InFlamGrav,
		references.InNoxGrav,
		references.InZuGrav,
		references.InSanctGrav,
		references.XenCorp,
	}
	for _, spell := range spells {
		t.Run(references.GetSpellInfo(spell).Name, func(t *testing.T) {
			state, callbacks := newSingleTriggerCombatState(t, impact)
			pending := &pendingCombatSpell{
				spell: spell, casterIdx: 0,
				caster: CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0},
				launch: CombatProjectileLaunch{
					From: references.Position{X: 2, Y: 5}, Aim: impact, Impact: impact,
					Missile: references.MissileRed, Animate: true,
				},
			}
			if spell == references.GravPor || spell == references.VasFlam || spell == references.XenCorp {
				_, pending.monsterIdx = addSyntheticTriggerMonster(
					state, impact, CombatMonsterLoyaltyMonster, references.MissileNone,
				)
				pending.target = CombatTurnIdentity{EntityIndex: pending.monsterIdx}
			}
			state.CombatScene.pendingSpell = pending
			callbacks.Reset()

			require.True(t, state.ResolvePendingCombatSpell())
			assertDungeonTriggerConsumed(t, state, 0)
			assert.Equal(t, 1, callbacks.DisplayUpdates)
		})
	}
}

func TestCombatSpellTriggerEligibilityAndBlockedExclusion(t *testing.T) {
	eligible := map[references.Spell]bool{
		references.GravPor:     true,
		references.VasFlam:     true,
		references.InFlamGrav:  true,
		references.InNoxGrav:   true,
		references.InZuGrav:    true,
		references.InSanctGrav: true,
		references.XenCorp:     true,
		references.AnXenEx:     false,
		references.RelXenBet:   false,
		references.InQuasXen:   false,
		references.InLor:       false,
	}
	for spell, want := range eligible {
		assert.Equal(t, want, combatSpellTriggersDungeonMap(spell), references.GetSpellInfo(spell).Name)
	}

	impact := references.Position{X: 6, Y: 5}
	state, callbacks := newSingleTriggerCombatState(t, impact)
	state.CombatScene.pendingSpell = &pendingCombatSpell{
		spell: references.VasFlam, casterIdx: 0,
		caster: CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0},
		launch: CombatProjectileLaunch{
			From: references.Position{X: 2, Y: 5}, Aim: references.Position{X: 8, Y: 5}, Impact: impact,
			Missile: references.MissileRed, Blocked: true, Animate: true,
		},
	}

	require.True(t, state.ResolvePendingCombatSpell())
	assertDungeonTriggerUnconsumed(t, state, 0, impact)
	assert.Zero(t, callbacks.DisplayUpdates)
}

func TestFreshLiveAndClearedFixedRoomLoadsRestoreSourceTriggers(t *testing.T) {
	data := syntheticDungeonTriggerData(indexedDungeonTrigger{
		index: 0,
		trigger: references.DungeonCombatMapTrigger{
			Trigger:      references.Position{X: 0, Y: 0},
			Destinations: [2]references.Position{{X: 0xff, Y: 0xff}, {X: 0xff, Y: 0xff}},
		},
	})
	dataPath := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dataPath, files.DUNGEON_CBT), data, 0o600))
	var cells references.DungeonCells
	position := references.Position{X: 1, Y: 1}
	cells.SetCell(0, position, references.DungeonCell(0xf0))
	state := newDungeonStateTestGame(t, cells)
	state.GameConfig = syntheticCombatConfig(dataPath)
	tiles := references.NewTileReferences()
	enemyReferences := syntheticTriggerEnemyReferences()
	state.GameReferences = &references.GameReferences{ //nolint:exhaustruct // Fixed-room fixture needs only combat references.
		TileReferences:  tiles,
		EnemyReferences: &enemyReferences,
	}
	callbacks := NewMockSystemCallbacks(t)
	state.SystemCallbacks = callbacks.ToSystemCallbacks()
	state.EventQueueManager = event_queue.NewEventQueueManager(func() int64 { return 0 }, nil)

	require.NoError(t, state.ResolvePendingDungeonRoomEncounter())
	trigger := references.Position{X: 0, Y: 0}
	assertDungeonTriggerUnconsumed(t, state, 0, trigger)
	require.True(t, state.CheckTrap(&trigger))
	assertDungeonTriggerConsumed(t, state, 0)
	state.TransitionFromCombatMap()
	state.synchronizeDungeonRoomEncounterForStablePosition()

	require.NoError(t, state.ResolvePendingDungeonRoomEncounter())
	assertDungeonTriggerUnconsumed(t, state, 0, trigger)
	state.TransitionFromCombatMap()

	require.True(t, state.activeDungeon.cells.SetCell(0, position, references.DungeonCell(0xa0)))
	state.synchronizeDungeonRoomEncounterForStablePosition()
	require.NoError(t, state.ResolvePendingDungeonRoomEncounter())
	assert.False(t, state.CombatScene.DungeonRoomOrigin.Live)
	assertDungeonTriggerUnconsumed(t, state, 0, trigger)
}

func newSingleTriggerCombatState(
	t *testing.T,
	triggerPosition references.Position,
) (*GameState, *MockSystemCallbacks) {
	t.Helper()

	return newSyntheticDungeonTriggerCombatState(t, indexedDungeonTrigger{
		index: 0,
		trigger: references.DungeonCombatMapTrigger{
			ReplacementTile: indexes.BrickFloor,
			Trigger:         triggerPosition,
			Destinations: [2]references.Position{
				{X: 0xff, Y: 0xff},
				{X: 0xff, Y: 0xff},
			},
		},
	})
}

func newSyntheticDungeonTriggerCombatState(
	t *testing.T,
	triggers ...indexedDungeonTrigger,
) (*GameState, *MockSystemCallbacks) {
	t.Helper()

	dataPath := t.TempDir()
	require.NoError(t, os.WriteFile(
		filepath.Join(dataPath, files.DUNGEON_CBT),
		syntheticDungeonTriggerData(triggers...),
		0o600,
	))
	combatMap, err := references.LoadCombatMap(
		0,
		references.DungeonCombatTerritory,
		syntheticCombatConfig(dataPath),
	)
	require.NoError(t, err)

	var roster [party_state.NRosterSlots]*party_state.PlayerCharacter
	for index := range roster {
		roster[index] = &party_state.PlayerCharacter{PartyMap: party_state.PartyMapUnrecruited}
	}
	avatar := roster[0]
	avatar.PartyMap = party_state.PartyMapActive
	avatar.Name[0] = 'A'
	avatar.Class = 'A'
	avatar.Strength = 30
	avatar.Dexterity = 30
	avatar.Intelligence = 30
	avatar.SetMaxHp(100)
	avatar.SetHp(100)
	avatar.SetStatus(party_state.Good)
	party := party_state.NewPartyStateFromRoster(roster)
	combat := NewCombatState(combatMap, CombatEntryNorth, party)
	tiles := references.NewTileReferences()
	enemyReferences := syntheticTriggerEnemyReferences()
	callbacks := NewMockSystemCallbacks(t)
	state := &GameState{ //nolint:exhaustruct // Focused synthetic combat fixture.
		PartyState: *party,
		GameReferences: &references.GameReferences{ //nolint:exhaustruct // Combat fixture needs only these references.
			TileReferences:  tiles,
			EnemyReferences: &enemyReferences,
		},
		SystemCallbacks: callbacks.ToSystemCallbacks(),
	}
	state.MapState.XTilesVisibleOnGameScreen = 19
	state.MapState.YTilesVisibleOnGameScreen = 13
	state.MapState.LayeredMaps = *map_state.NewLayeredMaps(
		tiles,
		&references.LargeMapReference{},
		&references.LargeMapReference{},
		19,
		13,
	)
	state.SetRandomSeed(1)
	state.configureCombatTurnQueue(combat)
	state.initializeCombatPartyRings(combat)
	state.commitCombatTransition(combat, references.DungeonCombatTerritory)
	state.CombatScene.SetPartyMemberPosition(0, references.Position{X: 2, Y: 2})
	callbacks.Reset()

	return state, callbacks
}

func syntheticDungeonTriggerData(triggers ...indexedDungeonTrigger) []byte {
	data := make([]byte, references.CombatMapDataSize)
	for row := range references.YCombatMapTiles {
		for column := range references.CombatMapTileDataBytes {
			data[int(row)*references.CombatMapBytesPerRow+column] = byte(indexes.BrickFloor)
		}
	}
	for slotIndex := range references.DungeonCombatTriggerSlotCount {
		writeSyntheticDungeonTrigger(data, slotIndex, references.DungeonCombatMapTrigger{
			Trigger: references.Position{
				X: references.DungeonTriggerConsumedPosition,
				Y: references.DungeonTriggerConsumedPosition,
			},
			Destinations: [2]references.Position{
				{X: references.DungeonTriggerConsumedPosition, Y: references.DungeonTriggerConsumedPosition},
				{X: references.DungeonTriggerConsumedPosition, Y: references.DungeonTriggerConsumedPosition},
			},
		})
	}
	for _, trigger := range triggers {
		writeSyntheticDungeonTrigger(data, trigger.index, trigger.trigger)
	}

	return data
}

func writeSyntheticDungeonTrigger(
	data []byte,
	slotIndex int,
	trigger references.DungeonCombatMapTrigger,
) {
	data[references.DungeonTriggerTileRow*references.CombatMapBytesPerRow+
		references.DungeonTriggerXOffset+slotIndex] = byte(trigger.ReplacementTile)
	data[references.DungeonTriggerCoordinateRow*references.CombatMapBytesPerRow+
		references.DungeonTriggerXOffset+slotIndex] = byte(trigger.Trigger.X)
	data[references.DungeonTriggerCoordinateRow*references.CombatMapBytesPerRow+
		references.DungeonTriggerYOffset+slotIndex] = byte(trigger.Trigger.Y)
	data[references.DungeonTriggerDestination1Row*references.CombatMapBytesPerRow+
		references.DungeonTriggerXOffset+slotIndex] = byte(trigger.Destinations[0].X)
	data[references.DungeonTriggerDestination1Row*references.CombatMapBytesPerRow+
		references.DungeonTriggerYOffset+slotIndex] = byte(trigger.Destinations[0].Y)
	data[references.DungeonTriggerDestination2Row*references.CombatMapBytesPerRow+
		references.DungeonTriggerXOffset+slotIndex] = byte(trigger.Destinations[1].X)
	data[references.DungeonTriggerDestination2Row*references.CombatMapBytesPerRow+
		references.DungeonTriggerYOffset+slotIndex] = byte(trigger.Destinations[1].Y)
}

func syntheticCombatConfig(dataPath string) *config.UltimaVConfiguration {
	gameConfig := &config.UltimaVConfiguration{} //nolint:exhaustruct // Synthetic loader needs only the data path.
	gameConfig.SavedConfigData = &config.UltimaVConfigurationFlags{
		Resolution:   0,
		FullScreen:   false,
		DataFilePath: dataPath,
	}

	return gameConfig
}

func syntheticTriggerEnemyReferences() references.EnemyReferences {
	enemyReferences := make(references.EnemyReferences, references.TotalMonsterTypes)
	enemyReferences[references.MonsterOrc] = references.EnemyReference{ //nolint:exhaustruct // Focused combat fixture.
		MonsterType:  references.MonsterOrc,
		Name:         "Orc",
		NamePlural:   "Orcs",
		Strength:     20,
		Dexterity:    20,
		Intelligence: 20,
		Armour:       1,
		Damage:       10,
		HitPoints:    100,
		AttackRange:  9,
		MaxPerMap:    8,
		EnemyAbilities: map[references.EnemyAbility]bool{
			references.Immortal: true,
		},
	}

	return enemyReferences
}

func addSyntheticTriggerMonster(
	state *GameState,
	position references.Position,
	loyalty CombatMonsterLoyalty,
	missile references.MissileType,
) (*CombatMonster, int) {
	reference := (*state.GameReferences.EnemyReferences)[references.MonsterOrc]
	reference.LargeMapMissile = missile
	monster := NewCombatMonster(references.MonsterOrc, &reference, position, newFixedRNG(1))
	monster.SetLoyalty(loyalty)

	return monster, state.CombatScene.AddMonsterToQueue(monster)
}

func installPendingTriggerAttack(
	state *GameState,
	attacker CombatTurnIdentity,
	source CombatAttackSource,
	launch CombatProjectileLaunch,
) {
	state.CombatScene.AttackTargeting = &CombatAttackTargeting{
		AttackerIndex: attacker.EntityIndex,
		Attacker:      attacker,
		Sources:       []CombatAttackSource{source},
		SourceIndex:   0,
		Origin:        launch.From,
		Cursor:        launch.Aim,
	}
	state.CombatScene.pendingAttack = &pendingCombatAttack{
		attacker: attacker,
		source:   source,
		launch:   launch,
		diagnostics: CombatDiagnostics{
			Source:  source.Equipment,
			Missile: source.Missile,
		},
	}
}

func assertDungeonTriggerConsumed(t *testing.T, state *GameState, slotIndex int) {
	t.Helper()
	slots, present := state.CombatScene.CombatMapRef.DungeonTriggerSlots()
	require.True(t, present)
	assert.Equal(t, references.Position{
		X: references.DungeonTriggerConsumedPosition,
		Y: references.DungeonTriggerConsumedPosition,
	}, slots[slotIndex].Trigger)
}

func assertDungeonTriggerUnconsumed(
	t *testing.T,
	state *GameState,
	slotIndex int,
	want references.Position,
) {
	t.Helper()
	slots, present := state.CombatScene.CombatMapRef.DungeonTriggerSlots()
	require.True(t, present)
	assert.Equal(t, want, slots[slotIndex].Trigger)
}
