package game_state

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
)

var errReviewedMutationUnavailable = errors.New("reviewed mutation unavailable")

func (g *GameState) validateReviewedMutationContext(operation string) error {
	if g.IsInCombat() {
		return fmt.Errorf("%w: %s is unavailable during combat", errReviewedMutationUnavailable, operation)
	}
	if g.EventQueueManager.HasActiveWorkflows() {
		return fmt.Errorf("%w: %s is unavailable during an active workflow", errReviewedMutationUnavailable, operation)
	}

	return nil
}

func reviewedMutationFingerprint(write func(hash.Hash)) string {
	digest := sha256.New()
	write(digest)

	return hex.EncodeToString(digest.Sum(nil))
}

// writeFingerprintf panics only if the in-memory SHA-256 writer violates its
// no-error Write contract; silently dropping bytes would make review stale checks unsafe.
func writeFingerprintf(digest hash.Hash, format string, args ...any) {
	if _, err := fmt.Fprintf(digest, format, args...); err != nil {
		panic(fmt.Sprintf("writing reviewed mutation fingerprint: %v", err))
	}
}
