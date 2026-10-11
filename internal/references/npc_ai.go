package references

// AiType identifies an authored schedule action or a Redux override behavior.
type AiType int

// AIOverrideProvenance identifies why a map unit's scheduled AI was replaced.
// It is separate from AiType because law enforcement and Shadowlord influence
// intentionally reuse the same executable AI states.
type AIOverrideProvenance string

// AI override provenance values identify the authoritative cause of a replacement.
const (
	AIOverrideProvenanceNone               AIOverrideProvenance = "none"
	AIOverrideProvenanceAutomaticAstaroth  AIOverrideProvenance = "automatic-astaroth"
	AIOverrideProvenanceAutomaticNosfentor AIOverrideProvenance = "automatic-nosfentor"
	AIOverrideProvenanceGuardHostility     AIOverrideProvenance = "guard-hostility"
	AIOverrideProvenanceGenericFlee        AIOverrideProvenance = "generic-flee"
)

// IsValid reports whether provenance belongs to the native save/runtime contract.
func (provenance AIOverrideProvenance) IsValid() bool {
	switch provenance {
	case AIOverrideProvenanceNone,
		AIOverrideProvenanceAutomaticAstaroth,
		AIOverrideProvenanceAutomaticNosfentor,
		AIOverrideProvenanceGuardHostility,
		AIOverrideProvenanceGenericFlee:
		return true
	default:
		return false
	}
}

const (
	// Unset not currently set - used for overriding AI
	Unset AiType = -1

	// Fixed stay put
	Fixed AiType = 0

	// Wander wander a small radius
	Wander = 1

	// BigWander wander a wide radius
	BigWander = 2

	// ChildRunAway children are playful and run away from you
	ChildRunAway = 3

	// CustomAi special tlk for merchants
	CustomAi = 4

	// Wander3 and Talk3 name the original schedule actions used by the
	// one-shot TALK3 to WANDER3 transition.
	Wander3 = Wander
	// Talk3 identifies the original one-shot merchant schedule action.
	Talk3 = CustomAi

	// TalkAlways approaches and talks without action 4's schedule-anchor range gate.
	TalkAlways = 5

	// ExtortOrAttackOrFollow you better pay the guard, or they will attack!
	ExtortOrAttackOrFollow = 6

	// DrudgeWorthThing he's a jerk - he will attack at first opportunity, and get as close to you as possible
	DrudgeWorthThing = 7

	// FleeFromAvatar civilians flee from Avatar when guards are called (50% chance)
	// This is a temporary override state that persists until map reload
	FleeFromAvatar = 8

	// HorseWander horses wander if they aren't tied up
	HorseWander = 100

	// FollowAroundAndBeAnnoyingThenNeverSeeAgain people who are freed walk close to you and are annoying - but won't talk
	FollowAroundAndBeAnnoyingThenNeverSeeAgain = 101

	// SmallWanderWantsToChat they will wander in a small radius, but if they are next to you, then they will want to chat
	SmallWanderWantsToChat = 102

	// Begging beggars wander, and when next to Avatar they want to talk
	Begging = 103

	// GenericExtortingGuard They generally ask for 30-60gp tribute
	GenericExtortingGuard = 104

	// HalfYourGoldExtortingGuard These jerks want half your gold! Generally found in Minoc
	HalfYourGoldExtortingGuard = 105

	// MerchantBuyingSelling Merchant that doesn't move
	MerchantBuyingSelling = 106

	// MerchantBuyingSellingCustom I THINK this is a merchant that sells, but also wanders a bit
	MerchantBuyingSellingCustom = 107

	// MerchantBuyingSellingWander Merchants that wander their own store, but still sell
	MerchantBuyingSellingWander = 108

	// FixedExceptAttackWhenIsWantedByThePoPo They are Fixed unless wanted by the popo, at which time they seek and attack
	FixedExceptAttackWhenIsWantedByThePoPo = 109

	// BlackthornGuardWander Blackthorn guards will check in for password, with a small wander
	BlackthornGuardWander = 110

	// BlackthornGuardFixed Blackthorn guard stays put, but will ask password if prompted
	BlackthornGuardFixed = 111
	// StoneGargoyleTrigger waits for nearby activation.
	StoneGargoyleTrigger = 112
)

// IsValid reports whether the AI value belongs to the runtime contract.
func (aiType AiType) IsValid() bool {
	switch aiType {
	case Unset,
		Fixed,
		Wander,
		BigWander,
		ChildRunAway,
		CustomAi,
		TalkAlways,
		ExtortOrAttackOrFollow,
		DrudgeWorthThing,
		FleeFromAvatar,
		HorseWander,
		FollowAroundAndBeAnnoyingThenNeverSeeAgain,
		SmallWanderWantsToChat,
		Begging,
		GenericExtortingGuard,
		HalfYourGoldExtortingGuard,
		MerchantBuyingSelling,
		MerchantBuyingSellingCustom,
		MerchantBuyingSellingWander,
		FixedExceptAttackWhenIsWantedByThePoPo,
		BlackthornGuardWander,
		BlackthornGuardFixed,
		StoneGargoyleTrigger:
		return true
	default:
		return false
	}
}
