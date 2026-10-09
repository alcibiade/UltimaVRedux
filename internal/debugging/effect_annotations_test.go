package debugging

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDurationEffectInspectorDescriptions(t *testing.T) {
	assert.Equal(t, "+3 defense bonus active", DurationEffectDescription("P"))
	assert.Equal(t, "Combat magic and magical enemy abilities suppressed for both sides", DurationEffectDescription("N"))
	assert.Equal(t, "Monsters lose half their combat actions; travel takes half time", DurationEffectDescription("Q"))
	assert.Empty(t, DurationEffectDescription(""))
}
