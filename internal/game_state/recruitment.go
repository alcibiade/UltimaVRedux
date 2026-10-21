package game_state

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

var (
	// ErrRecruitmentSourceUnavailable indicates that the requested source is not the current live town NPC.
	ErrRecruitmentSourceUnavailable = errors.New("recruitment source NPC is unavailable")
	// ErrInvalidRecruitment indicates that the requested source identity cannot be persisted in NPC flags.
	ErrInvalidRecruitment = errors.New("invalid recruitment request")
)

// RecruitNPC atomically recruits one live small-map NPC. All fallible checks
// precede mutation, then roster activation, persistent source absence, and live
// removal commit as one update-thread operation.
func (g *GameState) RecruitNPC(npc *map_units.NPCFriendly, npcName string) error {
	if npc == nil || strings.TrimSpace(npcName) == "" {
		return fmt.Errorf("%w: source NPC and name are required", ErrInvalidRecruitment)
	}
	location := g.MapState.PlayerLocation.Location
	npcSlot := npc.MapUnitDetails().NPCNum
	if location < references.Moonglow || location > references.SerpentsHold ||
		npc.NPCReference.Location != location || npcSlot < 0 || npcSlot >= party_state.NPcsPerLocation {
		return fmt.Errorf("%w: source location %d and NPC slot %d are not persistent town identities", ErrInvalidRecruitment, location, npcSlot)
	}
	if !g.PartyState.HasRoom() {
		return ErrPartyCapacity
	}
	if g.CurrentNPCAIController == nil {
		return ErrRecruitmentSourceUnavailable
	}
	units := g.CurrentNPCAIController.GetNpcs()
	live := units.GetMapUnitAtPositionAndFloorOrNil(npc.Pos(), npc.Floor())
	if live == nil || *live != npc || !npc.IsVisible() {
		return ErrRecruitmentSourceUnavailable
	}

	// JoinNPC performs its name lookup before changing the roster. Once it
	// succeeds, the validated live source can no longer produce a recoverable failure.
	if err := g.PartyState.JoinNPC(npc.NPCReference, npcName); err != nil {
		return err
	}
	g.PartyState.SetDeadNpc(location, npcSlot)
	if !units.RemoveNPCAtPositionAndFloor(npc.Pos(), npc.Floor()) {
		panic("validated recruitment source could not be removed")
	}

	return nil
}
