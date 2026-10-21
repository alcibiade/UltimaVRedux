// Package conversation runs semantic NPC conversations and defines the response protocol the talk dialog consumes.
package conversation

// Engine is the response protocol consumed by the conversation dialog.
type Engine interface {
	Start(actorID int) *ConversationResponse
	ProcessInput(input string) *ConversationResponse
	IsActive() bool
	Stop()
	ExtractNPCName() string
}
