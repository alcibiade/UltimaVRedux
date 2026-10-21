package references

// EnemyAbility is a special attack or defence an enemy may have; values match the original ability numbering.
type EnemyAbility int

// Enemy abilities, numbered from 0 as in the original game.
//
//goland:noinspection GoUnusedConst
const (
	Bludgeons         EnemyAbility = 0
	PossessCharm      EnemyAbility = 1
	Undead            EnemyAbility = 2
	DivideOnHit       EnemyAbility = 3
	Immortal          EnemyAbility = 4
	PoisonAtRange     EnemyAbility = 5
	StealsFood        EnemyAbility = 6
	NoCorpse          EnemyAbility = 7
	RangedMagic       EnemyAbility = 8
	Teleport          EnemyAbility = 9
	DisappearsOnDeath EnemyAbility = 10
	Invisibility      EnemyAbility = 11
	GatesInDaemon     EnemyAbility = 12
	Poison            EnemyAbility = 13
	InfectWithPlague  EnemyAbility = 14
)

// EnemyReference is one monster definition: its original combat statistics plus Redux behaviour flags.
type EnemyReference struct {
	MonsterType    MonsterType `json:"monsterType"    yaml:"monsterType"`
	KeyFrameTile   *Tile       `json:"keyFrameTile"   yaml:"keyFrameTile"`
	Armour         int         `json:"armour"         yaml:"armour"`
	Damage         int         `json:"damage"         yaml:"damage"`
	Dexterity      int         `json:"dexterity"      yaml:"dexterity"`
	HitPoints      int         `json:"hitPoints"      yaml:"hitPoints"`
	Intelligence   int         `json:"intelligence"   yaml:"intelligence"`
	MaxPerMap      int         `json:"maxPerMap"      yaml:"maxPerMap"`
	Strength       int         `json:"strength"       yaml:"strength"`
	TreasureNumber int         `json:"treasureNumber" yaml:"treasureNumber"`

	EnemyAbilities map[EnemyAbility]bool `json:"enemyAbilities" yaml:"enemyAbilities"`

	// Flattened behavior flags (previously in AdditionalEnemyFlags)
	Name                 string      `json:"name"                 yaml:"name"`
	NamePlural           string      `json:"namePlural"           yaml:"namePlural"`
	ActivelyAttacks      bool        `json:"activelyAttacks"      yaml:"activelyAttacks"`
	CanFlyOverWater      bool        `json:"canFlyOverWater"      yaml:"canFlyOverWater"`
	CanPassThroughWalls  bool        `json:"canPassThroughWalls"  yaml:"canPassThroughWalls"`
	DoNotMove            bool        `json:"doNotMove"            yaml:"doNotMove"`
	HasWhirlpoolMovement bool        `json:"hasWhirlpoolMovement" yaml:"hasWhirlpoolMovement"`
	IsSandEnemy          bool        `json:"isSandEnemy"          yaml:"isSandEnemy"`
	IsWaterEnemy         bool        `json:"isWaterEnemy"         yaml:"isWaterEnemy"`
	LargeMapMissile      MissileType `json:"largeMapMissile"      yaml:"largeMapMissile"`
	LargeMapMissileRange int         `json:"largeMapMissileRange" yaml:"largeMapMissileRange"`

	AttackRange int             `json:"attackRange" yaml:"attackRange"`
	Friend      *EnemyReference `json:"friend"      yaml:"friend"`
}

// CanSpawnToTile reports whether this enemy may be generated on the tile.
func (e *EnemyReference) CanSpawnToTile(tile *Tile) bool {
	if !e.isMonsterSpawnableOnTile(tile) {
		return false
	}

	var bCanSpawnOnTile bool

	switch {
	case e.IsSandEnemy:
		bCanSpawnOnTile = tile.isDesert()
	case e.IsWaterEnemy:
		bCanSpawnOnTile = tile.IsWaterEnemyPassable()
	default:
		bCanSpawnOnTile = tile.IsLandEnemyPassable()
	}

	return bCanSpawnOnTile
}

// CanMoveToTile reports whether this enemy may walk or swim onto the tile.
func (e *EnemyReference) CanMoveToTile(tile *Tile) bool {
	if tile == nil {
		return false
	}
	// Pirate ships reuse Stocks' monster-table slot but have a vehicle shape.
	if e.KeyFrameTile != nil && IsPirateShipSprite(e.KeyFrameTile.Index) {
		return tile.IsBoatPassable()
	}

	return OriginalCreatureTerrainAllowsShape(OriginalMonsterShape(e.MonsterType), tile)
}

func (e *EnemyReference) isMonsterSpawnableOnTile(tile *Tile) bool {
	return tile.IsBoatPassable() || tile.IsCarpetPassable() || tile.IsHorsePassable() ||
		tile.IsWalkingPassable() || tile.IsWaterEnemyPassable() || tile.IsLandEnemyPassable()
}

// HasAbility reports whether the enemy has the given special ability.
func (e *EnemyReference) HasAbility(ability EnemyAbility) bool {
	return e.EnemyAbilities[ability]
}
