package references

// Original Ultima V timing constants
// The original DOS game ran at approximately 18Hz (18.2Hz from IBM PC timer interrupt)
const (
	// OriginalTickMs is the duration of one game tick in the original Ultima V (~55ms)
	// Based on 18Hz tick rate: 1000ms / 18 ≈ 55.56ms
	OriginalTickMs = 55

	// StandardWalkDelayMs is the default walking speed for scripted sequences
	// Approximately 5-6 ticks per step for a deliberate walking pace
	StandardWalkDelayMs = 6 * OriginalTickMs // ~330ms
)
