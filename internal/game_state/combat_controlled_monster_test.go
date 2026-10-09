package game_state

import (
	"fmt"
	"maps"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/deterministic_random"
	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

func TestMagePhysicalDamagePolicyMatchesHostileAndControlledAttacks(t *testing.T) {
	for _, test := range []struct {
		name     string
		hp       int
		immortal bool
		poison   bool
	}{
		{name: "undead", hp: 200},
		{name: "low HP flee", hp: 40},
		{name: "half HP condition roll", hp: 80},
		{name: "immunity", hp: 200, immortal: true},
		{name: "immunity still reevaluates flee", hp: 40, immortal: true},
		{name: "poison fallback undead", hp: 200, poison: true},
		{name: "poison fallback flee", hp: 40, poison: true},
		{name: "poison fallback immunity", hp: 40, poison: true, immortal: true},
	} {
		for _, controlled := range []bool{false, true} {
			for _, ranged := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/controlled=%t/ranged=%t", test.name, controlled, ranged), func(t *testing.T) {
					gs, mock := setupControlledMonsterTest(t)
					require.NotNil(t, gs)
					attacker, attackerIndex := addControlledTestMonster(gs, references.MonsterMage,
						references.Position{X: 5, Y: 5}, CombatMonsterLoyaltyMonster)
					attacker.currentDex = 255 // Isolate damage policy from hit probability.
					attacker.enemyRef.Damage = 20
					attacker.enemyRef.AttackRange = 4
					require.True(t, attacker.HasAbility(references.RangedMagic))
					attacker.enemyRef.EnemyAbilities[references.Poison] = test.poison
					position := references.Position{X: 6, Y: 5}
					if ranged {
						position.X = 8
					}
					target, targetIndex := addControlledTestMonster(gs, references.MonsterSkeleton, position, CombatMonsterLoyaltyParty)
					target.enemyRef.HitPoints = 200
					target.enemyRef.Armour = 0
					target.enemyRef.EnemyAbilities[references.Immortal] = test.immortal
					target.SetCurrentHP(test.hp)
					if controlled {
						attacker.SetLoyalty(CombatMonsterLoyaltyParty)
						target.SetLoyalty(CombatMonsterLoyaltyMonster)
						authorizeControlledMonster(gs, attackerIndex)
					}
					for x := references.Coordinate(5); x <= 8; x++ {
						floor := references.Position{X: x, Y: 5}
						gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &floor, indexes.BrickFloor)
						gs.CombatScene.CombatMapRef.SetTileNumber(floor, indexes.BrickFloor)
					}
					// A hostile ranged turn alone has the missile-choice roll. All paths
					// then share physical hit, selected poison damage, and condition order.
					var seed uint64
					var oracle *deterministic_random.Source
					wantDamage := 10 // 20 fixed physical damage halved against Skeleton.
					for seed = 1; seed < 1000; seed++ {
						oracle = deterministic_random.New(seed)
						if !controlled && ranged && oracle.Intn(256) >= 128 {
							continue
						}
						rollOriginalD30(oracle)
						if test.poison {
							if oracle.Intn(4) == 0 {
								continue
							}
							wantDamage = oracle.Intn(21) / 2
							if wantDamage == 0 {
								continue
							}
						}
						break
					}
					require.Less(t, seed, uint64(1000))
					if test.immortal {
						wantDamage = 0
					}
					wantHP := test.hp - wantDamage
					wantFlee := wantHP < 50
					if wantHP >= 50 && wantHP < 100 {
						wantFlee = oracle.Intn(256) > 251
					}
					gs.SetRandomSeed(seed)
					mock.Reset()

					if controlled {
						require.True(t, gs.BeginCombatAttack())
						gs.CombatScene.AttackTargeting.Cursor = position
						result := gs.ConfirmCombatAttackTarget()
						if ranged {
							require.Equal(t, CombatAttackSourcePending, result.Outcome)
							result = gs.ResolvePendingCombatAttack()
						}
						assert.Equal(t, CombatAttackSequenceComplete, result.Outcome)
					} else {
						nearest := nearestPCResult{found: true, identity: CombatTurnIdentity{EntityIndex: targetIndex}, distance: integerDistance(attacker.Position(), position)}
						require.True(t, gs.monsterAttackWithRNG(attacker, attackerIndex, nearest, gs.AsRandomProvider()))
					}

					assert.Equal(t, wantHP, target.CurrentHP())
					assert.Equal(t, wantFlee, target.IsFleeing())
					assert.Equal(t, oracle.Calls(), gs.GetRNG().Calls(), "classification and low-HP flee must not add RNG")
					assert.Equal(t, oracle.Intn(10001), gs.RandomIntInRange(0, 10000))
					require.Len(t, mock.CombatEvents, 1)
					event := mock.CombatEvents[0]
					assert.False(t, event.Diagnostics.ProjectileBlocked)
					assert.Equal(t, wantDamage, event.HPLost)
					assert.False(t, event.Diagnostics.MagicDamage)
					assert.False(t, event.Diagnostics.MagicWeapon)
					assert.Equal(t, test.poison, event.Diagnostics.PoisonFallbackRolled)
					assert.Equal(t, wantDamage, event.Diagnostics.FinalDamage)
				})
			}
		}
	}
}

func setupControlledMonsterTest(t *testing.T) (*GameState, *MockSystemCallbacks) {
	t.Helper()

	gs, mock := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(1).
		Build()
	if gs == nil {
		return nil, nil
	}
	require.NoError(t, gs.TransitionToCombatMap(
		0,
		references.BritanniaCombatTerritory,
		CombatEntryNorth,
	))
	clearCombatMonstersForTest(gs.CombatScene)
	gs.CombatScene.TurnQueue = NewCombatTurnQueue()
	gs.CombatScene.turnsStarted = true
	mock.Reset()

	return gs, mock
}

func addControlledTestMonster(
	gs *GameState,
	monsterType references.MonsterType,
	position references.Position,
	loyalty CombatMonsterLoyalty,
) (*CombatMonster, int) {
	enemyReferences := *gs.GameReferences.EnemyReferences
	ref := enemyReferences[monsterType]
	ref.EnemyAbilities = maps.Clone(ref.EnemyAbilities)
	monster := NewCombatMonster(
		monsterType,
		&ref,
		position,
		newFixedRNG(1),
	)
	monster.SetLoyalty(loyalty)

	return monster, gs.CombatScene.AddMonsterToQueue(monster)
}

func authorizeControlledMonster(gs *GameState, monsterIndex int) {
	gs.CombatScene.CurrentTurnIdentity = CombatTurnIdentity{
		IsPartyMember: false,
		EntityIndex:   monsterIndex,
	}
	gs.CombatScene.currentTurnIdentitySet = true
	gs.CombatScene.ActiveMonsterIndex = monsterIndex
	gs.ClearActivePlayer()
}

func TestAdvanceCombatTurnRoutesPartyLoyalMonsterToPlayerInput(t *testing.T) {
	gs, mock := setupControlledMonsterTest(t)
	if gs == nil {
		return
	}

	loyal, loyalIndex := addControlledTestMonster(
		gs, references.MonsterOrc, references.Position{X: 5, Y: 5}, CombatMonsterLoyaltyParty,
	)
	loyal.currentDex = 35
	hostile, hostileIndex := addControlledTestMonster(
		gs, references.MonsterTroll, references.Position{X: 8, Y: 8}, CombatMonsterLoyaltyMonster,
	)
	hostile.currentDex = 1
	gs.CombatScene.TurnQueue.GetEntry(loyalIndex, false).turnCountdown = 1
	gs.CombatScene.TurnQueue.GetEntry(hostileIndex, false).turnCountdown = turnCountdownBase
	require.True(t, gs.ActivateQuicknessEffect(10), "quickness must not suppress a party-loyal turn")
	mock.Reset()

	gs.AdvanceCombatTurn()

	assert.False(t, gs.CombatScene.HasScheduledCombatStep())
	assert.Equal(t, CombatTurnIdentity{IsPartyMember: false, EntityIndex: loyalIndex},
		gs.CombatScene.GetCurrentTurnIdentity())
	assert.Equal(t, loyalIndex, gs.CombatScene.ActiveMonsterIndex)
	assert.Equal(t, -1, gs.GetActivePlayerIndex())
	assert.Contains(t, mock.Messages, loyal.MonsterType().Name()+":")
	for _, event := range mock.CombatEvents {
		assert.NotEqual(t, CombatConditionQuicknessSuppressed, event.Condition)
	}
}

func TestSoloModeAutoPassesPartyLoyalMonster(t *testing.T) {
	gs, _ := setupControlledMonsterTest(t)
	if gs == nil {
		return
	}
	pcProvider := NewCombatPCProvider(gs.PartyState.GetCharacter(0))
	pcProvider.sleeping = &gs.CombatScene.PartySleeping[0]
	pcProvider.underground = &gs.CombatScene.PartyUnderground[0]
	gs.CombatScene.TurnQueue.AddCombatant(pcProvider, 0)
	monster, monsterIndex := addControlledTestMonster(
		gs, references.MonsterOrc, references.Position{X: 5, Y: 5}, CombatMonsterLoyaltyParty,
	)
	gs.CombatScene.SetSoloMoveCharacter(0)
	gs.CombatScene.presentTurns = true
	selectedEntry := gs.CombatScene.TurnQueue.GetEntry(0, true)
	require.NotNil(t, selectedEntry)
	selectedEntry.turnCountdown = 1
	entry := gs.CombatScene.TurnQueue.GetEntry(monsterIndex, false)
	require.NotNil(t, entry)
	entry.turnCountdown = 1
	gs.CombatScene.TurnQueue.currentIndex = 1

	gs.AdvanceCombatTurn()

	assert.False(t, gs.CombatScene.HasScheduledCombatStep())
	assert.Equal(t, CombatTurnIdentity{IsPartyMember: true, EntityIndex: 0},
		gs.CombatScene.GetCurrentTurnIdentity())
	assert.True(t, monster.IsActive())
}

func TestAdvanceCombatTurnAnonymizesConcealedPartyLoyalMonster(t *testing.T) {
	gs, mock := setupControlledMonsterTest(t)
	if gs == nil {
		return
	}
	position := references.Position{X: 5, Y: 5}
	loyal, loyalIndex := addControlledTestMonster(
		gs, references.MonsterOrc, position, CombatMonsterLoyaltyParty,
	)
	loyal.currentDex = 35
	gs.CombatScene.TurnQueue.GetEntry(loyalIndex, false).turnCountdown = 1
	gs.GetCurrentLayeredMap().SetTileByLayer(map_state.MapLayer, &position, indexes.BlackSquare)
	mock.Reset()

	gs.AdvanceCombatTurn()

	assert.Equal(t, loyalIndex, gs.CombatScene.ActiveMonsterIndex,
		"concealment must not remove player control or change scheduling")
	assert.Contains(t, mock.Messages, "Unknown:")
	assert.NotContains(t, mock.Messages, loyal.MonsterType().Name()+":")
}

func TestControlledMonsterMovementAndPassUseItsStableQueueIdentity(t *testing.T) {
	gs, mock := setupControlledMonsterTest(t)
	if gs == nil {
		return
	}

	origin := references.Position{X: 5, Y: 5}
	first, _ := addControlledTestMonster(gs, references.MonsterInsect, origin, CombatMonsterLoyaltyParty)
	second, secondIndex := addControlledTestMonster(gs, references.MonsterInsect, origin, CombatMonsterLoyaltyParty)
	authorizeControlledMonster(gs, secondIndex)

	var direction references.Direction
	for _, candidate := range []references.Direction{references.Up, references.Right, references.Down, references.Left} {
		position := candidate.GetNewPositionInDirection(&origin)
		if position != nil && isCombatPositionValidForMonster(gs, *position, second, secondIndex) {
			direction = candidate

			break
		}
	}
	require.NotEqual(t, references.NoneDirection, direction)

	assert.Equal(t, CombatMoveMoved, gs.MoveCombatCharacter(direction))
	assert.Equal(t, origin, first.Position(), "the lower stack member must not move")
	assert.NotEqual(t, origin, second.Position(), "the authorized stack member must move")
	require.NotEmpty(t, mock.CombatEvents)
	assert.Equal(t, secondIndex, mock.CombatEvents[0].Attacker.Index)

	gs.FinishCombatAction()
	assert.NotEqual(t, CombatTurnIdentity{IsPartyMember: false, EntityIndex: secondIndex},
		gs.CombatScene.GetCurrentTurnIdentity(), "passing must consume the controlled monster turn")
}

func TestPCAttackRejectsPartyLoyalMonsterAsFriendly(t *testing.T) {
	gs, mock := setupControlledMonsterTest(t)
	if gs == nil {
		return
	}

	pcIndex := gs.CombatScene.GetActiveCombatantIndex()
	gs.CombatScene.CurrentTurnIdentity = CombatTurnIdentity{IsPartyMember: true, EntityIndex: pcIndex}
	gs.CombatScene.currentTurnIdentitySet = true
	origin := gs.CombatScene.GetPartyMemberPosition(pcIndex)
	targetPosition := references.Position{X: origin.X + 1, Y: origin.Y}
	loyal, _ := addControlledTestMonster(gs, references.MonsterOrc, targetPosition, CombatMonsterLoyaltyParty)
	hpBefore := loyal.CurrentHP()
	character := gs.PartyState.GetCharacter(pcIndex)
	character.SetEquippedItem(references.SlotHelm, references.NoEquipment)
	character.SetEquippedItem(references.SlotWeapon, references.NoEquipment)
	character.SetEquippedItem(references.SlotShield, references.NoEquipment)

	require.True(t, gs.BeginCombatAttack())
	gs.CombatScene.AttackTargeting.Cursor = targetPosition
	assert.Equal(t, CombatAimFriendly, gs.GetCombatAimDisposition(targetPosition))
	assert.Equal(t, CombatAttackSequenceComplete, gs.ConfirmCombatAttackTarget().Outcome)
	assert.Equal(t, hpBefore, loyal.CurrentHP())
	assert.Contains(t, mock.Messages, "Can't!")
	assert.Empty(t, mock.CombatEvents)
}

func TestControlledMonsterMeleePublishesMonsterIdentitiesAndAwardsNoXP(t *testing.T) {
	gs, mock := setupControlledMonsterTest(t)
	if gs == nil {
		return
	}

	attackerRef := &references.EnemyReference{ //nolint:exhaustruct // Focused deterministic combat fixture.
		MonsterType: references.MonsterOrc,
		Name:        "Controlled Orc",
		Damage:      100,
		Dexterity:   100,
		HitPoints:   20,
		AttackRange: 1,
	}
	targetRef := &references.EnemyReference{ //nolint:exhaustruct // Focused deterministic combat fixture.
		MonsterType: references.MonsterTroll,
		Name:        "Target Troll",
		Dexterity:   1,
		HitPoints:   1,
	}
	attacker := NewCombatMonster(
		references.MonsterOrc, attackerRef, references.Position{X: 5, Y: 5}, newFixedRNG(1),
	)
	attacker.SetLoyalty(CombatMonsterLoyaltyParty)
	attackerIndex := gs.CombatScene.AddMonsterToQueue(attacker)
	target := NewCombatMonster(
		references.MonsterTroll, targetRef, references.Position{X: 6, Y: 5}, newFixedRNG(1),
	)
	targetIndex := gs.CombatScene.AddMonsterToQueue(target)
	authorizeControlledMonster(gs, attackerIndex)
	experienceBefore := make([]uint16, gs.PartyState.GetCharacterCount())
	for index := range experienceBefore {
		experienceBefore[index] = gs.PartyState.GetCharacter(index).Exp
	}

	require.True(t, gs.BeginCombatAttack())
	gs.CombatScene.AttackTargeting.Cursor = target.Position()
	assert.Equal(t, CombatAttackSequenceComplete, gs.ConfirmCombatAttackTarget().Outcome)
	assert.Equal(t, []SoundEffect{SoundCombatSwing}, mock.SoundEffectsPlayed)
	require.True(t, target.IsDead())
	require.Len(t, mock.CombatEvents, 1)
	event := mock.CombatEvents[0]
	assert.Equal(t, CombatEventAttack, event.Kind)
	assert.Equal(t, CombatFeedbackEntity{IsPartyMember: false, Index: attackerIndex, Position: attacker.Position()}, event.Attacker)
	assert.Equal(t, targetIndex, event.Target.Index)
	assert.False(t, event.Target.IsPartyMember)
	for index, experience := range experienceBefore {
		assert.Equal(t, experience, gs.PartyState.GetCharacter(index).Exp)
	}
}

func TestControlledRedDaemonUsesRangeNineRedMissile(t *testing.T) {
	gs, mock := setupControlledMonsterTest(t)
	if gs == nil {
		return
	}

	var from, to references.Position
	foundClearRangeNine := false
	for y := range references.YCombatMapTiles {
		for x := 0; x+9 < int(references.XCombatMapTiles); x++ {
			candidateFrom := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
			candidateTo := references.Position{X: references.Coordinate(x + 9), Y: references.Coordinate(y)}
			impact, clear := combatMissileImpactPosition(
				gs.combatMissileTerrainAt, candidateFrom, candidateTo, references.MissileRed,
			)
			if clear && impact == candidateTo {
				from, to = candidateFrom, candidateTo
				foundClearRangeNine = true

				break
			}
		}
		if foundClearRangeNine {
			break
		}
	}
	require.True(t, foundClearRangeNine)
	daemon, daemonIndex := addControlledTestMonster(
		gs, references.MonsterDevil, from, CombatMonsterLoyaltyParty,
	)
	_, targetIndex := addControlledTestMonster(gs, references.MonsterOrc, to, CombatMonsterLoyaltyMonster)
	authorizeControlledMonster(gs, daemonIndex)

	require.True(t, gs.BeginCombatAttack())
	targeting := gs.CombatScene.AttackTargeting
	require.True(t, targeting.IsMonsterAttacker())
	assert.Equal(t, 9, targeting.CurrentSource().MaxRange)
	assert.Equal(t, references.MissileRed, targeting.CurrentSource().Missile)
	targeting.Cursor = to
	result := gs.ConfirmCombatAttackTarget()
	require.Equal(t, CombatAttackSourcePending, result.Outcome)
	require.NotNil(t, result.Launch)
	assert.Equal(t, CombatAttackSequenceComplete, gs.ResolvePendingCombatAttack().Outcome)
	assert.NotContains(t, mock.SoundEffectsPlayed, SoundCombatSwing, "ranged attacks do not use the melee opening cue")
	require.Len(t, mock.CombatEvents, 1)
	assert.Equal(t, daemonIndex, mock.CombatEvents[0].Attacker.Index)
	assert.Equal(t, targetIndex, mock.CombatEvents[0].Target.Index)
	assert.Equal(t, daemon.Position(), mock.CombatEvents[0].From)
	assert.Equal(t, to, mock.CombatEvents[0].To)
	assert.NotContains(t, mock.Messages, "Ranged attacks not implemented!")
}

func TestControlledSummonCanAttackPossessedPC(t *testing.T) {
	gs, mock := setupControlledMonsterTest(t)
	if gs == nil {
		return
	}
	attacker, attackerIndex := addControlledTestMonster(
		gs, references.MonsterDevil, references.Position{X: 5, Y: 5}, CombatMonsterLoyaltyParty,
	)
	targetIndex := 0
	targetPosition := references.Position{X: 6, Y: 5}
	gs.CombatScene.PartyActive[targetIndex] = true
	gs.CombatScene.SetPartyMemberPosition(targetIndex, targetPosition)
	gs.CombatScene.PartyPossessed[targetIndex] = true
	gs.PartyState.GetCharacter(targetIndex).SetStatus(party_state.Charmed)
	gs.PartyState.GetCharacter(targetIndex).Dexterity = 1
	authorizeControlledMonster(gs, attackerIndex)
	mock.Reset()

	gs.resolveControlledMonsterAttackAgainstPC(attackerIndex, CombatTurnIdentity{
		IsPartyMember: true,
		EntityIndex:   targetIndex,
	}, targetPosition)

	assert.Equal(t, []SoundEffect{SoundCombatSwing}, mock.SoundEffectsPlayed)
	require.NotEmpty(t, mock.CombatEvents)
	event := mock.CombatEvents[len(mock.CombatEvents)-1]
	assert.Equal(t, CombatEventAttack, event.Kind)
	assert.Equal(t, attackerIndex, event.Attacker.Index)
	assert.True(t, event.Target.IsPartyMember)
	assert.Equal(t, targetIndex, event.Target.Index)
	assert.True(t, attacker.IsPartyLoyal())
}
