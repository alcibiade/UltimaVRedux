//nolint:lll // The cohesive preset builder retains a fully qualified signature beside its scoped lint rationale.
package game_state

import (
	"errors"
	"fmt"
	"hash"

	"github.com/bradhannah/UltimaVRedux/internal/party_state"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

const (
	// UsableInventoryQuestItemTarget is the absolute quest-item quantity applied by the usable preset.
	UsableInventoryQuestItemTarget uint16 = 1
	// UsableInventoryMoonstoneTarget is the absolute moonstone quantity applied by the usable preset.
	UsableInventoryMoonstoneTarget uint16 = 1
	// UsableInventoryPotionTarget is the absolute potion quantity applied by the usable preset.
	UsableInventoryPotionTarget uint16 = 5
	// UsableInventoryScrollTarget is the absolute scroll quantity applied by the usable preset.
	UsableInventoryScrollTarget uint16 = 3
)

var (
	// ErrUsableInventoryPresetStale means the reviewed inventory no longer matches live state.
	ErrUsableInventoryPresetStale = errors.New("usable inventory changed after review")
	// ErrUsableInventoryPresetInvalid means the preset cannot be safely applied.
	ErrUsableInventoryPresetInvalid = errors.New("invalid usable inventory preset")
)

// UsableInventoryPresetReview is a detached review of the exact absolute changes the preset would apply.
type UsableInventoryPresetReview struct {
	Fingerprint string
	Changes     []DebugInventoryChange
}

// UsableInventoryPresetResult reports the committed changes and resulting review fingerprint.
type UsableInventoryPresetResult struct {
	BeforeFingerprint string
	AfterFingerprint  string
	Changes           []DebugInventoryChange
}

// ReviewUsableInventoryPreset returns an idempotent preset preview derived from authoritative item catalogs.
func (g *GameState) ReviewUsableInventoryPreset() (UsableInventoryPresetReview, error) {
	mutations, fingerprint, err := g.usableInventoryPresetMutations()
	if err != nil {
		return UsableInventoryPresetReview{}, err
	}
	changes := make([]DebugInventoryChange, 0, len(mutations))
	for _, mutation := range mutations {
		changes = append(changes, mutation.change)
	}

	return UsableInventoryPresetReview{Fingerprint: fingerprint, Changes: changes}, nil
}

// ApplyUsableInventoryPreset validates a reviewed fingerprint and atomically applies absolute target quantities.
func (g *GameState) ApplyUsableInventoryPreset(expectedFingerprint string) (UsableInventoryPresetResult, error) {
	if err := g.validateReviewedMutationContext("usable inventory preset"); err != nil {
		return UsableInventoryPresetResult{}, fmt.Errorf("%w: %w", ErrUsableInventoryPresetInvalid, err)
	}
	mutations, beforeFingerprint, err := g.usableInventoryPresetMutations()
	if err != nil {
		return UsableInventoryPresetResult{}, err
	}
	if expectedFingerprint == "" || expectedFingerprint != beforeFingerprint {
		return UsableInventoryPresetResult{}, ErrUsableInventoryPresetStale
	}

	changes := g.applyDebugInventoryMutations(mutations)
	_, afterFingerprint, err := g.usableInventoryPresetMutations()
	if err != nil {
		panic("validated usable inventory catalog became invalid: " + err.Error())
	}

	return UsableInventoryPresetResult{
		BeforeFingerprint: beforeFingerprint,
		AfterFingerprint:  afterFingerprint,
		Changes:           changes,
	}, nil
}

//nolint:cyclop // Each independent catalog has distinct validation and inventory mutation semantics.
func (g *GameState) usableInventoryPresetMutations() ([]debugInventoryMutation, string, error) { //nolint:funlen // The four catalog loops form one atomic preset preview.
	collector := debugInventoryMutations{
		game:     g,
		snapshot: g.GetDebugInventorySnapshot(),
		mutations: make([]debugInventoryMutation, 0,
			references.NumberOfQuestItems+references.NumberOfMoonstones+references.NumberOfPotions+references.NumberOfScrolls),
	}
	hasher := reviewedMutationFingerprint(func(digest hash.Hash) {
		for _, item := range references.QuestItemCatalog() {
			quantity := g.PartyState.Inventory.QuestItems.Get(item)
			writeFingerprintf(digest, "q:%d:%d;", item.ID(), quantity)
		}
		for _, moonstone := range references.MoonstoneCatalog() {
			quantity := g.PartyState.Inventory.Moonstones.Get(moonstone)
			writeFingerprintf(digest, "m:%d:%d;", moonstone.ID(), quantity)
		}
		for _, potion := range references.PotionCatalog() {
			writeFingerprintf(digest, "p:%d:%d;", potion.ID(), collector.snapshot.Potions[potion])
		}
		for _, scroll := range references.ScrollCatalog() {
			writeFingerprintf(digest, "s:%d:%d;", scroll.ID(), collector.snapshot.Scrolls[scroll])
		}
	})

	for _, item := range references.QuestItemCatalog() {
		if !item.IsValid() || UsableInventoryQuestItemTarget > party_state.ItemQuantitySmallMax {
			return nil, "", fmt.Errorf("%w: quest item catalog entry %d", ErrUsableInventoryPresetInvalid, item)
		}
		before := g.PartyState.Inventory.QuestItems.Get(item)
		if before != UsableInventoryQuestItemTarget {
			collector.mutations = append(collector.mutations, debugInventoryMutation{
				change: DebugInventoryChange{
					Path: fmt.Sprintf("quest_items.%d", item.ID()), Before: before, After: UsableInventoryQuestItemTarget,
				},
				apply: func() { g.PartyState.Inventory.QuestItems.Set(item, UsableInventoryQuestItemTarget) },
			})
		}
	}
	for _, moonstone := range references.MoonstoneCatalog() {
		if moonstone < references.NewMoon || moonstone > references.CrescentWaning ||
			UsableInventoryMoonstoneTarget > party_state.ItemQuantitySmallMax {
			return nil, "", fmt.Errorf("%w: moonstone catalog entry %d", ErrUsableInventoryPresetInvalid, moonstone)
		}
		before := g.PartyState.Inventory.Moonstones.Get(moonstone)
		if before != UsableInventoryMoonstoneTarget {
			collector.mutations = append(collector.mutations, debugInventoryMutation{
				change: DebugInventoryChange{
					Path: fmt.Sprintf("moonstones.%d", moonstone.ID()), Before: before, After: UsableInventoryMoonstoneTarget,
				},
				apply: func() { g.PartyState.Inventory.Moonstones.Set(moonstone, UsableInventoryMoonstoneTarget) },
			})
		}
	}
	for _, potion := range references.PotionCatalog() {
		if potion < references.Blue || potion > references.White ||
			UsableInventoryPotionTarget > party_state.ItemQuantitySmallMax {
			return nil, "", fmt.Errorf("%w: potion catalog entry %d", ErrUsableInventoryPresetInvalid, potion)
		}
		collector.appendPotion(potion, UsableInventoryPotionTarget)
	}
	for _, scroll := range references.ScrollCatalog() {
		if scroll < references.ScrollVasLor || scroll > references.ScrollAnTym ||
			UsableInventoryScrollTarget > party_state.ItemQuantityLargeMax {
			return nil, "", fmt.Errorf("%w: scroll catalog entry %d", ErrUsableInventoryPresetInvalid, scroll)
		}
		collector.appendScroll(scroll, UsableInventoryScrollTarget)
	}

	return collector.mutations, hasher, nil
}
