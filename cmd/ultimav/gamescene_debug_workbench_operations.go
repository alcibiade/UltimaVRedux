package main

import (
	"github.com/bradhannah/UltimaVRedux/internal/ai"
	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/game_state"
	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

type guardStatusResult struct {
	Supported                  bool   `json:"supported"`
	UnavailableReason          string `json:"unavailableReason,omitempty"`
	LocationID                 int    `json:"locationId"`
	LocationName               string `json:"locationName"`
	Floor                      int    `json:"floor"`
	Criminal                   bool   `json:"criminal"`
	FriendlyNPCCount           int    `json:"friendlyNpcCount"`
	PotentialGuardCount        int    `json:"potentialGuardCount"`
	HostilePotentialGuardCount int    `json:"hostilePotentialGuardCount"`
	OtherHostileNPCCount       int    `json:"otherHostileNpcCount"`
}

type shadowlordStatusEntry struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	AntiVirtue string `json:"antiVirtue"`
	Alive      bool   `json:"alive"`
}

type shadowlordAssignmentResult struct {
	LocationID     int    `json:"locationId"`
	LocationName   string `json:"locationName"`
	ShadowlordID   int    `json:"shadowlordId"`
	ShadowlordName string `json:"shadowlordName"`
}

type shadowlordStatusResult struct {
	Day                int                          `json:"day"`
	AliveCount         int                          `json:"aliveCount"`
	AllDefeated        bool                         `json:"allDefeated"`
	Summoned           bool                         `json:"summoned"`
	SummonedShadowlord *shadowlordStatusEntry       `json:"summonedShadowlord,omitempty"`
	AutomaticPresence  *shadowlordStatusEntry       `json:"automaticPresence,omitempty"`
	Shadowlords        []shadowlordStatusEntry      `json:"shadowlords"`
	TownAssignments    []shadowlordAssignmentResult `json:"townAssignments"`
}

type deathResurrectionResult struct {
	LocationID   int    `json:"locationId"`
	LocationName string `json:"locationName"`
	X            int    `json:"x"`
	Y            int    `json:"y"`
	Floor        int    `json:"floor"`
}

type deathStatusResult struct {
	WipeReview                activePartyWipeReviewResult `json:"wipeReview"`
	AllPartyDead              bool                        `json:"allPartyDead"`
	ActivePartyCount          int                         `json:"activePartyCount"`
	DeadPartyCount            int                         `json:"deadPartyCount"`
	Karma                     int                         `json:"karma"`
	ExperienceRetainedPercent int                         `json:"experienceRetainedPercent"`
	ExperienceLostPercent     int                         `json:"experienceLostPercent"`
	MinimumPostDeathKarma     int                         `json:"minimumPostDeathKarma"`
	MinimumPostDeathFood      int                         `json:"minimumPostDeathFood"`
	Resurrection              deathResurrectionResult     `json:"resurrection"`
}

type debugOptionsStatusResult struct {
	FreeMove                     bool `json:"freeMove"`
	TextPlaceholdersEnabled      bool `json:"textPlaceholdersEnabled"`
	DialogBoundsEnabled          bool `json:"dialogBoundsEnabled"`
	UnlimitedCastingResources    bool `json:"unlimitedCastingResources"`
	BypassSpellLevelRestrictions bool `json:"bypassSpellLevelRestrictions"`
	LightingOverlayEnabled       bool `json:"lightingOverlayEnabled"`
}

func (g *GameScene) executeOperationalDebugCommand(request debugging.Request) (debugging.Result, bool) {
	switch request.CommandID {
	case debugging.CommandGuardsStatus:
		return debugSuccessResult(request, "Guard status loaded.", newGuardStatusResult(g.gameState)), true
	case debugging.CommandShadowlordStatus:
		return debugSuccessResult(request, "Shadowlord status loaded.", newShadowlordStatusResult(g.gameState)), true
	case debugging.CommandDeathStatus:
		return debugSuccessResult(request, "Death status loaded.", newDeathStatusResult(g.gameState)), true
	case debugging.CommandDebugOptionsStatus:
		return debugSuccessResult(request, "Debug options loaded.", newDebugOptionsStatusResult(g.gameState)), true
	case debugging.CommandMovementFreeMoveSet:
		before, after := g.gameState.SetFreeMove(mustDebugBooleanArgument(request, "enabled"))

		return debugOptionMutationResult(request, debugging.ExecutionLiveSafe, "Free Move updated.", "debug.freeMove", before, after, g.gameState), true
	case debugging.CommandTextPlaceholdersSet:
		before, after := g.gameState.SetTextPlaceholders(mustDebugBooleanArgument(request, "enabled"))

		return debugOptionMutationResult(request, debugging.ExecutionPresentation, "Text placeholders updated.", "presentation.textPlaceholders", before, after, g.gameState), true
	case debugging.CommandDialogBoundsSet:
		before, after := g.gameState.SetDialogBoundingBoxes(mustDebugBooleanArgument(request, "enabled"))

		return debugOptionMutationResult(request, debugging.ExecutionPresentation, "Dialog bounds updated.", "presentation.dialogBounds", before, after, g.gameState), true
	case debugging.CommandUnlimitedCastingResourcesSet:
		before, after := g.gameState.SetUnlimitedCastingResources(mustDebugBooleanArgument(request, "enabled"))

		return debugOptionMutationResult(request, debugging.ExecutionLiveSafe, "Unlimited casting resources updated.", "debug.unlimitedCastingResources", before, after, g.gameState), true
	case debugging.CommandSpellLevelRestrictionsBypassSet:
		before, after := g.gameState.SetBypassSpellLevelRestrictions(mustDebugBooleanArgument(request, "enabled"))

		return debugOptionMutationResult(request, debugging.ExecutionLiveSafe, "Spell level bypass updated.", "debug.bypassSpellLevelRestrictions", before, after, g.gameState), true
	default:
		return debugging.Result{}, false
	}
}

func newGuardStatusResult(state *game_state.GameState) guardStatusResult {
	location := state.MapState.PlayerLocation.Location
	result := guardStatusResult{
		LocationID: int(location), LocationName: location.String(), Floor: int(state.MapState.PlayerLocation.Floor),
		Criminal: state.IsCriminal(),
	}
	controller, ok := state.CurrentNPCAIController.(*ai.NPCAIControllerSmallMap)
	if !ok {
		result.UnavailableReason = "Town guards are available only on a small map."

		return result
	}
	result.Supported = true
	for _, unit := range *controller.GetNpcs() {
		friendly, ok := unit.(*map_units.NPCFriendly)
		if !ok {
			continue
		}
		result.FriendlyNPCCount++
		potential := friendly.NPCReference.WantsToAttackAvatarWhenBadStuffGoesDown()
		hostile := friendly.MapUnitDetails().GetOverriddenAiType() == references.ExtortOrAttackOrFollow
		if potential {
			result.PotentialGuardCount++
			if hostile {
				result.HostilePotentialGuardCount++
			}
		} else if hostile {
			result.OtherHostileNPCCount++
		}
	}

	return result
}

func newShadowlordStatusResult(state *game_state.GameState) shadowlordStatusResult {
	entries := make([]shadowlordStatusEntry, 0, references.ShadowlordCount)
	for index := range references.ShadowlordCount {
		entries = append(entries, newShadowlordStatusEntry(references.Shadowlord(index), state.ShadowlordState))
	}
	assignments := make([]shadowlordAssignmentResult, 0)
	for location := game_state.ShadowlordTownMinLocation; location <= game_state.ShadowlordTownMaxLocation; location++ {
		lord := state.GetShadowlordForLocation(references.Location(location))
		if lord == references.NoShadowlord {
			continue
		}
		assignments = append(assignments, shadowlordAssignmentResult{
			LocationID: location, LocationName: references.Location(location).String(),
			ShadowlordID: int(lord), ShadowlordName: lord.String(),
		})
	}
	result := shadowlordStatusResult{
		Day: int(state.DateTime.Day), AliveCount: state.ShadowlordState.CountAlive(),
		AllDefeated: state.ShadowlordState.AllDefeated(), Summoned: state.ShadowlordState.Summoned,
		Shadowlords: entries, TownAssignments: assignments,
	}
	if state.ShadowlordState.Summoned && state.ShadowlordState.LastCalled != references.NoShadowlord {
		entry := newShadowlordStatusEntry(state.ShadowlordState.LastCalled, state.ShadowlordState)
		result.SummonedShadowlord = &entry
	}
	if state.ShadowlordState.HasAutomaticPresence() {
		entry := newShadowlordStatusEntry(state.ShadowlordState.AutomaticPresence, state.ShadowlordState)
		result.AutomaticPresence = &entry
	}

	return result
}

func newShadowlordStatusEntry(lord references.Shadowlord, state *references.ShadowlordState) shadowlordStatusEntry {
	return shadowlordStatusEntry{ID: int(lord), Name: lord.String(), AntiVirtue: lord.GetAntiVirtue(), Alive: state.IsAlive(lord)}
}

func newDeathStatusResult(state *game_state.GameState) deathStatusResult {
	active, dead := 0, 0
	for index := range party_state.MaxActiveParty {
		character := state.PartyState.GetCharacter(index)
		if character == nil || character.PartyMap != party_state.PartyMapActive {
			continue
		}
		active++
		if character.GetStatus() == party_state.Dead {
			dead++
		}
	}
	karma := state.PartyState.Karma.Value
	experienceRetainedPercent := karma
	if karma >= 98 {
		experienceRetainedPercent = game_state.KarmaPercentageBase
	}
	location := state.GetResurrectionLocation()
	position := state.GetResurrectionPosition()

	return deathStatusResult{
		WipeReview:   newActivePartyWipeReviewResult(state.ReviewActivePartyWipe()),
		AllPartyDead: state.PartyState.IsAllPartyDead(), ActivePartyCount: active, DeadPartyCount: dead,
		Karma: karma, ExperienceRetainedPercent: experienceRetainedPercent,
		ExperienceLostPercent: game_state.KarmaPercentageBase - experienceRetainedPercent,
		MinimumPostDeathKarma: game_state.MinimumKarmaAfterDeath, MinimumPostDeathFood: game_state.MinimumFoodAfterDeath,
		Resurrection: deathResurrectionResult{
			LocationID: int(location), LocationName: location.String(), X: int(position.X), Y: int(position.Y),
			Floor: int(game_state.ResurrectionFloor),
		},
	}
}

func newDebugOptionsStatusResult(state *game_state.GameState) debugOptionsStatusResult {
	snapshot := state.CurrentDebugOptionsSnapshot()

	return debugOptionsStatusResult{
		FreeMove: snapshot.FreeMove, TextPlaceholdersEnabled: snapshot.TextPlaceholdersEnabled,
		DialogBoundsEnabled:          snapshot.DialogBoundingBoxesShown,
		UnlimitedCastingResources:    snapshot.UnlimitedCastingResources,
		BypassSpellLevelRestrictions: snapshot.BypassSpellLevelRestrictions,
		LightingOverlayEnabled:       snapshot.LightingOverlayEnabled,
	}
}

func debugOptionMutationResult(request debugging.Request, mode debugging.ExecutionMode, summary, path string, before, after bool, state *game_state.GameState) debugging.Result {
	result := debugSuccessResultWithMode(request, mode, summary, newDebugOptionsStatusResult(state))
	appendSemanticChange(&result.Changes, path, before, after)

	return result
}
