package game_state

import "github.com/bradhannah/UltimaVRedux/internal/references"

// CommitCombatSkullKeyUse resolves a confirmed direction atomically from the
// retained PC origin. A paid miss still commits; cancellation and stale review
// spend nothing. The caller owns combat action completion. See ref:PROV-0006.
func (g *GameState) CommitCombatSkullKeyUse(
	commit *SkullKeyUseCommit,
	direction references.Direction,
) (SkullKeyUnlockReview, bool) {
	if commit == nil || commit.state != g || commit.committed || commit.owner.combat == nil ||
		(direction != references.Up && direction != references.Down &&
			direction != references.Left && direction != references.Right) {
		return SkullKeyUnlockReview{}, false
	}
	if g.CommitSkullKeyUse(commit) != SkullKeyUseConsumed {
		return SkullKeyUnlockReview{}, false
	}
	review := g.ReviewSkullKeyUnlock(commit, direction)
	review.Result = g.commitSkullKeyUnlock(review, false)

	return review, true
}
