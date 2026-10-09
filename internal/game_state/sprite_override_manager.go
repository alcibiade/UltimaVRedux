package game_state

import (
	"sync"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
)

// SpriteOverrideManager manages sprite overrides for characters and NPCs across different contexts.
// This system handles temporary visual state changes that are not tied to specific map positions.
//
// Examples:
// - Bard playing music animation during camping
// - Sleeping characters vs awake guards during camping
// - Combat status effects (stunned, poisoned, defending)
// - Cutscene-specific character animations
//
// This is separate from GetEffectiveTile which handles position-based map tile overrides.
type SpriteOverrideManager struct {
	overrides map[OverrideKey]indexes.SpriteIndex
	mu        sync.RWMutex
}

// OverrideKey uniquely identifies a sprite override by context and entity.
type OverrideKey struct {
	Context  string // The scenario context: "camping", "combat", "cutscene", "dialogue"
	EntityID string // The entity identifier: "party_0", "guard", "npc_merchant_5"
}

// NewSpriteOverrideManager creates a new sprite override manager.
func NewSpriteOverrideManager() *SpriteOverrideManager {
	return &SpriteOverrideManager{
		overrides: make(map[OverrideKey]indexes.SpriteIndex),
	}
}

// Set registers a sprite override for the specified context and entity.
// If an override already exists for this context/entity pair, it will be replaced.
func (m *SpriteOverrideManager) Set(context, entityID string, sprite indexes.SpriteIndex) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := OverrideKey{
		Context:  context,
		EntityID: entityID,
	}
	m.overrides[key] = sprite
}

// Clear removes a specific sprite override for the given context and entity.
// If no override exists for this combination, this is a no-op.
func (m *SpriteOverrideManager) Clear(context, entityID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := OverrideKey{
		Context:  context,
		EntityID: entityID,
	}
	delete(m.overrides, key)
}

// ClearContext removes all sprite overrides for the specified context.
// This is useful when transitioning between game states (e.g., leaving camping mode).
func (m *SpriteOverrideManager) ClearContext(context string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Iterate through all overrides and remove those matching the context
	for key := range m.overrides {
		if key.Context == context {
			delete(m.overrides, key)
		}
	}
}

// Get retrieves the sprite override for the specified context and entity.
// Returns the sprite index and true if an override exists, or zero value and false if not.
func (m *SpriteOverrideManager) Get(context, entityID string) (indexes.SpriteIndex, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := OverrideKey{
		Context:  context,
		EntityID: entityID,
	}
	sprite, exists := m.overrides[key]

	return sprite, exists
}
