package game_state

import (
	"github.com/bradhannah/UltimaVRedux/internal/event_queue"
	"github.com/bradhannah/UltimaVRedux/internal/references"
)

// ProjectileAnimationSequence builds reusable projectile travel followed by an impact callback.
func (g *GameState) ProjectileAnimationSequence(
	from, to references.Position,
	missile references.MissileType,
	onImpact func(),
) event_queue.Sequence {
	if onImpact == nil {
		panic("projectile impact callback is required")
	}

	return func(sequence *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
		var durationMs uint64

		return sequence.Then(func() {
			durationMs = g.SystemCallbacks.Visual.StartProjectile(from, to, missile)
		}).ThenSequence(func(sequence *event_queue.SequentialEventBuilder) *event_queue.SequentialEventBuilder {
			return sequence.Animate(durationMs, func(progress float32) {
				g.SystemCallbacks.Visual.UpdateProjectileProgress(progress)
			})
		}).Then(func() {
			g.SystemCallbacks.Visual.ClearProjectile()
		}).Then(onImpact)
	}
}
