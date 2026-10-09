package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestOriginalCombatShapeMovementClasses(t *testing.T) {
	tests := []struct {
		name        string
		shape       int
		terrain     indexes.SpriteIndex
		wantAllowed bool
	}{
		{name: "player chair", shape: originalCombatPlayerShape, terrain: indexes.ChairBackForward, wantAllowed: true},
		{name: "first human class chair", shape: originalCombatMonsterShape(references.MonsterMage), terrain: indexes.ChairBackForward, wantAllowed: true},
		{name: "rat chair", shape: originalCombatMonsterShape(references.MonsterRat), terrain: indexes.ChairBackForward},
		{name: "on foot floor", shape: originalCombatMonsterShape(references.MonsterWisp), terrain: indexes.BrickFloor, wantAllowed: true},
		{name: "on foot water", shape: originalCombatMonsterShape(references.MonsterWisp), terrain: indexes.Water1},
		{name: "on foot wall", shape: originalCombatMonsterShape(references.MonsterWisp), terrain: indexes.StoneBrickWall},
		{name: "water deep", shape: originalCombatMonsterShape(references.MonsterSeaHorse), terrain: indexes.Water1, wantAllowed: true},
		{name: "water river bridge", shape: originalCombatMonsterShape(references.MonsterSeaHorse), terrain: indexes.TrollBridgeHoriz, wantAllowed: true},
		{name: "water floor", shape: originalCombatMonsterShape(references.MonsterSeaHorse), terrain: indexes.BrickFloor},
		{name: "flying water", shape: originalCombatMonsterShape(references.MonsterBat), terrain: indexes.Water2, wantAllowed: true},
		{name: "flying river", shape: originalCombatMonsterShape(references.MonsterDevil), terrain: indexes.WaterStream4, wantAllowed: true},
		{name: "flying floor", shape: originalCombatMonsterShape(references.MonsterDragon), terrain: indexes.BrickFloor, wantAllowed: true},
		{name: "flying wall", shape: originalCombatMonsterShape(references.MonsterMongbat), terrain: indexes.StoneBrickWall},
		{name: "ether wall", shape: originalCombatMonsterShape(references.MonsterGhost), terrain: indexes.StoneBrickWall, wantAllowed: true},
		{name: "ether water", shape: originalCombatMonsterShape(references.MonsterShadowlord), terrain: indexes.WaterCoast},
		{name: "swamp exact", shape: originalCombatMonsterShape(references.MonsterRotWorm), terrain: indexes.Swamp, wantAllowed: true},
		{name: "swamp rejects grass", shape: originalCombatMonsterShape(references.MonsterRotWorm), terrain: indexes.Grass},
		{name: "grass exact", shape: originalCombatMonsterShape(references.MonsterCorpser), terrain: indexes.Grass, wantAllowed: true},
		{name: "grass rejects swamp", shape: originalCombatMonsterShape(references.MonsterCorpser), terrain: indexes.Swamp},
		{name: "deep exact", shape: originalCombatMonsterShape(references.MonsterWhirlpool), terrain: indexes.Water1, wantAllowed: true},
		{name: "deep rejects sea", shape: originalCombatMonsterShape(references.MonsterWhirlpool), terrain: indexes.Water2},
		{name: "desert exact", shape: originalCombatMonsterShape(references.MonsterSandtrap), terrain: indexes.Desert1, wantAllowed: true},
		{name: "desert rejects border", shape: originalCombatMonsterShape(references.MonsterSandtrap), terrain: indexes.LeftDesert2},
		{name: "force field rejected", shape: originalCombatMonsterShape(references.MonsterForceField), terrain: indexes.BrickFloor},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tile := &references.Tile{Index: test.terrain}
			assert.Equal(t, test.wantAllowed, combatTerrainAllowsOriginalShape(test.shape, tile))
		})
	}
}

func TestCombatPCRatUsesRatShapeForChairMovement(t *testing.T) {
	state, _ := newSyntheticCombatFindingState(t)
	identity := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
	destination := references.Position{X: 3, Y: 2}
	state.setCombatTerrain(destination, indexes.ChairBackForward)

	assert.True(t, state.isCombatPositionValidForPC(&destination, identity))
	entry := state.CombatScene.TurnQueue.GetEntryByIdentity(identity)
	require.NotNil(t, entry)
	entry.ratAppearance = true
	assert.False(t, state.isCombatPositionValidForPC(&destination, identity))
}

func TestCombatCloneUsesFixedPlayerPlacementShape(t *testing.T) {
	state, _ := newSyntheticCombatFindingState(t)
	setAllCombatTerrainForTest(state, indexes.Water1)
	_, batIndex := addSyntheticFindingMonster(
		state, references.MonsterBat, references.Position{X: 5, Y: 5}, CombatMonsterLoyaltyMonster,
	)
	state.SetRandomSeed(42)
	callsBefore := state.GetRNG().Calls()

	_, found := state.findCombatClonePosition(CombatTurnIdentity{EntityIndex: batIndex})

	assert.False(t, found, "a flying source does not give Clone a flying placement shape")
	assert.Equal(t, callsBefore+512, state.GetRNG().Calls())
}

func TestCombatKalXenUsesFixedRatPlacementShape(t *testing.T) {
	state, _ := newSyntheticCombatFindingState(t)
	setAllCombatTerrainForTest(state, indexes.Water1)
	values := []int{11} // Select Bat, which could itself fly over the water.
	for range combatSummonPlacementAttempts {
		values = append(values, 5, 5)
	}
	rng := newStrictSummonRNG(t, values...)

	result := state.castCombatKalXen(rng)

	assert.Equal(t, CombatSummonNoLegalPosition, result.Outcome)
	assert.Zero(t, result.Count)
	assert.Equal(t, values, rng.values[:rng.index])
}

func TestControlledMonsterCannotFleeShipCombatIndividually(t *testing.T) {
	for _, test := range []struct {
		name       string
		aboardShip bool
		wantResult CombatMoveResult
	}{
		{name: "ship blocks departure", aboardShip: true, wantResult: CombatMoveBlocked},
		{name: "land permits departure", wantResult: CombatMoveFled},
	} {
		t.Run(test.name, func(t *testing.T) {
			state, callbacks := newSyntheticCombatFindingState(t)
			monster, monsterIndex := addSyntheticFindingMonster(
				state, references.MonsterOrc, references.Position{X: 0, Y: 5}, CombatMonsterLoyaltyParty,
			)
			authorizeControlledMonster(state, monsterIndex)
			if test.aboardShip {
				npcReference := references.NPCReference{}
				npcReference.NpcType = references.Vehicle
				state.PartyVehicle = *map_units.NewNPCFriendlyVehicle(references.FrigateVehicle, npcReference)
			}
			callbacks.Reset()

			assert.Equal(t, test.wantResult, state.MoveCombatCharacter(references.Left))
			entry := state.CombatScene.TurnQueue.GetEntry(monsterIndex, false)
			require.NotNil(t, entry)
			if test.aboardShip {
				assert.True(t, monster.IsActive())
				assert.Equal(t, turnQueueActive, entry.status)
				assert.Equal(t, references.Position{X: 0, Y: 5}, monster.Position())
				assert.Contains(t, callbacks.Messages, "Stay with ship!")
				assert.Empty(t, callbacks.CombatEvents)
			} else {
				assert.False(t, monster.IsActive())
				assert.Equal(t, turnQueueFled, entry.status)
				require.Len(t, callbacks.CombatEvents, 1)
				assert.Equal(t, CombatEventFled, callbacks.CombatEvents[0].Kind)
			}
		})
	}
}

func TestProductionMimicUsesCompoundPoisonMask(t *testing.T) {
	for selector := 1; selector <= 3; selector++ {
		t.Run(string(rune('0'+selector)), func(t *testing.T) {
			state, _ := newSyntheticCombatFindingState(t)
			reference := (*state.GameReferences.EnemyReferences)[references.MonsterMimic]
			require.True(t, reference.HasAbility(references.Poison))
			require.False(t, reference.HasAbility(references.PoisonAtRange))
			mimic := NewCombatMonster(
				references.MonsterMimic, &reference, references.Position{X: 5, Y: 5}, newFixedRNG(1),
			)
			target := CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}
			character := state.PartyState.GetCharacter(0)
			character.SetStatus(party_state.Good)
			rng := newSequenceRNG(selector)
			diagnostics := CombatDiagnostics{}

			result := state.resolveSuccessfulMonsterHit(mimic, target, false, rng, &diagnostics)

			assert.True(t, result.replacedDamage)
			assert.False(t, result.damageSelected)
			assert.Equal(t, CombatFeedbackPoison, result.feedback)
			assert.True(t, character.IsPoisoned())
			assert.Equal(t, selector, diagnostics.PoisonRoll)
			assert.Equal(t, 1, rng.index)
		})
	}

	state, _ := newSyntheticCombatFindingState(t)
	reference := (*state.GameReferences.EnemyReferences)[references.MonsterMimic]
	reference.EnemyAbilities = map[references.EnemyAbility]bool{references.InfectWithPlague: true}
	plagueOnly := NewCombatMonster(
		references.MonsterMimic, &reference, references.Position{X: 5, Y: 5}, newFixedRNG(1),
	)
	rng := newSequenceRNG(3)
	diagnostics := CombatDiagnostics{}

	result := state.resolveSuccessfulMonsterHit(
		plagueOnly, CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}, false, rng, &diagnostics,
	)

	assert.False(t, result.replacedDamage)
	assert.False(t, diagnostics.PoisonRolled)
	assert.Zero(t, rng.index)
}

func TestHostileUsesMagicMissileDealsNonmagicalDamage(t *testing.T) {
	state, callbacks := newSyntheticCombatFindingState(t)
	state.CombatScene.SetPartyMemberPosition(0, references.Position{X: 10, Y: 10})
	mage, mageIndex := addSyntheticFindingMonster(
		state, references.MonsterMage, references.Position{X: 2, Y: 5}, CombatMonsterLoyaltyMonster,
	)
	skeleton, skeletonIndex := addSyntheticFindingMonster(
		state, references.MonsterSkeleton, references.Position{X: 5, Y: 5}, CombatMonsterLoyaltyParty,
	)
	skeleton.SetCurrentHP(20)
	callbacks.Reset()
	nearest := nearestPCResult{
		pcIndex:  NoTargetPC,
		identity: CombatTurnIdentity{EntityIndex: skeletonIndex},
		distance: 3,
		found:    true,
	}

	launch, attacked := state.prepareMonsterAttackWithRNG(
		mage, mageIndex, nearest, newStrictSummonRNG(t, 0, 60),
	)
	require.True(t, attacked)
	require.NotNil(t, launch)
	require.True(t, state.ResolvePendingMonsterMissile())

	assert.Equal(t, 13, skeleton.CurrentHP(), "rating 15 is halved against an undead target")
	require.Len(t, callbacks.CombatEvents, 1)
	event := callbacks.CombatEvents[0]
	assert.False(t, event.Diagnostics.MagicDamage)
	assert.False(t, event.Diagnostics.MagicWeapon)
	assert.True(t, event.Diagnostics.TargetUndead)
	assert.Equal(t, 15, event.Diagnostics.Damage.Damage)
	assert.Equal(t, 7, event.Diagnostics.FinalDamage)
}

func TestHostileUsesMagicMissileRetainsTurningAndNegation(t *testing.T) {
	t.Run("turning forces miss", func(t *testing.T) {
		state, _ := newSyntheticCombatFindingState(t)
		mage, mageIndex := addSyntheticFindingMonster(
			state, references.MonsterMage, references.Position{X: 2, Y: 5}, CombatMonsterLoyaltyMonster,
		)
		state.CombatScene.SetPartyMemberPosition(0, references.Position{X: 5, Y: 5})
		state.PartyState.GetCharacter(0).SetEquippedItem(references.SlotAmulet, references.AmuletOfTurning)
		nearest := nearestPCResult{
			pcIndex: 0, identity: CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}, distance: 3, found: true,
		}
		rng := newStrictSummonRNG(t, 0, 0, 2, 1)

		launch, attacked := state.prepareMonsterAttackWithRNG(mage, mageIndex, nearest, rng)

		require.True(t, attacked)
		require.NotNil(t, launch)
		require.NotNil(t, state.CombatScene.pendingMonsterMissile)
		assert.True(t, state.CombatScene.pendingMonsterMissile.diagnostics.ForcedMiss)
		assert.False(t, state.CombatScene.pendingMonsterMissile.diagnostics.MagicDamage)
		assert.Equal(t, []int{monsterSpecialRollRange, monsterSpecialRollRange, 3, 3}, rng.requests)
	})

	t.Run("negation suppresses launch", func(t *testing.T) {
		state, _ := newSyntheticCombatFindingState(t)
		mage, mageIndex := addSyntheticFindingMonster(
			state, references.MonsterMage, references.Position{X: 2, Y: 5}, CombatMonsterLoyaltyMonster,
		)
		state.CombatScene.SetPartyMemberPosition(0, references.Position{X: 5, Y: 5})
		require.True(t, state.ActivateNegateMagicEffect(3))
		nearest := nearestPCResult{
			pcIndex: 0, identity: CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0}, distance: 3, found: true,
		}
		rng := newStrictSummonRNG(t, 0)

		launch, attacked := state.prepareMonsterAttackWithRNG(mage, mageIndex, nearest, rng)

		assert.False(t, attacked)
		assert.Nil(t, launch)
		assert.Nil(t, state.CombatScene.pendingMonsterMissile)
		assert.Equal(t, []int{monsterSpecialRollRange}, rng.requests)
	})
}

func TestZeroRatedProductionMonsterHitsRemainZero(t *testing.T) {
	for _, monsterType := range []references.MonsterType{references.MonsterChild, references.MonsterBeggar} {
		for _, armored := range []bool{false, true} {
			name := monsterType.Name() + "/unarmored"
			if armored {
				name = monsterType.Name() + "/armored"
			}
			t.Run(name, func(t *testing.T) {
				state, callbacks := newSyntheticCombatFindingState(t)
				monster, monsterIndex := addSyntheticFindingMonster(
					state, monsterType, references.Position{X: 3, Y: 2}, CombatMonsterLoyaltyMonster,
				)
				require.Zero(t, monster.EnemyRef().Damage)
				character := state.PartyState.GetCharacter(0)
				character.SetEquippedItem(references.SlotArmour, references.NoEquipment)
				character.SetEquippedItem(references.SlotShield, references.NoEquipment)
				character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
				character.SetEquippedItem(references.SlotWeapon, references.NoEquipment)
				character.SetEquippedItem(references.SlotRing, references.NoEquipment)
				character.SetEquippedItem(references.SlotAmulet, references.NoEquipment)
				if armored {
					character.SetEquippedItem(references.SlotArmour, references.PlateMail)
				}
				hpBefore := character.GetCurrentHp()
				state.SetRandomSeed(seedForFirstIntnValue(t, state, originalD30SourceValueCount, 60))
				callbacks.Reset()
				nearest := nearestPCResult{
					pcIndex:  0,
					identity: CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0},
					distance: monsterAdjacentDistance,
					found:    true,
				}

				require.True(t, state.monsterMeleeAttack(monster, monsterIndex, nearest))

				assert.Equal(t, hpBefore, character.GetCurrentHp())
				require.Len(t, callbacks.CombatEvents, 1)
				event := callbacks.CombatEvents[0]
				assert.Equal(t, CombatFeedbackGraze, event.Result)
				assert.Zero(t, event.Diagnostics.Damage.AttackRoll)
				assert.Zero(t, event.Diagnostics.Damage.Damage)
				assert.Equal(t, armored, event.Diagnostics.Damage.ArmorRolled)
			})
		}
	}

	rng := newSequenceRNG(7)
	result := CombatCalcDamage(references.NoEquipment, 0, rng)
	assert.Equal(t, 1, result.AttackRoll)
	assert.Equal(t, 1, result.Damage)
	assert.Zero(t, rng.index, "bare hands retain their explicit one-point rating without a draw")
}

func newSyntheticCombatFindingState(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()
	state, callbacks := newSyntheticDungeonTriggerCombatState(t)
	state.GameReferences.EnemyReferences = references.NewAllEnemyReferences(state.GameReferences.TileReferences)

	return state, callbacks
}

func addSyntheticFindingMonster(
	state *GameState,
	monsterType references.MonsterType,
	position references.Position,
	loyalty CombatMonsterLoyalty,
) (*CombatMonster, int) {
	reference := (*state.GameReferences.EnemyReferences)[monsterType]
	monster := NewCombatMonster(monsterType, &reference, position, newFixedRNG(1))
	monster.SetLoyalty(loyalty)

	return monster, state.CombatScene.AddMonsterToQueue(monster)
}
