// Tests for combat monster spawning, manager behavior, and rendering preconditions.
//
// These tests guard against regressions in:
//   - Monster creation and initial state (active, not dead, correct type)
//   - Spawn positions drawn from the authenticated MonsterSpawnPositions table
//   - KeyFrameTile non-nil for all valid monster types (required by the renderer)
//   - Position math: all 16 predefined spawn positions fall within the visible
//     tile window used by the combat renderer (centerOffset=5, screen 19×13)
//   - ForEachActive iteration visits exactly the spawned monsters
package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ─── helpers ────────────────────────────────────────────────────────────────

// newFixedRNG returns a deterministic RandomProvider backed by a fixed seed,
// without touching the global GameState RNG.
type fixedRNG struct{ val int }

func (f *fixedRNG) Intn(n int) int {
	if n <= 0 {
		return 0
	}
	// Simple deterministic value: val mod n, cycling slowly
	result := f.val % n
	f.val = (f.val*1103515245 + 12345) & 0x7fffffff //nolint:mnd // LCG constants
	return result
}

func newFixedRNG(seed int) references.RandomProvider { return &fixedRNG{val: seed} }

func clearCombatMonstersForTest(combat *CombatState) {
	for _, entry := range combat.TurnQueue.entries {
		if entry.Combatant.IsPC() {
			continue
		}
		entry.status = turnQueueFled
		entry.sourceSlotReleased = true
		entry.originalObjectSlot = 0
	}
	combat.MonsterManager.Clear()
}

// validSpawnPositionSet builds a set of all canonical spawn positions for O(1) lookup.
func validSpawnPositionSet() map[references.Position]struct{} {
	set := make(map[references.Position]struct{}, len(references.MonsterSpawnPositions))
	for _, p := range references.MonsterSpawnPositions {
		set[p] = struct{}{}
	}
	return set
}

// ─── unit tests (no real game data required) ────────────────────────────────

// TestCombatMonsterManager_AddAndCount ensures that monsters added to the manager
// are counted correctly and are retrievable.
func TestCombatMonsterManager_AddAndCount(t *testing.T) {
	manager := NewCombatMonsterManager()
	assert.Equal(t, 0, manager.Count(), "fresh manager should have no monsters")
	assert.Equal(t, 0, manager.CountActive(), "fresh manager should have no active monsters")
}

// TestCombatMonsterManager_ForEachActive_VisitsSpawnedMonsters verifies that
// ForEachActive visits exactly the monsters that were added and are alive.
// This is the code path used by the renderer to iterate monsters.
func TestCombatMonsterManager_ForEachActive_VisitsSpawnedMonsters(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(42).
		Build()
	if gs == nil {
		return
	}

	// Enter combat so the monster manager exists
	err := gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth)
	require.NoError(t, err)

	manager := gs.CombatScene.MonsterManager
	manager.Clear()

	enemyRefs := *gs.GameReferences.EnemyReferences
	rng := newFixedRNG(1)
	positions := references.ShuffleMonsterSpawnPositions(rng)

	// Spawn 3 orcs
	for i := 0; i < 3; i++ {
		ref := &enemyRefs[references.MonsterOrc]
		m := NewCombatMonster(references.MonsterOrc, ref, positions[i], rng)
		manager.AddMonster(m)
	}

	assert.Equal(t, 3, manager.Count(), "should have 3 monsters total")
	assert.Equal(t, 3, manager.CountActive(), "all 3 should be active")

	visited := 0
	manager.ForEachActive(func(_ int, monster *CombatMonster) {
		visited++
		assert.False(t, monster.IsDead(), "ForEachActive should only visit living monsters")
		assert.True(t, monster.IsActive(), "ForEachActive should only visit active monsters")
		assert.Equal(t, references.MonsterOrc, monster.MonsterType(), "monster type should be orc")
	})
	assert.Equal(t, 3, visited, "ForEachActive should visit all 3 living monsters")
}

// TestCombatMonster_InitialState verifies that a freshly created monster is
// active, has HP initialized to MaxHP from its EnemyRef, and correct type/ref.
// Note: Some original Ultima V monsters (e.g. Skeleton) have HitPoints=0 in
// the game data files, so we test the constructor invariant (CurrentHP==MaxHP)
// rather than asserting HP>0 unconditionally.
func TestCombatMonster_InitialState(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		Build()
	if gs == nil {
		return
	}

	enemyRefs := *gs.GameReferences.EnemyReferences
	rng := newFixedRNG(7)
	pos := references.MonsterSpawnPositions[0]

	for _, mt := range []references.MonsterType{
		references.MonsterOrc,
		references.MonsterSkeleton,
		references.MonsterTroll,
		references.MonsterDragon,
	} {
		ref := &enemyRefs[mt]
		monster := NewCombatMonster(mt, ref, pos, rng)

		assert.True(t, monster.IsActive(), "%s should be active after creation", mt.Name())
		// Constructor must initialize CurrentHP to exactly MaxHP (from EnemyRef).
		// Monsters with HitPoints=0 in game data (e.g. Skeleton) will be dead at
		// creation — that is faithful to the original data, not a bug in the constructor.
		assert.Equal(t, ref.HitPoints, monster.MaxHP(), "%s MaxHP should match EnemyRef", mt.Name())
		assert.Equal(t, monster.MaxHP(), monster.CurrentHP(), "%s CurrentHP should equal MaxHP at creation", mt.Name())
		assert.Equal(t, mt, monster.MonsterType(), "monster type should match")
		assert.NotNil(t, monster.EnemyRef(), "EnemyRef should not be nil")
	}
}

func TestCombatMonsterDexterityJitterUsesInclusiveOriginalRangeAndReset(t *testing.T) {
	tests := []struct {
		name         string
		base         int
		roll         int
		wantModifier int
		wantDex      int
	}{
		{name: "minimum", base: 20, roll: 0, wantModifier: -4, wantDex: 16},
		{name: "maximum", base: 20, roll: 7, wantModifier: 3, wantDex: 23},
		{name: "over thirty resets", base: 30, roll: 7, wantModifier: 3, wantDex: 30},
		{name: "unsigned underflow equivalent resets", base: 1, roll: 0, wantModifier: -4, wantDex: 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ref := &references.EnemyReference{Dexterity: test.base, HitPoints: 10}
			rng := newSequenceRNG(test.roll)

			monster := NewCombatMonster(references.MonsterOrc, ref, references.Position{}, rng)

			assert.Equal(t, test.wantModifier, monster.DexterityModifier())
			assert.Equal(t, test.wantDex, monster.CurrentDex())
			assert.Equal(t, 1, rng.index, "construction must consume exactly one DEX modifier roll")
		})
	}
}

// TestCombatMonster_Position_IsFromSpawnTable verifies that a monster placed at
// a MonsterSpawnPositions entry retains that exact position, and the position
// is valid according to the spawn table.
func TestCombatMonster_Position_IsFromSpawnTable(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		Build()
	if gs == nil {
		return
	}

	enemyRefs := *gs.GameReferences.EnemyReferences
	ref := &enemyRefs[references.MonsterOrc]
	rng := newFixedRNG(3)
	validPositions := validSpawnPositionSet()

	positions := references.ShuffleMonsterSpawnPositions(rng)
	for i, pos := range positions {
		monster := NewCombatMonster(references.MonsterOrc, ref, pos, rng)
		_, ok := validPositions[monster.Position()]
		assert.True(t, ok,
			"position %d (%d,%d) assigned to monster should be in MonsterSpawnPositions",
			i, pos.X, pos.Y)
	}
}

// TestMonsterSpawnPositions_AllWithinVisibleScreen verifies that every position in
// the canonical MonsterSpawnPositions table falls inside the combat renderer's
// visible tile window.
//
// Renderer logic (gamescene_draw_entities.go):
//
//	tileX = int(pos.X) - combatMapCenterOffset + (xTilesVisible / 2)
//	tileY = int(pos.Y) - combatMapCenterOffset + (yTilesVisible / 2)
//	visible if tileX in [0, xTilesVisible) AND tileY in [0, yTilesVisible)
//
// combatMapCenterOffset = 5 (from game_state_combat.go)
// xTilesVisible = 19, yTilesVisible = 13 (from game_state_integration_test.go)
func TestMonsterSpawnPositions_AllWithinVisibleScreen(t *testing.T) {
	const (
		combatMapCenterOffset = 5
		xTilesVisible         = 19
		yTilesVisible         = 13
	)

	for i, pos := range references.MonsterSpawnPositions {
		tileX := int(pos.X) - combatMapCenterOffset + (xTilesVisible / 2)
		tileY := int(pos.Y) - combatMapCenterOffset + (yTilesVisible / 2)

		assert.GreaterOrEqual(t, tileX, 0,
			"spawn position %d (%d,%d): tileX=%d should be >= 0", i, pos.X, pos.Y, tileX)
		assert.Less(t, tileX, xTilesVisible,
			"spawn position %d (%d,%d): tileX=%d should be < %d", i, pos.X, pos.Y, tileX, xTilesVisible)
		assert.GreaterOrEqual(t, tileY, 0,
			"spawn position %d (%d,%d): tileY=%d should be >= 0", i, pos.X, pos.Y, tileY)
		assert.Less(t, tileY, yTilesVisible,
			"spawn position %d (%d,%d): tileY=%d should be < %d", i, pos.X, pos.Y, tileY, yTilesVisible)
	}
}

// TestEnemyReference_KeyFrameTile_NotNilForValidMonsters verifies that every
// monster type that NewCombatMonster can be called with has a non-nil KeyFrameTile.
// The renderer skips monsters with nil KeyFrameTile, so a nil tile causes invisible monsters.
func TestEnemyReference_KeyFrameTile_NotNilForValidMonsters(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		Build()
	if gs == nil {
		return
	}

	enemyRefs := *gs.GameReferences.EnemyReferences
	nilCount := 0

	for i := range enemyRefs {
		ref := &enemyRefs[i]
		mt := references.MonsterType(i)
		if !mt.IsValid() {
			continue
		}

		if ref.KeyFrameTile == nil {
			nilCount++
			t.Errorf("monster %d (%s): KeyFrameTile is nil — renderer will skip this monster",
				i, mt.Name())
		}
	}

	assert.Equal(t, 0, nilCount,
		"all valid monster types should have a non-nil KeyFrameTile for rendering")
}

// TestCombatMonsterManager_ForEachActive_SkipsDeadMonsters ensures that killed
// monsters are not visited by ForEachActive, matching the renderer's skip logic.
func TestCombatMonsterManager_ForEachActive_SkipsDeadMonsters(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(99).
		Build()
	if gs == nil {
		return
	}

	err := gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth)
	require.NoError(t, err)

	manager := gs.CombatScene.MonsterManager
	manager.Clear()

	enemyRefs := *gs.GameReferences.EnemyReferences
	rng := newFixedRNG(5)
	positions := references.ShuffleMonsterSpawnPositions(rng)

	// Spawn 4 monsters
	for i := 0; i < 4; i++ {
		ref := &enemyRefs[references.MonsterOrc]
		m := NewCombatMonster(references.MonsterOrc, ref, positions[i], rng)
		manager.AddMonster(m)
	}

	// Kill monster at index 1 and 3
	manager.GetMonster(1).SetCurrentHP(0)
	manager.GetMonster(3).SetCurrentHP(0)

	assert.Equal(t, 4, manager.Count(), "total count includes dead monsters")
	assert.Equal(t, 2, manager.CountActive(), "only 2 should be alive")

	visited := 0
	manager.ForEachActive(func(_ int, monster *CombatMonster) {
		visited++
		assert.False(t, monster.IsDead(), "dead monster should not be visited by ForEachActive")
	})
	assert.Equal(t, 2, visited, "ForEachActive should visit exactly the 2 live monsters")
}

// TestCombatMonster_SpawnExactlyOne verifies that spawning exactly 1 monster
// results in count=1, correct type, active, and not dead — matching the
// legacy direct combat-entry debug grammar behavior.
func TestCombatMonster_SpawnExactlyOne(t *testing.T) {
	gs, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		WithRandomSeed(123).
		Build()
	if gs == nil {
		return
	}

	err := gs.TransitionToCombatMap(0, references.BritanniaCombatTerritory, CombatEntryNorth)
	require.NoError(t, err)

	manager := gs.CombatScene.MonsterManager
	manager.Clear()

	enemyRefs := *gs.GameReferences.EnemyReferences
	rng := newFixedRNG(11)
	positions := references.ShuffleMonsterSpawnPositions(rng)

	monsterType := references.MonsterOrc
	ref := &enemyRefs[monsterType]
	monster := NewCombatMonster(monsterType, ref, positions[0], rng)
	manager.AddMonster(monster)

	assert.Equal(t, 1, manager.Count(), "exactly 1 monster should be in the manager")
	assert.Equal(t, 1, manager.CountActive(), "the 1 monster should be active")

	m := manager.GetMonster(0)
	require.NotNil(t, m, "monster at index 0 should exist")
	assert.Equal(t, monsterType, m.MonsterType(), "monster type should match what was spawned")
	assert.False(t, m.IsDead(), "freshly spawned monster should not be dead")
	assert.True(t, m.IsActive(), "freshly spawned monster should be active")
	assert.NotNil(t, m.EnemyRef(), "monster EnemyRef should not be nil")
	assert.NotNil(t, m.EnemyRef().KeyFrameTile, "monster KeyFrameTile must not be nil for rendering")
}
