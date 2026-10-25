package game_state

import (
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// MerchantShopReview is a read-only production snapshot of one shop with its
// merchant's canonical sprite identity.
type MerchantShopReview[S any] struct {
	State               S
	MerchantSpriteIndex indexes.SpriteIndex
}

// canonicalMerchant resolves a merchant ID to its definition and canonical NPC
// reference for reviews that run without a live session. wrongFamily is the
// family-specific error returned when the ID names another kind of merchant.
func (g *GameState) canonicalMerchant(
	id references.MerchantID,
	family references.MerchantFamily,
	wrongFamily error,
	role string,
) (references.MerchantDefinition, references.NPCReference, error) {
	var none references.MerchantDefinition
	if g == nil || g.GameReferences == nil || g.GameReferences.MerchantReferences == nil ||
		g.GameReferences.NPCReferences == nil || g.GameReferences.LocationReferences == nil {
		return none, references.NPCReference{}, fmt.Errorf("%w: merchant references are unavailable", ErrInvalidMerchantSession)
	}
	merchant, ok := g.GameReferences.MerchantReferences.Definition(id)
	if !ok {
		return none, references.NPCReference{}, fmt.Errorf("%w: unknown merchant ID %q", ErrInvalidMerchantSession, id)
	}
	if merchant.Family != family {
		return none, references.NPCReference{}, fmt.Errorf("%w: merchant %q is not %s", wrongFamily, id, role)
	}
	if merchant.Location.GetMapType() != references.SmallMapType ||
		g.GameReferences.LocationReferences.GetLocationReference(merchant.Location) == nil {
		return none, references.NPCReference{}, fmt.Errorf(
			"%w: merchant %q has no canonical small-map location", ErrInvalidMerchantSession, id)
	}
	npcs := g.GameReferences.NPCReferences.GetNPCReferencesByLocation(merchant.Location)
	if npcs == nil || merchant.NPCIndex < 0 || merchant.NPCIndex >= len(*npcs) {
		return none, references.NPCReference{}, fmt.Errorf(
			"%w: merchant %q has no canonical NPC reference", ErrInvalidMerchantSession, id)
	}
	npcReference := (*npcs)[merchant.NPCIndex]
	resolved, err := g.GameReferences.MerchantReferences.Resolve(
		references.DialogNumber(npcReference.DialogNumber), npcReference.Location, merchant.NPCIndex,
	)
	if err != nil || resolved != merchant {
		return none, references.NPCReference{}, fmt.Errorf(
			"%w: canonical NPC reference does not match merchant %q", ErrInvalidMerchantSession, id)
	}

	return merchant, npcReference, nil
}

// reviewMerchantShop builds one shop's production state for the current pricing
// character without issuing a session, invoking callbacks, consuming RNG,
// advancing time, or consulting loaded NPC state.
func reviewMerchantShop[S any](
	g *GameState,
	id references.MerchantID,
	family references.MerchantFamily,
	wrongFamily error,
	role string,
	build func(references.MerchantDefinition, *party_state.PlayerCharacter, int) (S, error),
) (MerchantShopReview[S], error) {
	merchant, npcReference, err := g.canonicalMerchant(id, family, wrongFamily, role)
	if err != nil {
		return MerchantShopReview[S]{}, err
	}
	buyer, buyerIndex := g.merchantPricingCharacter()
	if buyer == nil {
		return MerchantShopReview[S]{}, ErrMerchantBuyerUnavailable
	}
	state, err := build(merchant, buyer, buyerIndex)
	if err != nil {
		return MerchantShopReview[S]{}, err
	}

	return MerchantShopReview[S]{State: state, MerchantSpriteIndex: npcReference.GetSpriteIndex()}, nil
}
