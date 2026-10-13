package audio

import "github.com/bradhannah/UltimaVRedux/internal/references"

// createAmbientSound keeps older gain/routing fixtures on the production source
// reconciliation path without requiring a real map tile.
func (asm *AmbientSoundManager) createAmbientSound(
	definition AmbientSoundDefinition, position references.Position, distance float64, key AmbientSoundType,
) {
	asm.mu.Lock()
	defer asm.mu.Unlock()
	asm.reconcileSourcesLocked(map[AmbientSoundType]ambientCandidate{
		key: {definition: definition, position: position, distance: distance, modifier: 1},
	})
	asm.serviceLocked()
}
