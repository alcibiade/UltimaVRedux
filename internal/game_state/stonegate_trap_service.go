package game_state

import (
	"errors"
	"fmt"
	"hash"
	"sort"

	"github.com/bradhannah/UltimaVRedux/internal/map_state"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

var (
	// ErrStonegateTrapInvalid means the volcano trap is unavailable in the current context.
	ErrStonegateTrapInvalid = errors.New("invalid Stonegate trap request")
	// ErrStonegateTrapStale means floor, object, or party state changed after review.
	ErrStonegateTrapStale = errors.New("stonegate trap state changed after review")
)

// StonegateTrapReview is a detached snapshot of all authority destroyed by the trap.
type StonegateTrapReview struct {
	Fingerprint    string
	Floor          references.FloorNumber
	MapUnitCount   int
	ItemStackCount int
	Members        []ActivePartyWipeMemberSnapshot
}

// StonegateTrapOutcome reports the atomic floor flood, object clear, and party wipe.
type StonegateTrapOutcome struct {
	BeforeFingerprint string
	AfterFingerprint  string
	Floor             references.FloorNumber
	ClearedMapUnits   int
	ClearedItemStacks int
	InitialDamage     []ActivePartyWipeMemberChange
	KilledMembers     []ActivePartyWipeMemberChange
}

// ReviewStonegateTrap captures floor, object, and active-party authority at the hazard checkpoint.
func (g *GameState) ReviewStonegateTrap() (StonegateTrapReview, error) {
	if g.MapState.PlayerLocation.Location != references.Stonegate {
		return StonegateTrapReview{}, fmt.Errorf("%w: party is not in Stonegate", ErrStonegateTrapInvalid)
	}
	if g.GetCurrentLayeredMap() == nil || g.CurrentNPCAIController == nil ||
		g.CurrentNPCAIController.GetNpcs() == nil {
		return StonegateTrapReview{}, fmt.Errorf("%w: floor authority is unavailable", ErrStonegateTrapInvalid)
	}
	if tile := g.GetCurrentLayeredMap().GetTileByLayer(map_state.MapLayer, &g.MapState.PlayerLocation.Position); tile == nil || !tile.Is(indexes.BrickFloorHole) {
		return StonegateTrapReview{}, fmt.Errorf("%w: Avatar is not on the volcano trap", ErrStonegateTrapInvalid)
	}
	if g.PartyVehicle.GetVehicleDetails().VehicleType == references.CarpetVehicle || g.PartyState.IsAllPartyDead() {
		return StonegateTrapReview{}, fmt.Errorf("%w: trap requires a living party not riding a carpet", ErrStonegateTrapInvalid)
	}

	members := g.activePartyWipeMembers()
	if len(members) == 0 {
		return StonegateTrapReview{}, fmt.Errorf("%w: active party is empty", ErrStonegateTrapInvalid)
	}

	return StonegateTrapReview{
		Fingerprint:    g.stonegateTrapFingerprint(),
		Floor:          g.MapState.PlayerLocation.Floor,
		MapUnitCount:   len(*g.CurrentNPCAIController.GetNpcs()),
		ItemStackCount: len(g.ItemStacksMap.Entries()),
		Members:        members,
	}, nil
}

// CommitStonegateTrap atomically floods the floor, clears objects, and kills the active party.
func (g *GameState) CommitStonegateTrap(expectedFingerprint string) (StonegateTrapOutcome, error) {
	review, err := g.ReviewStonegateTrap()
	if err != nil {
		return StonegateTrapOutcome{}, err
	}
	if expectedFingerprint == "" || expectedFingerprint != review.Fingerprint {
		return StonegateTrapOutcome{}, ErrStonegateTrapStale
	}

	outcome := StonegateTrapOutcome{
		BeforeFingerprint: review.Fingerprint,
		AfterFingerprint:  review.Fingerprint,
		Floor:             review.Floor,
		ClearedMapUnits:   review.MapUnitCount,
		ClearedItemStacks: review.ItemStackCount,
		InitialDamage:     make([]ActivePartyWipeMemberChange, 0, len(review.Members)),
		KilledMembers:     make([]ActivePartyWipeMemberChange, 0, len(review.Members)),
	}
	// Town status processing calls party damage before the
	// Stonegate branch: independent 1d8 rolls, in active-slot order, including
	// sleeping/poisoned members but excluding the already dead. No armor save.
	// See ref:ENV-0002.
	for _, member := range review.Members {
		if member.Status == party_state.Dead {
			continue
		}
		character := g.PartyState.GetCharacter(member.Slot)
		character.SubtractHp(uint16(g.RandomIntInRange(TrapFallDamageMin, TrapFallDamageMax)))
		g.clearUnavailableActivePlayer()
		outcome.InitialDamage = append(outcome.InitialDamage, ActivePartyWipeMemberChange{
			Before: member, After: activePartyWipeMemberSnapshot(member.Slot, character),
		})
	}
	g.FloodFloorWithLava()
	g.ClearAllObjectsInCurrentLocation()
	for _, member := range review.Members {
		character := g.PartyState.GetCharacter(member.Slot)
		character.SetHp(0)
		character.SetStatus(party_state.Dead)
		outcome.KilledMembers = append(outcome.KilledMembers, ActivePartyWipeMemberChange{
			Before: member,
			After:  activePartyWipeMemberSnapshot(member.Slot, character),
		})
	}
	g.clearUnavailableActivePlayer()
	g.SystemCallbacks.Screen.UpdateStatsDisplay()
	outcome.AfterFingerprint = g.stonegateTrapFingerprint()

	return outcome, nil
}

// processStonegateLavaTrap resolves accepted standing terrain synchronously;
// cosmetic delay can neither stale the review nor cancel the consequence.
func (g *GameState) processStonegateLavaTrap() bool {
	review, err := g.ReviewStonegateTrap()
	if err != nil {
		return false
	}
	outcome, err := g.CommitStonegateTrap(review.Fingerprint)
	if err != nil {
		panic(fmt.Sprintf("Stonegate trap changed during synchronous hazard resolution: %v", err))
	}
	g.SystemCallbacks.Flow.Consequences.TriggerStonegateTrap(outcome)

	return true
}

func (g *GameState) stonegateTrapFingerprint() string {
	currentMap := g.GetCurrentLayeredMap()
	units := g.CurrentNPCAIController.GetNpcs()
	roster := g.CurrentPartyRosterSnapshot()
	location := g.MapState.PlayerLocation

	return reviewedMutationFingerprint(func(digest hash.Hash) {
		writeFingerprintf(
			digest,
			"%s:%d:%d:%d:%d;",
			roster.Fingerprint,
			location.Location,
			location.Floor,
			location.Position.X,
			location.Position.Y,
		)
		for y := range 32 {
			for x := range 32 {
				position := references.Position{X: references.Coordinate(x), Y: references.Coordinate(y)}
				tile := currentMap.GetTileByLayer(map_state.MapLayer, &position)
				if tile == nil {
					writeFingerprintf(digest, "nil,")

					continue
				}
				writeFingerprintf(digest, "%d,", tile.Index)
			}
		}
		for _, unit := range *units {
			details := unit.MapUnitDetails()
			writeFingerprintf(
				digest,
				"%d:%d:%d:%d:%d:%t:%d;",
				unit.GetMapUnitType(),
				unit.Pos().X,
				unit.Pos().Y,
				unit.Floor(),
				unit.GetKeySpriteIndex(),
				unit.IsVisible(),
				details.NPCNum,
			)
		}
		itemStacks := g.ItemStacksMap.Entries()
		sort.Slice(itemStacks, func(i, j int) bool {
			if itemStacks[i].Position.Y != itemStacks[j].Position.Y {
				return itemStacks[i].Position.Y < itemStacks[j].Position.Y
			}

			return itemStacks[i].Position.X < itemStacks[j].Position.X
		})
		for _, entry := range itemStacks {
			writeFingerprintf(digest, "%d:%d[", entry.Position.X, entry.Position.Y)
			for _, item := range entry.Stack.Items {
				if item.Item == nil {
					writeFingerprintf(digest, "nil;")

					continue
				}
				writeFingerprintf(digest, "%d:%d:%d;", item.Item.Type(), item.Item.ID(), item.Quantity)
			}
			writeFingerprintf(digest, "]")
		}
	})
}
