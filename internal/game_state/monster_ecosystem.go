package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const maximumMonsterGenerationBaseOdds = 1000

// ErrInvalidMonsterGenerationBaseOdds identifies an out-of-range generation setting.
var ErrInvalidMonsterGenerationBaseOdds = errors.New("invalid monster generation base odds")

// MonsterEcosystemEnemySnapshot is a detached view of a current-map enemy.
type MonsterEcosystemEnemySnapshot struct {
	RuntimeToken         int
	MonsterType          references.MonsterType
	Name                 string
	Position             references.Position
	Floor                references.FloorNumber
	Visible              bool
	ActivelyAttacks      bool
	CanFlyOverWater      bool
	CanPassThroughWalls  bool
	DoNotMove            bool
	HasWhirlpoolMovement bool
	SkipNextTurn         bool
	IsSandEnemy          bool
	IsWaterEnemy         bool
}

// MonsterEcosystemSnapshot is a detached view of the current map's unit ecosystem.
type MonsterEcosystemSnapshot struct {
	Location           references.Location
	MapType            references.GeneralMapType
	Floor              references.FloorNumber
	GenerationEnabled  bool
	GenerationBaseOdds int
	Capacity           int
	CurrentUnitCount   int
	EnemyCount         int
	FriendlyCount      int
	VehicleCount       int
	Enemies            []MonsterEcosystemEnemySnapshot
}

// SetMonsterGenerationEnabled explicitly enables or disables natural monster generation.
func (g *GameState) SetMonsterGenerationEnabled(enabled bool) {
	g.DebugOptions.MonsterGen = enabled
}

// SetMonsterGenerationBaseOdds sets the maximum inclusive natural-generation roll.
func (g *GameState) SetMonsterGenerationBaseOdds(oneInX int) error {
	if oneInX < 1 || oneInX > maximumMonsterGenerationBaseOdds {
		return fmt.Errorf(
			"%w: must be between 1 and %d: %d",
			ErrInvalidMonsterGenerationBaseOdds,
			maximumMonsterGenerationBaseOdds,
			oneInX,
		)
	}

	g.TheOdds.SetMonsterGenerationRollMaximum(oneInX)

	return nil
}

// CurrentMonsterEcosystemSnapshot returns a detached snapshot of current-map units.
func (g *GameState) CurrentMonsterEcosystemSnapshot() MonsterEcosystemSnapshot {
	location := g.MapState.PlayerLocation
	snapshot := MonsterEcosystemSnapshot{
		Location:           location.Location,
		MapType:            location.Location.GetMapType(),
		Floor:              location.Floor,
		GenerationEnabled:  g.DebugOptions.MonsterGen,
		GenerationBaseOdds: g.TheOdds.GetMonsterGenerationRollMaximum(),
		Capacity:           map_units.MaximumNpcsPerMap,
		CurrentUnitCount:   0,
		EnemyCount:         0,
		FriendlyCount:      0,
		VehicleCount:       0,
		Enemies:            make([]MonsterEcosystemEnemySnapshot, 0),
	}
	if g.CurrentNPCAIController == nil || g.CurrentNPCAIController.GetNpcs() == nil {
		return snapshot
	}

	units := *g.CurrentNPCAIController.GetNpcs()
	snapshot.CurrentUnitCount = len(units)
	for _, unit := range units {
		switch typedUnit := unit.(type) {
		case *map_units.NPCEnemy:
			ref := typedUnit.EnemyReference
			snapshot.EnemyCount++
			snapshot.Enemies = append(snapshot.Enemies, MonsterEcosystemEnemySnapshot{
				RuntimeToken:         typedUnit.MapUnitDetails().NPCNum,
				MonsterType:          ref.MonsterType,
				Name:                 ref.Name,
				Position:             typedUnit.Pos(),
				Floor:                typedUnit.Floor(),
				Visible:              typedUnit.IsVisible(),
				ActivelyAttacks:      ref.ActivelyAttacks,
				CanFlyOverWater:      ref.CanFlyOverWater,
				CanPassThroughWalls:  ref.CanPassThroughWalls,
				DoNotMove:            ref.DoNotMove,
				HasWhirlpoolMovement: ref.HasWhirlpoolMovement,
				SkipNextTurn:         typedUnit.MapUnitDetails().SkipNextTurn,
				IsSandEnemy:          ref.IsSandEnemy,
				IsWaterEnemy:         ref.IsWaterEnemy,
			})
		case *map_units.NPCFriendly:
			if typedUnit.NPCReference.GetNPCType() == references.Vehicle {
				snapshot.VehicleCount++
			} else {
				snapshot.FriendlyCount++
			}
		}
	}

	return snapshot
}

// ClearCurrentMapEnemies clears enemies when the active controller supports it.
// The second result is false when the controller cannot authoritatively perform the operation.
func (g *GameState) ClearCurrentMapEnemies() (int, bool) {
	if g.IsInCombat() {
		return 0, false
	}
	controller, ok := g.CurrentNPCAIController.(interface{ ClearEnemies() int })
	if !ok {
		return 0, false
	}

	return controller.ClearEnemies(), true
}
