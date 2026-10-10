package debugging

import (
	"encoding/json"
	"strconv"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	// CommandWorldPositionSet atomically applies an exact position or authoritative route target.
	CommandWorldPositionSet = "world.position.set"
	// CommandDungeonExplorerStatus reports the active semantic dungeon floor.
	CommandDungeonExplorerStatus = "dungeon.explorer.status"
	// CommandDungeonExplorerSetup installs one exact production dungeon setup.
	CommandDungeonExplorerSetup = "dungeon.explorer.setup"

	categoryDungeonExplorer     = "Dungeon Explorer"
	parameterWorldPlacement     = "placement"
	parameterWorldRoute         = "route"
	parameterWorldLocation      = "location"
	parameterWorldFloor         = "floor"
	parameterDungeon            = "dungeon"
	parameterDungeonFacing      = "facing"
	parameterExpectedLocationID = "expectedLocationId"
	parameterExpectedFloor      = "expectedFloor"
	parameterExpectedSource     = "expectedSourceFingerprint"
	dungeonName                 = "Dungeon"
	floorName                   = "Floor"
	dungeonFacingNorth          = "north"
	dungeonFacingEast           = "east"
	dungeonFacingSouth          = "south"
	dungeonFacingWest           = "west"
	dungeonFacingNorthLabel     = "North"
	dungeonFacingEastLabel      = "East"
	dungeonFacingSouthLabel     = "South"
	dungeonFacingWestLabel      = "West"
	worldRouteBritanniaLabel    = "Britannia"
	worldRouteUnderworldLabel   = "Underworld"

	// WorldPlacementExact preserves arbitrary location, floor, and coordinate setup.
	WorldPlacementExact = "exact"
	// WorldPlacementDungeonOutside resolves a dungeon entrance on a selected world plane.
	WorldPlacementDungeonOutside = "dungeon_outside_entrance"
	// WorldPlacementDungeonInsideStart resolves the selected route's fresh dungeon entry.
	WorldPlacementDungeonInsideStart = "dungeon_inside_start"
	// WorldPlacementDungeonUnderworldExit resolves the floor-seven down-ladder endpoint.
	WorldPlacementDungeonUnderworldExit = "dungeon_underworld_exit"
	// WorldRouteBritannia selects the surface route.
	WorldRouteBritannia = "britannia"
	// WorldRouteUnderworld selects the Underworld route.
	WorldRouteUnderworld = "underworld"
)

//nolint:funlen,exhaustruct // Complete schemas are kept together for direct Go/TypeScript review.
func worldDungeonExplorerCommands() []CommandDefinition {
	minimumWorldLocation, maximumWorldLocation := 0, int(references.Doom)
	minimumWorldFloor, maximumWorldFloor := -1, references.DungeonFloorCount-1
	minimumWorldCoordinate, maximumWorldCoordinate := 0, int(references.XLargeMapTiles)-1
	minimumDungeonFloor, maximumDungeonFloor := 0, references.DungeonFloorCount-1
	minimumDungeonCoordinate, maximumDungeonCoordinate := 0, int(references.XDungeonMapTiles)-1
	minimumRevision := 0
	reviewedWorldParameters := []ParameterDefinition{
		{ID: parameterExpectedLocationID, Name: "Reviewed Location", Kind: ParameterInteger, Required: true, Minimum: &minimumWorldLocation, Maximum: &maximumWorldLocation, Availability: []string{contextGameplay, contextOutsideCombat}, ScenarioRuntime: true},
		{ID: parameterExpectedFloor, Name: "Reviewed Floor", Kind: ParameterInteger, Required: true, Minimum: &minimumWorldFloor, Maximum: &maximumWorldFloor, Availability: []string{contextGameplay, contextOutsideCombat}, ScenarioRuntime: true},
		{ID: "expectedX", Name: "Reviewed X", Kind: ParameterInteger, Required: true, Minimum: &minimumWorldCoordinate, Maximum: &maximumWorldCoordinate, Availability: []string{contextGameplay, contextOutsideCombat}, ScenarioRuntime: true},
		{ID: "expectedY", Name: "Reviewed Y", Kind: ParameterInteger, Required: true, Minimum: &minimumWorldCoordinate, Maximum: &maximumWorldCoordinate, Availability: []string{contextGameplay, contextOutsideCombat}, ScenarioRuntime: true},
		{ID: parameterExpectedSource, Name: "Reviewed Source Fingerprint", Kind: ParameterString, Required: true, Availability: []string{contextGameplay, contextOutsideCombat}, ScenarioRuntime: true},
	}

	worldParameters := make([]ParameterDefinition, 0, 6+len(reviewedWorldParameters))
	worldParameters = append(worldParameters, []ParameterDefinition{
		{ID: parameterWorldPlacement, Name: "Placement", Description: "Choose exact coordinates or one authoritative dungeon route target.", Kind: ParameterEnum, Required: false, Default: json.RawMessage(`"exact"`), Choices: worldPlacementChoices(), ControlHint: controlHintSelect, Availability: []string{contextGameplay, contextOutsideCombat}},
		{ID: parameterWorldRoute, Name: "Dungeon Route", Description: "Select the Britannia or Underworld dungeon route when the placement uses one.", Kind: ParameterEnum, Required: false, Default: json.RawMessage(`"britannia"`), Choices: worldRouteChoices(), ControlHint: controlHintSelect, Availability: []string{contextGameplay, contextOutsideCombat}},
		{ID: parameterWorldLocation, Name: "Destination", Description: "Choose Britannia/Underworld, a small location, or a dungeon.", Kind: ParameterEnum, Required: true, Choices: worldLocationChoices(), ControlHint: controlHintSelect, Availability: []string{contextGameplay, contextOutsideCombat}},
		{ID: parameterWorldFloor, Name: floorName, Description: "Choose a floor valid for an exact destination.", Kind: ParameterEnum, Required: false, Default: json.RawMessage(`"0"`), Choices: worldFloorChoices(), ControlHint: controlHintSelect, Availability: []string{contextGameplay, contextOutsideCombat}},
		{ID: "x", Name: "X", Kind: ParameterInteger, Required: false, Default: json.RawMessage(`0`), Minimum: &minimumWorldCoordinate, Maximum: &maximumWorldCoordinate, ControlHint: controlHintNumber, Availability: []string{contextGameplay, contextOutsideCombat}},
		{ID: "y", Name: "Y", Kind: ParameterInteger, Required: false, Default: json.RawMessage(`0`), Minimum: &minimumWorldCoordinate, Maximum: &maximumWorldCoordinate, ControlHint: controlHintNumber, Availability: []string{contextGameplay, contextOutsideCombat}},
	}...)
	worldParameters = append(worldParameters, reviewedWorldParameters...)

	dungeonParameters := make([]ParameterDefinition, 0, 7+len(reviewedWorldParameters))
	dungeonParameters = append(dungeonParameters, []ParameterDefinition{
		{ID: parameterDungeon, Name: dungeonName, Kind: ParameterEnum, Required: true, Choices: dungeonLocationChoices(), ControlHint: controlHintSelect, Availability: []string{contextGameplay, contextOutsideCombat}},
		{ID: parameterWorldFloor, Name: floorName, Kind: ParameterInteger, Required: false, Default: json.RawMessage(`0`), Minimum: &minimumDungeonFloor, Maximum: &maximumDungeonFloor, ControlHint: controlHintNumber, Availability: []string{contextGameplay, contextOutsideCombat}},
		{ID: "x", Name: "X", Kind: ParameterInteger, Required: false, Default: json.RawMessage(`4`), Minimum: &minimumDungeonCoordinate, Maximum: &maximumDungeonCoordinate, ControlHint: controlHintNumber, Availability: []string{contextGameplay, contextOutsideCombat}},
		{ID: "y", Name: "Y", Kind: ParameterInteger, Required: false, Default: json.RawMessage(`4`), Minimum: &minimumDungeonCoordinate, Maximum: &maximumDungeonCoordinate, ControlHint: controlHintNumber, Availability: []string{contextGameplay, contextOutsideCombat}},
		{ID: parameterDungeonFacing, Name: "Facing", Kind: ParameterEnum, Required: false, Default: json.RawMessage(`"east"`), Choices: dungeonFacingChoices(), ControlHint: controlHintSelect, Availability: []string{contextGameplay, contextOutsideCombat}},
		{ID: "lit", Name: "Lit", Kind: ParameterBoolean, Required: false, Default: json.RawMessage(`true`), ControlHint: controlHintSelect, Availability: []string{contextGameplay, contextOutsideCombat}},
	}...)
	dungeonParameters = append(dungeonParameters, reviewedWorldParameters...)
	dungeonParameters = append(dungeonParameters, ParameterDefinition{
		ID: "expectedDungeonRevision", Name: "Reviewed Dungeon Revision", Kind: ParameterInteger, Required: true,
		Minimum: &minimumRevision, Availability: []string{contextGameplay, contextOutsideCombat}, ScenarioRuntime: true,
	})

	return []CommandDefinition{
		{
			ID: CommandWorldPositionSet, Category: "World", Name: "Set World Position",
			Description: "Atomically apply one reviewed exact position or authoritative dungeon route target.",
			Aliases:     []string{"world navigator set", "teleport location floor"}, Parameters: worldParameters,
			ExecutionModes: []ExecutionMode{ExecutionSynchronized},
		},
		{
			ID: CommandDungeonExplorerStatus, Category: categoryDungeonExplorer, Name: categoryDungeonExplorer,
			Description: "Inspect the active semantic dungeon floor, facing, lighting, and packed cells.",
			Aliases:     []string{"dungeon status", "dungeon map"}, ExecutionModes: []ExecutionMode{ExecutionQuery},
		},
		{
			ID: CommandDungeonExplorerSetup, Category: categoryDungeonExplorer, Name: "Set Up " + dungeonName,
			Description: "Atomically install one reviewed dungeon, floor, coordinate, facing, and light setup.",
			Aliases:     []string{"dungeon setup", "enter dungeon"}, Parameters: dungeonParameters,
			ExecutionModes: []ExecutionMode{ExecutionSynchronized},
		},
	}
}

func worldPlacementChoices() []Choice {
	return []Choice{
		{Value: WorldPlacementExact, Label: "Arbitrary exact position"},
		{Value: WorldPlacementDungeonOutside, Label: "Outside dungeon entrance"},
		{Value: WorldPlacementDungeonInsideStart, Label: "Inside dungeon start"},
		{Value: WorldPlacementDungeonUnderworldExit, Label: "Dungeon Underworld-exit ladder"},
	}
}

func worldRouteChoices() []Choice {
	return []Choice{
		{Value: WorldRouteBritannia, Label: worldRouteBritanniaLabel},
		{Value: WorldRouteUnderworld, Label: worldRouteUnderworldLabel},
	}
}

func worldLocationChoices() []Choice {
	choices := make([]Choice, 0, int(references.Doom)+1)
	for location := references.BritanniaUnderworld; location <= references.Doom; location++ {
		label := location.String()
		if location == references.BritanniaUnderworld {
			label = "Britannia / Underworld"
		}
		choices = append(choices, Choice{Value: strconv.Itoa(int(location)), Label: label})
	}

	return choices
}

func dungeonLocationChoices() []Choice {
	choices := make([]Choice, 0, int(references.Doom-references.Deceit)+1)
	for location := references.Deceit; location <= references.Doom; location++ {
		choices = append(choices, Choice{Value: strconv.Itoa(int(location)), Label: location.String()})
	}

	return choices
}

func worldFloorChoices() []Choice {
	choices := make([]Choice, 0, 1+references.DungeonFloorCount)
	choices = append(choices, Choice{Value: "-1", Label: "Underworld / Basement (-1)"})
	for floor := range references.DungeonFloorCount {
		choices = append(choices, Choice{
			Value: strconv.Itoa(floor),
			Label: "Floor " + strconv.Itoa(floor) + " / Dungeon Level " + strconv.Itoa(floor+1),
		})
	}

	return choices
}

func dungeonFacingChoices() []Choice {
	return []Choice{
		{Value: dungeonFacingNorth, Label: dungeonFacingNorthLabel},
		{Value: dungeonFacingEast, Label: dungeonFacingEastLabel},
		{Value: dungeonFacingSouth, Label: dungeonFacingSouthLabel},
		{Value: dungeonFacingWest, Label: dungeonFacingWestLabel},
	}
}
