package map_units

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bradhannah/UltimaVRedux/internal/references"
)

func TestSetOverriddenAITypeAssignsGenericProvenance(t *testing.T) {
	tests := []struct {
		name       string
		aiType     references.AiType
		provenance references.AIOverrideProvenance
	}{
		{name: "hostility", aiType: references.ExtortOrAttackOrFollow, provenance: references.AIOverrideProvenanceGuardHostility},
		{name: "unlimited hostility", aiType: references.DrudgeWorthThing, provenance: references.AIOverrideProvenanceGuardHostility},
		{name: "flee", aiType: references.FleeFromAvatar, provenance: references.AIOverrideProvenanceGenericFlee},
		{name: "other", aiType: references.FollowAroundAndBeAnnoyingThenNeverSeeAgain, provenance: references.AIOverrideProvenanceNone},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			npc := NewNPCFriendly(references.NPCReference{}, 1)

			npc.MapUnitDetails().SetOverriddenAiType(test.aiType)

			assert.Equal(t, test.aiType, npc.MapUnitDetails().GetOverriddenAiType())
			assert.Equal(t, test.provenance, npc.MapUnitDetails().AIOverrideProvenance())
		})
	}
}

func TestSetLawAIOverridePreservesAutomaticProvenanceAndAssignsGenericCause(t *testing.T) {
	tests := []struct {
		name          string
		initialAI     references.AiType
		initialCause  references.AIOverrideProvenance
		lawAI         references.AiType
		expectedCause references.AIOverrideProvenance
	}{
		{
			name: "Astaroth cause survives law flee", initialAI: references.DrudgeWorthThing,
			initialCause: references.AIOverrideProvenanceAutomaticAstaroth, lawAI: references.FleeFromAvatar,
			expectedCause: references.AIOverrideProvenanceAutomaticAstaroth,
		},
		{
			name: "Nosfentor cause survives law hostility", initialAI: references.FleeFromAvatar,
			initialCause: references.AIOverrideProvenanceAutomaticNosfentor, lawAI: references.DrudgeWorthThing,
			expectedCause: references.AIOverrideProvenanceAutomaticNosfentor,
		},
		{
			name: "generic hostility records guard cause", initialAI: references.Unset,
			initialCause: references.AIOverrideProvenanceNone, lawAI: references.DrudgeWorthThing,
			expectedCause: references.AIOverrideProvenanceGuardHostility,
		},
		{
			name: "Astaroth cause survives law hostility", initialAI: references.FleeFromAvatar,
			initialCause: references.AIOverrideProvenanceAutomaticAstaroth, lawAI: references.DrudgeWorthThing,
			expectedCause: references.AIOverrideProvenanceAutomaticAstaroth,
		},
		{
			name: "existing action6 hostility remains supported", initialAI: references.Unset,
			initialCause: references.AIOverrideProvenanceNone, lawAI: references.ExtortOrAttackOrFollow,
			expectedCause: references.AIOverrideProvenanceGuardHostility,
		},
		{
			name: "generic flee records flee cause", initialAI: references.Unset,
			initialCause: references.AIOverrideProvenanceNone, lawAI: references.FleeFromAvatar,
			expectedCause: references.AIOverrideProvenanceGenericFlee,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			npc := NewNPCFriendly(references.NPCReference{}, 1)
			details := npc.MapUnitDetails()
			details.SetAIOverride(test.initialAI, test.initialCause)

			details.SetLawAIOverride(test.lawAI)

			assert.Equal(t, test.lawAI, details.GetOverriddenAiType())
			assert.Equal(t, test.expectedCause, details.AIOverrideProvenance())
		})
	}
}
