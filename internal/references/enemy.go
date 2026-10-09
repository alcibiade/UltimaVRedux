package references

type EnemyAbility int

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

type EnemyReference struct {
	MonsterType    MonsterType `json:"monster_type"    yaml:"monster_type"`
	KeyFrameTile   *Tile       `json:"key_frame_tile"  yaml:"key_frame_tile"`
	Armour         int         `json:"armour"          yaml:"armour"`
	Damage         int         `json:"damage"          yaml:"damage"`
	Dexterity      int         `json:"dexterity"       yaml:"dexterity"`
	HitPoints      int         `json:"hit_points"      yaml:"hit_points"`
	Intelligence   int         `json:"intelligence"    yaml:"intelligence"`
	MaxPerMap      int         `json:"max_per_map"     yaml:"max_per_map"`
	Strength       int         `json:"strength"        yaml:"strength"`
	TreasureNumber int         `json:"treasure_number" yaml:"treasure_number"`

	EnemyAbilities map[EnemyAbility]bool `json:"enemy_abilities" yaml:"enemy_abilities"`

	// Flattened behavior flags (previously in AdditionalEnemyFlags)
	Name                 string      `json:"name"                    yaml:"name"`
	NamePlural           string      `json:"name_plural"             yaml:"name_plural"`
	ActivelyAttacks      bool        `json:"actively_attacks"        yaml:"actively_attacks"`
	CanFlyOverWater      bool        `json:"can_fly_over_water"      yaml:"can_fly_over_water"`
	CanPassThroughWalls  bool        `json:"can_pass_through_walls"  yaml:"can_pass_through_walls"`
	DoNotMove            bool        `json:"do_not_move"             yaml:"do_not_move"`
	HasWhirlpoolMovement bool        `json:"has_whirlpool_movement"  yaml:"has_whirlpool_movement"`
	IsSandEnemy          bool        `json:"is_sand_enemy"           yaml:"is_sand_enemy"`
	IsWaterEnemy         bool        `json:"is_water_enemy"          yaml:"is_water_enemy"`
	LargeMapMissile      MissileType `json:"large_map_missile"       yaml:"large_map_missile"`
	LargeMapMissileRange int         `json:"large_map_missile_range" yaml:"large_map_missile_range"`

	AttackRange int             `json:"attack_range" yaml:"attack_range"`
	Friend      *EnemyReference `json:"friend"       yaml:"friend"`
}

func (e *EnemyReference) CanSpawnToTile(tile *Tile) bool {
	if !e.isMonsterSpawnableOnTile(tile) {
		return false
	}

	var bCanSpawnOnTile bool

	if e.IsSandEnemy {
		bCanSpawnOnTile = tile.IsDesert()
	} else if e.IsWaterEnemy {
		bCanSpawnOnTile = tile.IsWaterEnemyPassable()
	} else {
		bCanSpawnOnTile = tile.IsLandEnemyPassable()
	}

	return bCanSpawnOnTile
}

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

func (e *EnemyReference) HasAbility(ability EnemyAbility) bool {
	return e.EnemyAbilities[ability]
}

func EnemyAbilityToString(ability EnemyAbility) string {
	switch ability {
	case Bludgeons:
		return "Bludgeons"
	case PossessCharm:
		return "PossessCharm"
	case Undead:
		return "Undead"
	case DivideOnHit:
		return "DivideOnHit"
	case Immortal:
		return "Immortal"
	case PoisonAtRange:
		return "PoisonAtRange"
	case StealsFood:
		return "StealsFood"
	case NoCorpse:
		return "NoCorpse"
	case RangedMagic:
		return "RangedMagic"
	case Teleport:
		return "Teleport"
	case DisappearsOnDeath:
		return "DisappearsOnDeath"
	case Invisibility:
		return "Invisibility"
	case GatesInDaemon:
		return "GatesInDaemon"
	case Poison:
		return "Poison"
	case InfectWithPlague:
		return "InfectWithPlague"
	default:
		return "Unknown"
	}
}
