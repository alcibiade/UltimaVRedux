package main

// TalkType represents the type of talk interaction to perform.
// Used by StartTalkWorkflow to branch to appropriate dialog/effect sequence.
type TalkType int

const (
	// TalkTypeNoNPC means no NPC found in the direction
	TalkTypeNoNPC TalkType = iota

	// TalkTypeShadowlordHostile means NPC is under Astaroth (Hatred) influence
	// NPC attacks party and then flees
	TalkTypeShadowlordHostile

	// TalkTypeShadowlordFlee means NPC is under Nosfentor (Cowardice) influence
	// NPC cowers and refuses to talk
	TalkTypeShadowlordFlee

	// TalkTypeGuard means NPC is a guard with dialogue
	// Routes to arrest or extortion system based on criminal status
	TalkTypeGuard

	// TalkTypeSleeping means NPC is on a bed tile (sleeping)
	// Shows "Zzzzzz..." for a sleeping interlocutor (ref:TALK-0005).
	TalkTypeSleeping

	// TalkTypeMerchant means one of the exact production service codes 0x81-0x88.
	TalkTypeMerchant

	// TalkTypeNormal means normal NPC conversation
	// Opens LinearTalkDialog for full conversation
	TalkTypeNormal

	// TalkTypeNonFriendly means NPC exists but isn't friendly
	// Shows "Can't talk to that!" message
	TalkTypeNonFriendly
)
