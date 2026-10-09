package game_state

import (
	"testing"

	"github.com/bradhannah/UltimaVRedux/internal/references"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActionGetSmallMapRetainsItemWhenInventoryRejectsTransfer(t *testing.T) {
	state, _ := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithPlayerAt(10, 10).
		WithSystemCallbacks().
		Build()
	target := references.Position{X: 10, Y: 9}
	require.NoError(t, state.ItemStacksMap.Push(&target, &references.ItemStack{Items: references.ItemStacks{
		{Item: references.InLor, Quantity: 1},
	}}))

	assert.False(t, state.ActionGetSmallMap(references.Up))
	stack, found := state.ItemStacksMap.Snapshot(&target)
	require.True(t, found)
	require.Len(t, stack.Items, 1)
	assert.Equal(t, references.InLor, stack.Items[0].Item)
}
