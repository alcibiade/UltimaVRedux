package workbench

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
)

// ProtocolVersion identifies the JSON envelope contract spoken by the browser and server.
const ProtocolVersion = 1

const (
	messageSessionHello     = "session.hello"
	messageSessionReady     = "session.ready"
	messageCatalogRequest   = "catalog.request"
	messageCatalogSnapshot  = "catalog.snapshot"
	messageCommandExecute   = "command.execute"
	messageCommandAccepted  = "command.accepted"
	messageCommandCompleted = "command.completed"
	messageCommandFailed    = "command.failed"
	messageCommandAck       = "command.ack"
	messageOperationStatus  = "operation.status"
	messageProtocolError    = "protocol.error"
)

var (
	errInvalidEnvelope = errors.New("invalid debug workbench envelope")
	errInvalidPayload  = errors.New("invalid debug workbench payload")
)

type envelope struct {
	Version int             `json:"version"`
	ID      string          `json:"id,omitempty"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type sessionHelloPayload struct {
	SessionID string `json:"sessionId,omitempty"`
}

type sessionReadyPayload struct {
	SessionID       string `json:"sessionId"`
	SceneGeneration uint64 `json:"sceneGeneration"`
	Resumed         bool   `json:"resumed"`
}

type catalogSnapshotPayload struct {
	SceneGeneration uint64                        `json:"sceneGeneration"`
	OptionsRevision uint64                        `json:"optionsRevision"`
	Commands        []debugging.CommandDefinition `json:"commands"`
	OptionSets      []debugging.OptionSet         `json:"optionSets,omitempty"`
}

type commandExecutePayload struct {
	CommandID               string                     `json:"commandId"`
	ExpectedSceneGeneration uint64                     `json:"expectedSceneGeneration,omitempty"`
	ExpectedOptionsRevision uint64                     `json:"expectedOptionsRevision,omitempty"`
	Arguments               map[string]json.RawMessage `json:"arguments,omitempty"`
}

type commandAcceptedPayload struct {
	RequestID string `json:"requestId"`
	CommandID string `json:"commandId"`
}

type commandAckPayload struct {
	RequestID string `json:"requestId"`
}

type operationStatusPayload struct {
	RequestID string `json:"requestId"`
}

type protocolErrorPayload struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func newEnvelope(requestID, messageType string, payload any) envelope {
	raw, err := json.Marshal(payload)
	if err != nil {
		panic(fmt.Sprintf("marshalling debug workbench message %q: %v", messageType, err))
	}

	return envelope{
		Version: ProtocolVersion,
		ID:      requestID,
		Type:    messageType,
		Payload: raw,
	}
}

func decodePayload(message envelope, target any) error {
	if len(message.Payload) == 0 {
		return fmt.Errorf("%w: message %q requires a payload", errInvalidPayload, message.Type)
	}
	if err := json.Unmarshal(message.Payload, target); err != nil {
		return fmt.Errorf("%w for message %q: %w", errInvalidPayload, message.Type, err)
	}

	return nil
}

func validateEnvelope(message envelope) error {
	if message.Version != ProtocolVersion {
		return fmt.Errorf("%w: unsupported protocol version %d", errInvalidEnvelope, message.Version)
	}
	if message.Type == "" {
		return fmt.Errorf("%w: message type is required", errInvalidEnvelope)
	}

	return nil
}
