package references

// Dungeon and camping monster pools and spawn positions.
// Camping and first-person dungeon pools are distinct; ref:CAMP-0002 and ref:DNG-0010.

// CampingAmbushMonsters are monsters that attack when the party is camping in the wilderness.
// Wilderness ambush candidates are separate from the dungeon wanderer pool; ref:CAMP-0002.
//
//nolint:gochecknoglobals // Immutable wilderness ambush pool; ref:CAMP-0002.
var CampingAmbushMonsters = [8]MonsterType{
	MonsterTroll,
	MonsterRat,
	MonsterBat,
	MonsterSlime,
	MonsterSpider,
	MonsterGremlin,
	MonsterHeadless,
	MonsterRat,
}

// DungeonWanderingMonsterCount is the number of dungeon creature/animation pairs; ref:DNG-0010.
const DungeonWanderingMonsterCount = 8

// DungeonWanderingMonsterDefinition couples one MON archive/shape index to the
// monster identity and initial animation metadata at that same source index.
type DungeonWanderingMonsterDefinition struct {
	Monster   MonsterType
	Animation byte
}

// Dungeon creature/animation pairs (ref:DNG-0010). Keep this private so gameplay cannot
// mutate source data and make selection, rendering, and validation disagree.
//
//nolint:gochecknoglobals // Immutable dungeon creature/animation pairs; ref:DNG-0010.
var dungeonWanderingMonsterDefinitions = [DungeonWanderingMonsterCount]DungeonWanderingMonsterDefinition{
	{Monster: MonsterRat, Animation: 0x60},
	{Monster: MonsterBat, Animation: 0xa0},
	{Monster: MonsterSpider, Animation: 0x00},
	{Monster: MonsterGhost, Animation: 0x90},
	{Monster: MonsterSlime, Animation: 0x80},
	{Monster: MonsterGremlin, Animation: 0x60},
	{Monster: MonsterGazer, Animation: 0x00},
	{Monster: MonsterReaper, Animation: 0x00},
}

// DungeonWanderingMonsterDefinitionAt returns one detached source definition.
func DungeonWanderingMonsterDefinitionAt(index int) (DungeonWanderingMonsterDefinition, bool) {
	if index < 0 || index >= len(dungeonWanderingMonsterDefinitions) {
		return DungeonWanderingMonsterDefinition{}, false
	}

	return dungeonWanderingMonsterDefinitions[index], true
}

// DungeonWanderingMonsterDefinitionFor resolves immutable archive and animation
// metadata from the durable monster identity.
func DungeonWanderingMonsterDefinitionFor(monster MonsterType) (DungeonWanderingMonsterDefinition, int, bool) {
	for index, definition := range dungeonWanderingMonsterDefinitions {
		if definition.Monster == monster {
			return definition, index, true
		}
	}

	return DungeonWanderingMonsterDefinition{}, 0, false
}

// IsDungeonWanderingMonster reports membership in the first-person wanderer pool; ref:DNG-0010.
func IsDungeonWanderingMonster(monster MonsterType) bool {
	_, _, ok := DungeonWanderingMonsterDefinitionFor(monster)

	return ok
}

// MonsterSpawnPositions are 16 predefined positions for placing monsters on the 11x11 combat map.
// Original encounter-placement coordinates; ref:COMBAT-0001.
// Encounter generation uses these positions independently of creature identities.
//
//nolint:gochecknoglobals, mnd // Immutable encounter-placement coordinates; ref:COMBAT-0001.
var MonsterSpawnPositions = [16]Position{
	{X: 2, Y: 0},
	{X: 5, Y: 0},
	{X: 8, Y: 0},
	{X: 4, Y: 1},
	{X: 6, Y: 1},
	{X: 1, Y: 2},
	{X: 5, Y: 2},
	{X: 9, Y: 2},
	{X: 3, Y: 3},
	{X: 7, Y: 3},
	{X: 0, Y: 4},
	{X: 4, Y: 4},
	{X: 6, Y: 4},
	{X: 10, Y: 4},
	{X: 2, Y: 5},
	{X: 8, Y: 5},
}

// GetCampingAmbusher returns a random camping ambusher monster.
// Used when generating camping encounter monsters.
func GetCampingAmbusher(rng RandomProvider) MonsterType {
	return CampingAmbushMonsters[rng.Intn(len(CampingAmbushMonsters))]
}

// GetDungeonWanderingMonster returns a random dungeon wandering monster.
// Used when generating random dungeon encounters.
func GetDungeonWanderingMonster(rng RandomProvider) MonsterType {
	definition, ok := DungeonWanderingMonsterDefinitionAt(rng.Intn(DungeonWanderingMonsterCount))
	if !ok {
		panic("dungeon wandering monster selection is out of range")
	}

	return definition.Monster
}

// GetMonsterSpawnPosition returns the spawn position for the nth monster.
// Index should be 0-15 for valid positions.
// For indices >= 16, wraps around using modulo.
func GetMonsterSpawnPosition(index int) Position {
	return MonsterSpawnPositions[index%len(MonsterSpawnPositions)]
}

// GetAllMonsterSpawnPositions returns a copy of all spawn positions.
// Useful for shuffling positions for random monster placement.
func GetAllMonsterSpawnPositions() []Position {
	positions := make([]Position, len(MonsterSpawnPositions))
	copy(positions, MonsterSpawnPositions[:])

	return positions
}

// ShuffleMonsterSpawnPositions returns shuffled spawn positions.
// Randomizes the placement order without changing the coordinate set.
func ShuffleMonsterSpawnPositions(rng RandomProvider) []Position {
	positions := GetAllMonsterSpawnPositions()

	// Fisher-Yates shuffle
	for i := len(positions) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		positions[i], positions[j] = positions[j], positions[i]
	}

	return positions
}

// Monster count bounds and companion threshold; ref:ENEMY-0003.
const (
	// MaxMonstersInCombat is the maximum number of monsters in a combat encounter.
	// Redux reserves this many monster positions in its encounter composition.
	MaxMonstersInCombat = 16

	// SoloMonsterCount is the count for solo monsters (dragons, mimics, etc.)
	SoloMonsterCount = 1

	// SwarmMonsterCount is the count for swarm monsters (bats, slimes, mongbats)
	SwarmMonsterCount = 16

	// GuardGroupCount is the count for guard groups
	GuardGroupCount = 8

	// FriendDivisor is used in friend threshold calculation: count/FriendDivisor + 1
	FriendDivisor = 4
)
