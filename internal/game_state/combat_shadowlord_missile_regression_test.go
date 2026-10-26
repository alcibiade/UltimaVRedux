package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestShadowlordOccupiedBoundaryMissile(t *testing.T) {
	for _, test := range []struct {
		name      string
		invisible bool
		empty     bool
		blocked   bool
	}{
		{name: "visible occupant clears sampled boundary"},
		{name: "invisible occupant leaves boundary", invisible: true, blocked: true},
		{name: "empty boundary remains blocking", empty: true, blocked: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, _, source := newSmallMapShadowlordEncounterFixture(t, references.Faulinei,
				SmallMapEncounterAutomaticShadowlord, references.Position{X: 10, Y: 10}, references.Position{X: 10, Y: 9})
			_, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterShadowlordAdjacency)
			require.NoError(t, err)
			combat := state.CombatScene
			from := references.Position{X: 1, Y: 2}
			to := references.Position{X: 2, Y: 1}
			require.Equal(t, indexes.SpriteIndex(indexes.Grass), combat.CombatMapRef.GetTileNumber(from))
			require.Equal(t, indexes.SpriteIndex(indexes.ShadowlordBoundary3), combat.CombatMapRef.GetTileNumber(to))
			// This diagonal arrow samples the destination before original projectile rules's final-sample
			// exception. A straight shot may accidentally hide the missing overlay.
			// See ref:ENEMY-0007.
			impact, clear := combatMissileImpactPosition(combat.CombatMapRef.GetTileNumber, from, to, references.MissileArrow)
			require.False(t, clear)
			require.Equal(t, to, impact)
			shadowlord := combat.MonsterManager.GetMonster(0)
			require.NotNil(t, shadowlord)
			shadowlord.SetPosition(to)
			shadowlord.SetInvisible(test.invisible)
			if test.empty {
				shadowlord.SetPosition(references.Position{X: 5, Y: 5})
			}
			actor := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
			combat.SetPartyMemberPosition(0, from)
			combat.CurrentTurnIdentity = actor
			combat.currentTurnIdentitySet = true
			character := state.PartyState.GetCharacter(0)
			character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
			character.SetEquippedItem(references.SlotShield, references.NoEquipment)
			character.SetEquippedItem(references.SlotWeapon, references.Bow)
			state.PartyState.Inventory.Equipment.Set(references.Arrows, 10)
			require.True(t, state.BeginCombatAttack())
			combat.AttackTargeting.Cursor = to
			preview := state.PreviewCombatAttack()
			require.NotNil(t, preview.Aim)
			assert.Equal(t, test.blocked, preview.Aim.Status == CombatShotBlocked)
			// Commit a coordinate shot to avoid a random hit/scatter moving this
			// precisely sampled path. Impact still resolves the actual occupant.
			result := state.preparePendingCombatAttack(combat.AttackTargeting,
				combat.AttackTargeting.CurrentSource(), CombatTurnIdentity{}, false)
			require.NotNil(t, result.Launch)
			assert.Equal(t, test.blocked, result.Launch.Blocked)
			assert.Equal(t, !test.blocked, combat.pendingAttack.impactTargetSet)
		})
	}
}

func TestShadowlordArenaBlockedClearStatusSequence(t *testing.T) {
	state, callbacks, source := newSmallMapShadowlordEncounterFixture(t, references.Faulinei,
		SmallMapEncounterAutomaticShadowlord, references.Position{X: 10, Y: 10}, references.Position{X: 10, Y: 9})
	_, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterShadowlordAdjacency)
	require.NoError(t, err)
	combat := state.CombatScene
	shadowlord := combat.MonsterManager.GetMonster(0)
	require.NotNil(t, shadowlord)
	require.Equal(t, references.Position{X: 5, Y: 5}, shadowlord.Position())
	require.Equal(t, references.Position{X: 3, Y: 8}, combat.PartyPositions[0])
	for index := 1; index < party_state.MaxActiveParty; index++ {
		combat.PartyActive[index] = false
		combat.TurnQueue.MarkFled(index, true)
	}
	character := state.PartyState.GetCharacter(0)
	character.SetMaxHp(100)
	character.SetHp(100)
	character.SetStatus(party_state.Good)
	character.Dexterity = 1
	for _, slot := range []references.EquipmentSlot{references.SlotArmour, references.SlotShield, references.SlotHelm, references.SlotAmulet} {
		character.SetEquippedItem(slot, references.NoEquipment)
	}
	// Select a reproducible turn which passes the raw-slot possession and
	// invisibility opportunities, then attempts a ranged hit without scattering.
	const attackSeed = 25
	state.SetRandomSeed(attackSeed)
	callbacks.Reset()
	entry := combat.TurnQueue.GetEntryByIdentity(CombatTurnIdentity{EntityIndex: 0})
	result := state.executeMonsterTurn(entry)
	require.NotNil(t, result.ProjectileLaunch)
	require.True(t, result.ProjectileLaunch.Blocked)
	assert.False(t, result.Moved)
	assert.False(t, result.Teleported)
	assert.Equal(t, references.Position{X: 5, Y: 5}, shadowlord.Position())
	blockedTile := combat.CombatMapRef.GetTileNumber(result.ProjectileLaunch.Impact)
	assert.GreaterOrEqual(t, int(blockedTile), indexes.ShadowlordBoundary1)
	assert.LessOrEqual(t, int(blockedTile), indexes.ShadowlordBoundary16)
	calls := state.GetRNG().Calls()
	require.True(t, state.ResolvePendingMonsterMissile())
	assert.Equal(t, calls, state.GetRNG().Calls(), "blocked impact consumes no status/damage RNG")
	assert.Equal(t, uint16(100), character.GetCurrentHp())
	assert.Equal(t, party_state.Good, character.GetStatus())
	t.Logf("seed=%d blocked %v -> %v at %v tile=%02x; HP=100, no movement or status", attackSeed,
		result.ProjectileLaunch.From, result.ProjectileLaunch.Aim, result.ProjectileLaunch.Impact, blockedTile)

	shadowlord.SetPosition(references.Position{X: 4, Y: 7})
	combat.SetPartyMemberPosition(0, references.Position{X: 4, Y: 9})
	for _, poisonRoll := range []int{0, 1} {
		state.SetRandomSeed(attackSeed)
		result = state.executeMonsterTurn(entry)
		require.NotNil(t, result.ProjectileLaunch)
		require.False(t, result.ProjectileLaunch.Blocked)
		state.SetRandomSeed(seedForFirstIntnValue(t, state, 4, poisonRoll))
		callbacks.Reset()
		require.True(t, state.ResolvePendingMonsterMissile())
		require.Len(t, callbacks.CombatEvents, 1)
		event := callbacks.CombatEvents[0]
		assert.True(t, event.Diagnostics.PoisonRolled)
		assert.Equal(t, poisonRoll, event.Diagnostics.PoisonRoll)
		assert.Equal(t, uint16(70), character.GetCurrentHp(), "poison replaces the second damage application")
		assert.Equal(t, poisonRoll != 0, character.IsPoisoned())
		t.Logf("clear %v -> %v; poison roll=%d HP=%d status=%v", result.ProjectileLaunch.From,
			result.ProjectileLaunch.Aim, poisonRoll, character.GetCurrentHp(), character.GetStatus())
	}
	assert.Equal(t, 30, shadowlord.EnemyRef().Damage)
	assert.Equal(t, 99, shadowlord.EnemyRef().HitPoints)
	assert.Equal(t, 9, shadowlord.EnemyRef().AttackRange)
	assert.GreaterOrEqual(t, shadowlord.CurrentDex(), 26)
	assert.LessOrEqual(t, shadowlord.CurrentDex(), 30)
	assert.Equal(t, 36-shadowlord.CurrentDex(), shadowlord.TurnDelay())
}

func TestCombatMissileOverlayRespectsObjectVisibility(t *testing.T) {
	position := references.Position{X: 2, Y: 1}
	for _, test := range []struct {
		name    string
		terrain indexes.SpriteIndex
		setup   func(*GameState)
		blocked bool
	}{
		{name: "empty wall", blocked: true},
		{name: "visible monster", setup: func(state *GameState) {
			addSyntheticTriggerMonster(state, position, CombatMonsterLoyaltyMonster, references.MissileNone)
		}},
		{name: "invisible monster", blocked: true, setup: func(state *GameState) {
			monster, _ := addSyntheticTriggerMonster(state, position, CombatMonsterLoyaltyMonster, references.MissileNone)
			monster.SetInvisible(true)
		}},
		{name: "retired monster", blocked: true, setup: func(state *GameState) {
			monster, _ := addSyntheticTriggerMonster(state, position, CombatMonsterLoyaltyMonster, references.MissileNone)
			monster.SetActive(false)
		}},
		{name: "invisible PC still has object overlay", setup: func(state *GameState) {
			state.CombatScene.SetPartyMemberPosition(0, position)
			state.CombatScene.PartyInvisible[0] = true
		}},
		{name: "corpse", setup: func(state *GameState) {
			require.True(t, state.CombatScene.AddObject(NewCombatCorpse(position)))
		}},
		{name: "protection field", setup: func(state *GameState) {
			require.True(t, state.CombatScene.AddField(NewCombatField(CombatFieldProtection, position)))
		}},
		{name: "arch conceals object but permits missiles", terrain: indexes.BrickWallArchway, setup: func(state *GameState) {
			addSyntheticTriggerMonster(state, position, CombatMonsterLoyaltyMonster, references.MissileNone)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, _ := newSyntheticDungeonTriggerCombatState(t)
			raw := brickFloorCombatRaw()
			tile := test.terrain
			if tile == 0 {
				tile = indexes.StoneBrickWall
			}
			raw[position.X][position.Y] = byte(tile)
			state.CombatScene.CombatMapRef.SetRawData(raw)
			if test.setup != nil {
				test.setup(state)
			}
			launch := traceCombatProjectileLaunch(state.combatMissileTerrainAt,
				references.Position{X: 1, Y: 2}, position, references.MissileArrow)
			assert.Equal(t, test.blocked, launch.Blocked)
			if tile == indexes.BrickWallArchway {
				assert.Equal(t, tile, state.combatMissileTerrainAt(position))
			}
			assert.Equal(t, tile, state.CombatScene.CombatMapRef.GetTileNumber(position), "compositing must not erase terrain")
		})
	}
}

func TestShadowlordArenaSpecialActionsCrossBarriers(t *testing.T) {
	state, callbacks, source := newSmallMapShadowlordEncounterFixture(t, references.Faulinei,
		SmallMapEncounterAutomaticShadowlord, references.Position{X: 10, Y: 10}, references.Position{X: 10, Y: 9})
	_, err := state.StartSmallMapCombatEncounter(source, SmallMapEncounterShadowlordAdjacency)
	require.NoError(t, err)
	combat := state.CombatScene
	shadowlord := combat.MonsterManager.GetMonster(0)
	require.NotNil(t, shadowlord)
	launch := traceCombatProjectileLaunch(state.combatMissileTerrainAt, shadowlord.Position(), combat.PartyPositions[0], references.MissileRed)
	require.True(t, launch.Blocked)
	state.PartyState.GetCharacter(0).Intelligence = 1
	callbacks.Reset()
	rng := newSequenceRNG(0, 29)
	require.True(t, state.executeHostileMonsterSpecialMoveWithRNG(shadowlord, 0, rng))
	assert.True(t, combat.PartyPossessed[0], "possession is not a missile and crosses the boundary")
	assert.True(t, shadowlord.IsActive())
	assert.Equal(t, 2, rng.index)
	require.Len(t, callbacks.CombatEvents, 1)
	assert.Equal(t, CombatEventPossession, callbacks.CombatEvents[0].Kind)

	callbacks.Reset()
	rng = newSequenceRNG(31, 31)
	require.True(t, state.executeHostileMonsterSpecialMoveWithRNG(shadowlord, 0, rng))
	assert.True(t, shadowlord.IsInvisible())
	assert.Equal(t, 2, rng.index)
	require.Len(t, callbacks.CombatEvents, 1)
	assert.Equal(t, CombatEventVisibility, callbacks.CombatEvents[0].Kind)

	callbacks.Reset()
	destination := references.Position{X: 2, Y: 1}
	require.Equal(t, indexes.SpriteIndex(indexes.ShadowlordBoundary3), combat.CombatMapRef.GetTileNumber(destination))
	rng = newSequenceRNG(0, int(destination.X), int(destination.Y))
	require.True(t, state.tryMonsterTeleportWithRNG(shadowlord, 0, false, rng))
	assert.Equal(t, destination, shadowlord.Position(), "ether movement admits an unoccupied boundary")
	assert.Equal(t, 3, rng.index)
	require.Len(t, callbacks.CombatEvents, 1)
	assert.Equal(t, CombatEventTeleport, callbacks.CombatEvents[0].Kind)
}
