package references

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInQuasXenReagentMask(t *testing.T) {
	want := uint16(1<<SulfurAsh | 1<<Ginseng | 1<<SpiderSilk | 1<<BloodMoss | 1<<MandrakeRoot)

	mask := GetSpellReagents(InQuasXen)

	assert.Equal(t, want, mask)
	assert.Zero(t, mask&(1<<NightShade))
}
