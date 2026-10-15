package event_queue

import "github.com/bradhannah/UltimaVRedux/pkg/semantic_input"

// SystemCallbacks interface for input management
type SystemCallbacks interface {
	BlockAllInput(block bool)
	IsAnyKeyPressed() bool
	GetPressedKey() *semantic_input.Key
}

// AnimationDriver interface for frame-by-frame animation updates
// Used by RunScript to drive script executors without circular imports
type AnimationDriver interface {
	Update(deltaMs int)
	IsComplete() bool
}

// DelayedEvent represents a single delayed action with optional chaining
type DelayedEvent struct {
	ExecuteAt int64
	Action    func() error

	// Dynamic chaining support
	NextAction  func() error // Next action to execute after this one
	NextDelay   int64        // Delay in ms before executing NextAction
	IsWaitPoint bool         // True if this event should wait for input before continuing
}

type queueLifecycle uint8

const (
	queueCreated queueLifecycle = iota
	queueDeferred
	queueStarted
	queueFinalizing
	queueFinalized
	queueCoalesced
)

type queueLane uint8

const (
	queueLaneWorkflow queueLane = iota
	queueLaneFeedback
	queueLaneBackground
)

// NamedEventQueue represents an isolated event sequence
type NamedEventQueue struct {
	name                  string
	events                []*DelayedEvent
	continuousEvents      []ContinuousEvent // Events that run every frame
	timeSource            func() int64
	manager               *EventQueueManager
	autoBlockUnblockInput bool
	lastUpdateTime        int64 // Track time for delta calculations
	priority              int   // Execution priority (lower number = higher priority)
	lane                  queueLane

	// Input waiting state
	waitingForInput   bool
	inputWaitCallback func() error
	keyWasPressed     bool // Track if key was pressed last frame for edge detection
	requireKeyRelease bool // Debounce: must wait for all keys to be released before accepting new press

	// Delayed input unblock state (Phase 1F: prevent key bleeding when queue finishes)
	pendingInputUnblock bool        // Queue finished but waiting for all keys to be released before unblocking
	inputReleaseReady   func() bool // Optional owner-specific release guard, never used by WaitForAnyKey.

	// Dynamic chaining state
	pendingNextAction func() error // Action waiting to be scheduled after input
	pendingNextDelay  int64        // Delay for the pending action

	// Cleanup action (from Finally())
	lastAction func() // Cleanup action that always runs at sequence end

	// Termination support (for HandoffTo pattern)
	terminateRequested bool // When true, queue stops after current event and triggers OnComplete

	// Lifecycle callbacks
	onComplete func() // Called when queue finishes naturally or via HandoffTo

	// Lifecycle invariants
	lifecycle       queueLifecycle
	inputBlockOwned bool
	pausedAt        int64
	isPaused        bool
}

// EventQueueManager manages multiple named event queues
//
// IMPORTANT: Single-threaded Assumption
// EventQueueManager assumes single-threaded execution in Ebitengine's game loop.
// Update() is called once per frame on the main thread only.
// No goroutines access EventQueueManager (web profiling goroutine is isolated).
// If multi-threaded access is ever needed, add sync.RWMutex protection.
//
// Workflow and feedback queues are serialized independently. Legacy background
// queues remain concurrent and should not be used for new presentation effects.
type EventQueueManager struct {
	queues          map[string]*NamedEventQueue
	timeSource      func() int64
	systemCallbacks SystemCallbacks
	inputBlockCount int                // Reference counting for overlapping input blocks
	executionDepth  int                // Track callback execution depth (0=not executing, >0=inside callback)
	activeQueue     *NamedEventQueue   // Currently executing queue (only one at a time)
	pendingQueues   []*NamedEventQueue // Queues waiting for execution (sorted by priority)
	deferredStarts  []*NamedEventQueue // Starts accepted inside callbacks and dispatched at a safe boundary
	// Non-input work continues while acknowledgement waits cannot accept keys.
	inputPollingSuspended bool

	// Feedback queues serialize presentation-only effects independently of workflows.
	feedbackQueues        map[string]*NamedEventQueue
	activeFeedbackQueue   *NamedEventQueue
	pendingFeedbackQueues []*NamedEventQueue

	// Background queue support - runs concurrently with main queue for UI-only effects
	// All background queues run in parallel - each processes independently
	activeBackgroundQueues []*NamedEventQueue // All currently executing background queues (concurrent)
	shutdown               bool               // Permanent scene-owned cancellation state
	shutdownAction         func()
}
