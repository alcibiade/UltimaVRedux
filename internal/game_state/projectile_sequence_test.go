package game_state

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestProjectileAnimationSequenceClearsBeforeImpact(t *testing.T) {
	gs, mock := NewIntegrationTestBuilder(t).
		WithLocation(references.Britain).
		WithSystemCallbacks().
		Build()
	if gs == nil {
		return
	}
	from := references.Position{X: 5, Y: 5}
	to := references.Position{X: 8, Y: 5}
	impactRan := false

	gs.EventQueueManager.Workflow("test_projectile_sequence").
		Priority(event_queue.PriorityHigh).
		Sequential().
		First(func() {}).
		ThenSequence(gs.ProjectileAnimationSequence(from, to, references.MissileCannonBall, func() {
			impactRan = true
			assert.Equal(t, 1, mock.ProjectilesClearedCount)
		})).
		Finally(func() { gs.SystemCallbacks.Visual.ClearProjectile() }).
		Start()
	for range 10 {
		gs.ElapsedMs += 1000
		gs.EventQueueManager.Update()
		if !gs.EventQueueManager.HasActiveWorkflows() {
			break
		}
	}

	require.Len(t, mock.ProjectilesStarted, 1)
	assert.Equal(t, from, mock.ProjectilesStarted[0].FromPos)
	assert.Equal(t, to, mock.ProjectilesStarted[0].ToPos)
	assert.True(t, impactRan)
	assert.Equal(t, 2, mock.ProjectilesClearedCount)
	assert.False(t, gs.EventQueueManager.HasActiveWorkflows())
}
