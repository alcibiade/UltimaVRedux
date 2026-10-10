package debugging

import "errors"

// Default bridge capacities bound work performed and retained between frames.
const (
	DefaultRequestCapacity = 64
	DefaultResultCapacity  = 128
)

// ErrRequestQueueFull indicates that the update thread has not caught up with submissions.
var ErrRequestQueueFull = errors.New("debug request queue is full")

// Bridge transfers immutable requests and results across the network/update boundary.
type Bridge struct {
	requests chan Request
	results  chan Result
}

// NewBridge constructs a bounded bridge. Capacities must be positive.
func NewBridge(requestCapacity, resultCapacity int) *Bridge {
	if requestCapacity <= 0 || resultCapacity <= 0 {
		panic("debug bridge capacities must be positive")
	}

	return &Bridge{
		requests: make(chan Request, requestCapacity),
		results:  make(chan Result, resultCapacity),
	}
}

// NewDefaultBridge constructs a bridge with application defaults.
func NewDefaultBridge() *Bridge {
	return NewBridge(DefaultRequestCapacity, DefaultResultCapacity)
}

// Submit queues a request without blocking the caller.
func (b *Bridge) Submit(request Request) error {
	select {
	case b.requests <- cloneRequest(request):
		return nil
	default:
		return ErrRequestQueueFull
	}
}

// Process executes at most maxRequests queued requests without blocking.
// It stops before execution when no result capacity is available.
func (b *Bridge) Process(executor Executor, maxRequests int) int {
	if executor == nil {
		panic("debug bridge executor is required")
	}
	if maxRequests <= 0 {
		return 0
	}

	processed := 0
	for processed < maxRequests {
		if len(b.results) == cap(b.results) {
			return processed
		}

		select {
		case request := <-b.requests:
			result := executor.ExecuteDebugCommand(request)
			b.results <- cloneResult(result)
			processed++
		default:
			return processed
		}
	}

	return processed
}

// NextResult returns the next result without blocking.
func (b *Bridge) NextResult() (Result, bool) {
	select {
	case result := <-b.results:
		return cloneResult(result), true
	default:
		var empty Result

		return empty, false
	}
}
