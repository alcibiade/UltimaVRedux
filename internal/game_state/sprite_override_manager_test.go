package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/sprites/indexes"
	"github.com/stretchr/testify/assert"
)

func TestSpriteOverrideManager_BasicOperations(t *testing.T) {
	manager := NewSpriteOverrideManager()

	// Test initial state - no overrides should exist
	_, exists := manager.Get("camping", "guard")
	assert.False(t, exists)

	// Test setting an override
	testSprite := indexes.BardPlaying_KeyIndex
	manager.Set("camping", "guard", testSprite)

	// Verify override was set

	sprite, exists := manager.Get("camping", "guard")
	assert.True(t, exists)
	assert.Equal(t, testSprite, sprite)

	// Test non-existent override
	_, exists = manager.Get("camping", "nonexistent")
	assert.False(t, exists)

	// Test different context
	_, exists = manager.Get("combat", "guard")
	assert.False(t, exists)
}

func TestSpriteOverrideManager_ClearOperations(t *testing.T) {
	manager := NewSpriteOverrideManager()

	// Set up test data
	manager.Set("camping", "guard", indexes.BardPlaying_KeyIndex)
	manager.Set("camping", "party_0", indexes.SpriteIndex(indexes.DeadBody))
	manager.Set("combat", "party_1", indexes.SpriteIndex(indexes.DeadBody))

	// Test clearing specific override
	manager.Clear("camping", "guard")

	_, exists := manager.Get("camping", "guard")
	assert.False(t, exists)

	// Other overrides should still exist
	_, exists = manager.Get("camping", "party_0")
	assert.True(t, exists)

	// Test clearing entire context
	manager.ClearContext("camping")

	// Camping overrides should be gone
	_, exists = manager.Get("camping", "party_0")
	assert.False(t, exists)

	// Combat override should still exist
	_, exists = manager.Get("combat", "party_1")
	assert.True(t, exists)
}

func TestSpriteOverrideManager_ReplaceOverride(t *testing.T) {
	manager := NewSpriteOverrideManager()

	// Set initial override
	original := indexes.BardPlaying_KeyIndex
	manager.Set("camping", "guard", original)

	// Replace with new override
	replacement := indexes.SpriteIndex(indexes.DeadBody)
	manager.Set("camping", "guard", replacement)

	// Should have the new value
	sprite, exists := manager.Get("camping", "guard")
	assert.True(t, exists)
	assert.Equal(t, replacement, sprite)
	assert.NotEqual(t, original, sprite)
}

func TestSpriteOverrideManager_ConcurrentAccess(t *testing.T) {
	manager := NewSpriteOverrideManager()

	// This is a basic test for concurrent access
	// In a real concurrent scenario, we'd use goroutines
	manager.Set("camping", "guard", indexes.BardPlaying_KeyIndex)

	// Concurrent reads should not interfere
	sprite1, exists1 := manager.Get("camping", "guard")
	sprite2, exists2 := manager.Get("camping", "guard")

	assert.True(t, exists1)
	assert.True(t, exists2)
	assert.Equal(t, sprite1, sprite2)
}
