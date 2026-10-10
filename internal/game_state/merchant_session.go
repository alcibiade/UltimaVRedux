package game_state

import (
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/map_units"
	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

var (
	// ErrInvalidMerchantSession indicates that a session was not issued by this
	// GameState or no longer matches its canonical merchant and buyer identity.
	ErrInvalidMerchantSession = errors.New("invalid merchant session")
	// ErrMerchantClosed indicates that the merchant has not arrived at work or
	// is already expected to leave it.
	ErrMerchantClosed = errors.New("merchant is closed")
	// ErrMerchantHorsebackRefusal indicates the original rider-only refusal.
	ErrMerchantHorsebackRefusal = errors.New("merchant refuses horseback rider")
	// ErrMerchantBuyerUnavailable indicates that no eligible pricing character exists.
	ErrMerchantBuyerUnavailable = errors.New("merchant pricing character unavailable")
)

// MerchantSession is an immutable value issued by GameState for one concrete
// merchant interaction. Its private pointers bind the value to the issuing
// state, loaded NPC, and captured pricing-character identity.
type MerchantSession struct {
	owner              *GameState
	sequence           uint64
	bypassAvailability bool
	merchant           references.MerchantDefinition
	npc                *map_units.NPCFriendly
	sprite             indexes.SpriteIndex
	buyer              *party_state.PlayerCharacter
	buyerIndex         int
	buyerName          string
	buyerIntelligence  int
}

// DebugMerchantSessionReview reports the real loaded canonical NPC and pricing identity without issuing a session.
type DebugMerchantSessionReview struct {
	Merchant              references.MerchantDefinition
	MerchantSpriteIndex   indexes.SpriteIndex
	PricingCharacterIndex int
	PricingCharacterName  string
	NPCFloor              references.FloorNumber
}

// Merchant returns the canonical merchant definition captured by this session.
func (s MerchantSession) Merchant() references.MerchantDefinition { return s.merchant }

// MerchantID returns the merchant's stable canonical identity.
func (s MerchantSession) MerchantID() references.MerchantID { return s.merchant.ID }

// Family returns the merchant's concrete service family.
func (s MerchantSession) Family() references.MerchantFamily { return s.merchant.Family }

// MerchantSpriteIndex returns the detached sprite identity captured from the loaded merchant.
func (s MerchantSession) MerchantSpriteIndex() indexes.SpriteIndex { return s.sprite }

// PricingCharacterIndex returns the captured zero-based active roster index.
func (s MerchantSession) PricingCharacterIndex() int { return s.buyerIndex }

// PricingCharacterName returns the captured pricing character's stable display identity.
func (s MerchantSession) PricingCharacterName() string {
	return s.buyerName
}

// BeginMerchantSession resolves and validates one loaded merchant NPC and
// captures the pricing character once for the lifetime of the session.
func (g *GameState) BeginMerchantSession(npc *map_units.NPCFriendly) (MerchantSession, error) {
	return g.beginMerchantSession(npc, false)
}

func (g *GameState) beginMerchantSession(
	npc *map_units.NPCFriendly,
	bypassAvailability bool,
) (MerchantSession, error) {
	if g == nil || g.GameReferences == nil || g.GameReferences.MerchantReferences == nil {
		return MerchantSession{}, fmt.Errorf("%w: merchant references are unavailable", ErrInvalidMerchantSession)
	}
	if err := g.validateLoadedMerchantNPC(npc); err != nil {
		return MerchantSession{}, err
	}

	location := g.MapState.PlayerLocation.Location
	merchant, err := g.GameReferences.MerchantReferences.Resolve(
		references.DialogNumber(npc.NPCReference.DialogNumber),
		location,
		npc.MapUnitDetails().NPCNum,
	)
	if err != nil {
		return MerchantSession{}, fmt.Errorf("%w: resolving merchant: %w", ErrInvalidMerchantSession, err)
	}
	if !bypassAvailability {
		if err := g.validateMerchantAvailability(npc, merchant.Family); err != nil {
			return MerchantSession{}, err
		}
	}

	buyer, buyerIndex := g.merchantPricingCharacter()
	if buyer == nil {
		return MerchantSession{}, ErrMerchantBuyerUnavailable
	}

	g.tavernCustomerSessionSequence = 0
	g.merchantSessionSequence++

	return MerchantSession{
		owner:              g,
		sequence:           g.merchantSessionSequence,
		bypassAvailability: bypassAvailability,
		merchant:           merchant,
		npc:                npc,
		sprite:             npc.GetKeySpriteIndex(),
		buyer:              buyer,
		buyerIndex:         buyerIndex,
		buyerName:          buyer.GetNameAsString(),
		buyerIntelligence:  int(buyer.Intelligence),
	}, nil
}

// BeginDebugMerchantSessionByID resolves the real loaded canonical NPC, applies
// Talk's TALK3 one-shot, and bypasses only schedule and horseback session gates.
func (g *GameState) BeginDebugMerchantSessionByID(id references.MerchantID) (MerchantSession, error) {
	_, npc, err := g.resolveLoadedMerchantByID(id)
	if err != nil {
		return MerchantSession{}, err
	}
	npc.ApplyTalk3OneShot()

	return g.beginMerchantSession(npc, true)
}

// ReviewDebugMerchantSessionByID validates a debug launch against the real
// loaded canonical NPC without changing the current merchant-session token.
func (g *GameState) ReviewDebugMerchantSessionByID(
	id references.MerchantID,
) (DebugMerchantSessionReview, error) {
	merchant, npc, err := g.resolveLoadedMerchantByID(id)
	if err != nil {
		return DebugMerchantSessionReview{}, err
	}
	if err = g.validateLoadedMerchantNPC(npc); err != nil {
		return DebugMerchantSessionReview{}, err
	}
	resolved, err := g.GameReferences.MerchantReferences.Resolve(
		references.DialogNumber(npc.NPCReference.DialogNumber),
		npc.NPCReference.Location,
		npc.MapUnitDetails().NPCNum,
	)
	if err != nil || resolved != merchant {
		return DebugMerchantSessionReview{}, fmt.Errorf(
			"%w: loaded merchant no longer matches %q",
			ErrInvalidMerchantSession,
			id,
		)
	}
	buyer, buyerIndex := g.merchantPricingCharacter()
	if buyer == nil {
		return DebugMerchantSessionReview{}, ErrMerchantBuyerUnavailable
	}

	return DebugMerchantSessionReview{
		Merchant:              merchant,
		MerchantSpriteIndex:   npc.GetKeySpriteIndex(),
		PricingCharacterIndex: buyerIndex,
		PricingCharacterName:  buyer.GetNameAsString(),
		NPCFloor:              npc.Floor(),
	}, nil
}

func (g *GameState) resolveLoadedMerchantByID(
	id references.MerchantID,
) (references.MerchantDefinition, *map_units.NPCFriendly, error) {
	if g == nil || g.GameReferences == nil || g.GameReferences.MerchantReferences == nil {
		return references.MerchantDefinition{}, nil, fmt.Errorf(
			"%w: merchant references are unavailable",
			ErrInvalidMerchantSession,
		)
	}
	merchant, ok := g.GameReferences.MerchantReferences.Definition(id)
	if !ok {
		return references.MerchantDefinition{}, nil, fmt.Errorf(
			"%w: unknown merchant ID %q",
			ErrInvalidMerchantSession,
			id,
		)
	}
	if merchant.Location != g.MapState.PlayerLocation.Location {
		return references.MerchantDefinition{}, nil, fmt.Errorf(
			"%w: merchant location %s does not match current location %s",
			ErrInvalidMerchantSession,
			merchant.Location,
			g.MapState.PlayerLocation.Location,
		)
	}
	if g.CurrentNPCAIController == nil || g.CurrentNPCAIController.GetNpcs() == nil {
		return references.MerchantDefinition{}, nil, fmt.Errorf(
			"%w: current NPC controller is unavailable",
			ErrInvalidMerchantSession,
		)
	}

	for _, mapUnit := range *g.CurrentNPCAIController.GetNpcs() {
		npc, isFriendly := mapUnit.(*map_units.NPCFriendly)
		if !isFriendly || npc.MapUnitDetails().NPCNum != merchant.NPCIndex ||
			npc.NPCReference.Location != merchant.Location {
			continue
		}

		return merchant, npc, nil
	}

	return references.MerchantDefinition{}, nil, fmt.Errorf(
		"%w: merchant %q is not loaded by the current controller",
		ErrInvalidMerchantSession,
		id,
	)
}

// BeginMerchantSessionByID resolves a stable merchant identity to the exact
// loaded NPC, then delegates all production validation to BeginMerchantSession.
func (g *GameState) BeginMerchantSessionByID(id references.MerchantID) (MerchantSession, error) {
	_, npc, err := g.resolveLoadedMerchantByID(id)
	if err != nil {
		return MerchantSession{}, err
	}

	return g.BeginMerchantSession(npc)
}

// EndMerchantSession invalidates session when it is still the current token.
// Ending an older session must not close a newer interaction.
func (g *GameState) EndMerchantSession(session MerchantSession) {
	if g == nil || session.owner != g || session.sequence == 0 ||
		session.sequence != g.merchantSessionSequence {
		return
	}

	g.tavernCustomerSessionSequence = 0
	g.merchantSessionSequence++
}

func (g *GameState) validateLoadedMerchantNPC(npc *map_units.NPCFriendly) error {
	if npc == nil {
		return fmt.Errorf("%w: merchant NPC is required", ErrInvalidMerchantSession)
	}
	if g.CurrentNPCAIController == nil {
		return fmt.Errorf("%w: current NPC controller is unavailable", ErrInvalidMerchantSession)
	}
	mapUnits := g.CurrentNPCAIController.GetNpcs()
	if mapUnits == nil {
		return fmt.Errorf("%w: current NPC controller is unavailable", ErrInvalidMerchantSession)
	}

	loaded := false
	for _, mapUnit := range *mapUnits {
		if mapUnit == npc {
			loaded = true

			break
		}
	}
	if !loaded {
		return fmt.Errorf("%w: merchant NPC is not managed by the current controller", ErrInvalidMerchantSession)
	}
	if npc.IsEmptyMapUnit() || !npc.IsVisible() {
		return fmt.Errorf("%w: merchant NPC is empty or hidden", ErrInvalidMerchantSession)
	}
	if npc.Floor() != g.MapState.PlayerLocation.Floor {
		return fmt.Errorf(
			"%w: NPC floor %d does not match current floor %d",
			ErrInvalidMerchantSession,
			npc.Floor(),
			g.MapState.PlayerLocation.Floor,
		)
	}
	if npc.NPCReference.Location != g.MapState.PlayerLocation.Location {
		return fmt.Errorf(
			"%w: NPC location %s does not match current location %s",
			ErrInvalidMerchantSession,
			npc.NPCReference.Location,
			g.MapState.PlayerLocation.Location,
		)
	}

	return nil
}

func (g *GameState) validateMerchantAvailability(
	npc *map_units.NPCFriendly,
	family references.MerchantFamily,
) error {
	arrived := npc.ArrivedScheduleSlot()
	desired := npc.NPCReference.Schedule.DesiredSlot(g.DateTime)
	if !arrived.IsMerchantSlot() || !desired.IsMerchantSlot() {
		return fmt.Errorf("%w: arrived slot %d, desired slot %d", ErrMerchantClosed, arrived, desired)
	}
	if g.PartyVehicle.GetVehicleDetails().VehicleType == references.HorseVehicle &&
		family != references.MerchantFamilyHorse {
		return ErrMerchantHorsebackRefusal
	}

	return nil
}

func (g *GameState) merchantPricingCharacter() (*party_state.PlayerCharacter, int) {
	if index := g.GetActivePlayerIndex(); index >= 0 && index < party_state.MaxActiveParty {
		candidate := g.PartyState.GetCharacter(index)
		if isEligibleMerchantBuyer(candidate) {
			return candidate, index
		}
	}

	var selected *party_state.PlayerCharacter
	selectedIndex := -1
	g.PartyState.ForEachCharacter(func(index int, candidate *party_state.PlayerCharacter) {
		if selected == nil && index < party_state.MaxActiveParty && isEligibleMerchantBuyer(candidate) {
			selected = candidate
			selectedIndex = index
		}
	})

	return selected, selectedIndex
}

func isEligibleMerchantBuyer(character *party_state.PlayerCharacter) bool {
	return character != nil && character.PartyMap == party_state.PartyMapActive &&
		(character.IsGood() || character.IsPoisoned())
}

func (g *GameState) validateMerchantSession(
	session MerchantSession,
	wantedFamily references.MerchantFamily,
) (*party_state.PlayerCharacter, references.MerchantDefinition, error) {
	merchant, err := g.validateMerchantSessionIdentity(session, wantedFamily)
	if err != nil {
		return nil, references.MerchantDefinition{}, err
	}
	currentBuyer := g.merchantCharacterAt(session.buyerIndex)
	if currentBuyer != session.buyer || !isEligibleMerchantBuyer(session.buyer) ||
		session.buyer.GetNameAsString() != session.buyerName {
		return nil, references.MerchantDefinition{}, fmt.Errorf(
			"%w: captured pricing character is no longer eligible",
			ErrInvalidMerchantSession,
		)
	}

	return session.buyer, merchant, nil
}

func (g *GameState) validateMerchantSessionIdentity(
	session MerchantSession,
	wantedFamily references.MerchantFamily,
) (references.MerchantDefinition, error) {
	if session.owner != g || session.sequence == 0 || session.sequence != g.merchantSessionSequence {
		return references.MerchantDefinition{}, fmt.Errorf(
			"%w: session was not issued by this GameState or has been superseded",
			ErrInvalidMerchantSession,
		)
	}
	if session.merchant.Family != wantedFamily {
		return references.MerchantDefinition{}, fmt.Errorf(
			"%w: merchant family 0x%02x is not 0x%02x",
			ErrInvalidMerchantSession,
			byte(session.merchant.Family),
			byte(wantedFamily),
		)
	}
	if g.GameReferences == nil || g.GameReferences.MerchantReferences == nil {
		return references.MerchantDefinition{}, fmt.Errorf(
			"%w: merchant references or NPC are unavailable",
			ErrInvalidMerchantSession,
		)
	}
	if err := g.validateLoadedMerchantNPC(session.npc); err != nil {
		return references.MerchantDefinition{}, err
	}
	if g.MapState.PlayerLocation.Location != session.merchant.Location ||
		session.npc.NPCReference.Location != session.merchant.Location {
		return references.MerchantDefinition{}, fmt.Errorf(
			"%w: merchant location changed",
			ErrInvalidMerchantSession,
		)
	}
	merchant, err := g.GameReferences.MerchantReferences.Resolve(
		references.DialogNumber(session.npc.NPCReference.DialogNumber),
		session.npc.NPCReference.Location,
		session.npc.MapUnitDetails().NPCNum,
	)
	if err != nil {
		return references.MerchantDefinition{}, fmt.Errorf(
			"%w: resolving canonical merchant: %w",
			ErrInvalidMerchantSession,
			err,
		)
	}
	if merchant != session.merchant {
		return references.MerchantDefinition{}, fmt.Errorf(
			"%w: canonical merchant no longer matches",
			ErrInvalidMerchantSession,
		)
	}
	if !session.bypassAvailability {
		if err := g.validateMerchantAvailability(session.npc, merchant.Family); err != nil {
			return references.MerchantDefinition{}, err
		}
	}

	return merchant, nil
}

// validateInnMerchantSession preserves every canonical session gate while
// allowing only the Inn family to move or store its captured pricing record.
func (g *GameState) validateInnMerchantSession(
	session MerchantSession,
) (*party_state.PlayerCharacter, int, references.MerchantDefinition, error) {
	merchant, err := g.validateMerchantSessionIdentity(session, references.MerchantFamilyInn)
	if err != nil {
		return nil, -1, references.MerchantDefinition{}, err
	}
	buyerIndex := g.merchantCharacterIndex(session.buyer)
	if buyerIndex < 0 || session.buyer == nil || session.buyer.GetNameAsString() != session.buyerName {
		return nil, -1, references.MerchantDefinition{}, fmt.Errorf(
			"%w: captured pricing character is no longer eligible",
			ErrInvalidMerchantSession,
		)
	}

	return session.buyer, buyerIndex, merchant, nil
}

func (g *GameState) merchantCharacterAt(wantedIndex int) *party_state.PlayerCharacter {
	if wantedIndex < 0 || wantedIndex >= party_state.MaxActiveParty {
		return nil
	}

	var found *party_state.PlayerCharacter
	g.PartyState.ForEachCharacter(func(index int, character *party_state.PlayerCharacter) {
		if index == wantedIndex {
			found = character
		}
	})

	return found
}

func (g *GameState) merchantCharacterIndex(wanted *party_state.PlayerCharacter) int {
	if wanted == nil {
		return -1
	}
	for index := range party_state.NRosterSlots {
		if g.PartyState.HasCharacter(index) && g.PartyState.GetCharacter(index) == wanted {
			return index
		}
	}

	return -1
}
