package debugging

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingExecutor struct {
	mu       sync.Mutex
	requests []Request
}

type sharedResultExecutor struct {
	invocation *CommandInvocation
	data       json.RawMessage
	changes    []StateChange
}

func (e *sharedResultExecutor) ExecuteDebugCommand(request Request) Result {
	return Result{
		SessionID:              request.SessionID,
		RequestID:              request.ID,
		CommandID:              request.CommandID,
		SceneGeneration:        1,
		Invocation:             e.invocation,
		EffectiveExecutionMode: ExecutionQuery,
		Success:                true,
		Summary:                "shared",
		ErrorCode:              "",
		ErrorMessage:           "",
		Changes:                e.changes,
		Data:                   e.data,
	}
}

func (e *recordingExecutor) ExecuteDebugCommand(request Request) Result {
	e.mu.Lock()
	e.requests = append(e.requests, request)
	e.mu.Unlock()

	return SuccessResult(request, ExecutionQuery, request.CommandID, nil)
}

func (e *recordingExecutor) count() int {
	e.mu.Lock()
	defer e.mu.Unlock()

	return len(e.requests)
}

func (e *recordingExecutor) firstRequest() Request {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.requests[0]
}

func TestBridgeProcessesRequestsInFIFOOrder(t *testing.T) {
	bridge := NewBridge(3, 3)
	executor := &recordingExecutor{}
	for i := 1; i <= 3; i++ {
		err := bridge.Submit(Request{
			SessionID: "test-session",
			ID:        fmt.Sprintf("request-%d", i),
			CommandID: fmt.Sprintf("test.command-%d", i),
		})
		require.NoError(t, err)
	}

	processed := bridge.Process(executor, 3)

	assert.Equal(t, 3, processed)
	for i := 1; i <= 3; i++ {
		result, ok := bridge.NextResult()
		require.True(t, ok)
		assert.Equal(t, fmt.Sprintf("request-%d", i), result.RequestID)
	}
}

func TestBridgeRejectsRequestWhenQueueIsFull(t *testing.T) {
	bridge := NewBridge(1, 1)
	require.NoError(t, bridge.Submit(Request{SessionID: "test-session", ID: "request-1", CommandID: "test.command"}))

	err := bridge.Submit(Request{SessionID: "test-session", ID: "request-2", CommandID: "test.command"})

	assert.ErrorIs(t, err, ErrRequestQueueFull)
}

func TestBridgeDoesNotExecuteWithoutResultCapacity(t *testing.T) {
	bridge := NewBridge(2, 1)
	executor := &recordingExecutor{}
	require.NoError(t, bridge.Submit(Request{SessionID: "test-session", ID: "request-1", CommandID: "test.command"}))
	require.NoError(t, bridge.Submit(Request{SessionID: "test-session", ID: "request-2", CommandID: "test.command"}))

	assert.Equal(t, 1, bridge.Process(executor, 2))
	assert.Equal(t, 1, executor.count())
	assert.Equal(t, 0, bridge.Process(executor, 2))
	assert.Equal(t, 1, executor.count())

	_, ok := bridge.NextResult()
	require.True(t, ok)
	assert.Equal(t, 1, bridge.Process(executor, 2))
	assert.Equal(t, 2, executor.count())
}

func TestBridgeConcurrentSubmission(t *testing.T) {
	const requestCount = 32
	bridge := NewBridge(requestCount, requestCount)
	executor := &recordingExecutor{}
	processed := make(chan int, 1)
	drained := make(chan int, 1)
	go func() {
		total := 0
		for total < requestCount {
			total += bridge.Process(executor, 1)
			runtime.Gosched()
		}
		processed <- total
	}()
	go func() {
		total := 0
		for total < requestCount {
			if _, ok := bridge.NextResult(); ok {
				total++
			}
			runtime.Gosched()
		}
		drained <- total
	}()

	errors := make(chan error, requestCount)
	var wg sync.WaitGroup
	for i := range requestCount {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			errors <- bridge.Submit(Request{
				SessionID: "test-session",
				ID:        fmt.Sprintf("request-%d", index),
				CommandID: "test.command",
			})
		}(i)
	}
	wg.Wait()
	close(errors)
	for err := range errors {
		require.NoError(t, err)
	}

	assert.Equal(t, requestCount, <-processed)
	assert.Equal(t, requestCount, <-drained)
	assert.Equal(t, requestCount, executor.count())
}

func TestBridgeCopiesSubmittedRequestData(t *testing.T) {
	bridge := NewBridge(1, 1)
	executor := &recordingExecutor{}
	argument := json.RawMessage(`"original"`)
	request := Request{
		SessionID: "test-session",
		ID:        "request-1",
		CommandID: "test.command",
		Arguments: map[string]json.RawMessage{"value": argument},
	}
	require.NoError(t, bridge.Submit(request))
	argument[1] = 'X'
	delete(request.Arguments, "value")

	require.Equal(t, 1, bridge.Process(executor, 1))
	recorded := executor.firstRequest()

	assert.JSONEq(t, `"original"`, string(recorded.Arguments["value"]))
}

func TestBridgeCopiesQueuedAndConsumedResultData(t *testing.T) {
	sharedArgument := json.RawMessage(`{"nested":[1,true,"original"]}`)
	sharedInvocation := &CommandInvocation{
		CommandID: "test.command",
		Arguments: map[string]json.RawMessage{"value": sharedArgument},
	}
	sharedData := json.RawMessage(`{"value":"original"}`)
	sharedChanges := []StateChange{{
		Path:   "test.value",
		Before: json.RawMessage(`"before"`),
		After:  json.RawMessage(`"after"`),
	}}
	executor := &sharedResultExecutor{invocation: sharedInvocation, data: sharedData, changes: sharedChanges}
	bridge := NewBridge(2, 2)
	for index := 1; index <= 2; index++ {
		require.NoError(t, bridge.Submit(Request{
			SessionID: "test-session",
			ID:        fmt.Sprintf("request-%d", index),
			CommandID: "test.command",
		}))
	}
	require.Equal(t, 2, bridge.Process(executor, 2))
	sharedArgument[11] = '9'
	delete(sharedInvocation.Arguments, "value")
	sharedData[10] = 'X'
	sharedChanges[0].After[1] = 'X'

	first, ok := bridge.NextResult()
	require.True(t, ok)
	require.NotNil(t, first.Invocation)
	assert.Equal(t, "test.command", first.Invocation.CommandID)
	assert.JSONEq(t, `{"nested":[1,true,"original"]}`, string(first.Invocation.Arguments["value"]))
	assert.JSONEq(t, `{"value":"original"}`, string(first.Data))
	assert.JSONEq(t, `"after"`, string(first.Changes[0].After))
	first.Data[10] = 'Y'
	first.Changes[0].After[1] = 'Y'
	first.Invocation.Arguments["value"][11] = '8'
	first.Invocation.CommandID = "mutated.command"

	second, ok := bridge.NextResult()
	require.True(t, ok)
	require.NotNil(t, second.Invocation)
	assert.Equal(t, "test.command", second.Invocation.CommandID)
	assert.JSONEq(t, `{"nested":[1,true,"original"]}`, string(second.Invocation.Arguments["value"]))
	assert.JSONEq(t, `{"value":"original"}`, string(second.Data))
	assert.JSONEq(t, `"after"`, string(second.Changes[0].After))
}
