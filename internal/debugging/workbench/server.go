// Package workbench serves the local browser debug workbench.
package workbench

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"

	"github.com/bradhannah/UltimaVRedux/internal/debugging"
	"github.com/bradhannah/UltimaVRedux/internal/debugging/webui"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
	"github.com/bradhannah/UltimaVRedux/internal/sprites"
)

// Default server limits keep the workbench local and bound resource use.
const (
	DefaultPort              = 8585
	DefaultAddress           = "127.0.0.1:8585"
	defaultOutboundCapacity  = 128
	defaultMaxMessageBytes   = 64 * 1024
	defaultResultPoll        = 5 * time.Millisecond
	defaultReadHeaderTimeout = 5 * time.Second
	maxSessions              = 16
	maxSessionOperations     = 128
	sessionIDBytes           = 16
	tokenBytes               = 32
	defaultLogLimit          = 200
	maximumLogLimit          = 256
	logAuthHeader            = "X-U5r-Log-Token"
)

var (
	// ErrInvalidConfig indicates that server configuration violates safety constraints.
	ErrInvalidConfig = errors.New("invalid debug workbench configuration")
	// ErrAlreadyStarted indicates that Start was called more than once.
	ErrAlreadyStarted = errors.New("debug workbench server already started")
	// ErrNotStarted indicates that shutdown was requested before startup.
	ErrNotStarted = errors.New("debug workbench server not started")
	// ErrOutboundQueueFull indicates that a slow client exhausted its bounded queue.
	ErrOutboundQueueFull = errors.New("debug workbench outbound queue is full")
	// ErrControllerConnected indicates that the initial single-controller limit was reached.
	ErrControllerConnected = errors.New("a debug workbench controller is already connected")
	// ErrServerStopping indicates that shutdown has closed the connection admission gate.
	ErrServerStopping = errors.New("debug workbench server is stopping")
)

const (
	errorCodeInvalidPayload  = "invalid_payload"
	errorCodeSessionRequired = "session_required"
	messageSessionRequired   = "session.hello must be sent first"
)

// ControlPlane is the concurrency-safe application boundary used by the server.
type ControlPlane interface {
	SubmitDebugRequest(request debugging.Request) error
	NextDebugResult() (debugging.Result, bool)
	DebugDiscoverySnapshot() debugging.DiscoverySnapshot
	DebugSceneGeneration() uint64
}

// Config controls the local workbench server without exposing gameplay settings.
type Config struct {
	Address          string
	OutboundCapacity int
	MaxMessageBytes  int64
	ResultPoll       time.Duration
}

// DefaultConfig returns conservative loopback-only server settings.
func DefaultConfig() Config {
	return Config{
		Address:          DefaultAddress,
		OutboundCapacity: defaultOutboundCapacity,
		MaxMessageBytes:  defaultMaxMessageBytes,
		ResultPoll:       defaultResultPoll,
	}
}

type operationRecord struct {
	request debugging.Request
	result  *debugging.Result
}

type sessionState struct {
	id         string
	operations map[string]*operationRecord
	lastUsed   time.Time
}

type clientConnection struct {
	server    *Server
	conn      *websocket.Conn
	cancel    context.CancelFunc
	outbound  chan envelope
	sessionID string
}

// Server owns the loopback listener, active browser connection, and resumable operation records.
type Server struct {
	control  ControlPlane
	config   Config
	assets   fs.FS
	token    string
	logToken string

	done     chan struct{}
	stopOnce sync.Once

	mu         sync.Mutex
	listener   net.Listener
	httpServer *http.Server
	active     *clientConnection
	sessions   map[string]*sessionState
	stopped    bool
	serveErr   chan error
	dispatchWG sync.WaitGroup
	clientWG   sync.WaitGroup
}

// NewServer validates configuration and creates a stopped workbench server.
func NewServer(control ControlPlane, config Config) (*Server, error) {
	if control == nil {
		return nil, fmt.Errorf("%w: control plane is required", ErrInvalidConfig)
	}
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	token, err := randomID(tokenBytes)
	if err != nil {
		return nil, fmt.Errorf("generating debug workbench token: %w", err)
	}
	logToken, err := randomID(tokenBytes)
	if err != nil {
		return nil, fmt.Errorf("generating debug log reader token: %w", err)
	}

	return &Server{
		control:    control,
		config:     config,
		assets:     webui.Assets(),
		token:      token,
		logToken:   logToken,
		done:       make(chan struct{}),
		stopOnce:   sync.Once{},
		mu:         sync.Mutex{},
		listener:   nil,
		httpServer: nil,
		active:     nil,
		sessions:   make(map[string]*sessionState),
		stopped:    false,
		serveErr:   make(chan error, 1),
		dispatchWG: sync.WaitGroup{},
		clientWG:   sync.WaitGroup{},
	}, nil
}

// Start binds synchronously and starts HTTP and result-dispatch goroutines.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return ErrServerStopping
	}
	if s.listener != nil {
		return ErrAlreadyStarted
	}

	listenConfig := net.ListenConfig{}
	listener, err := listenConfig.Listen(context.Background(), "tcp", s.config.Address)
	if err != nil {
		return fmt.Errorf("binding debug workbench to %s: %w", s.config.Address, err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.serveIndex)
	mux.HandleFunc("GET /logs", s.serveIndex)
	mux.HandleFunc("GET /api/logs", s.serveLogEntries)
	mux.HandleFunc("GET /assets/", s.serveAsset)
	mux.HandleFunc("GET /api/tile-atlas.png", s.serveTileAtlas)
	mux.HandleFunc("GET /healthz", s.serveHealth)
	mux.HandleFunc("GET /ws", s.serveWebSocket)
	s.listener = listener
	s.httpServer = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: defaultReadHeaderTimeout,
	}

	s.dispatchWG.Add(1)
	go s.dispatchResults()
	go s.serve()

	return nil
}

// URL returns the bound browser URL after Start.
func (s *Server) URL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return ""
	}

	return "http://" + s.listener.Addr().String()
}

// Errors reports unexpected HTTP serving failures.
func (s *Server) Errors() <-chan error {
	return s.serveErr
}

// Shutdown stops active connections, HTTP serving, and result dispatch.
func (s *Server) Shutdown(ctx context.Context) error {
	httpServer := s.beginShutdown()
	if httpServer == nil {
		return ErrNotStarted
	}

	err := httpServer.Shutdown(ctx)
	waitErr := s.waitForBackgroundWork(ctx)
	if err != nil || waitErr != nil {
		return fmt.Errorf("shutting down debug workbench server: %w", errors.Join(err, waitErr))
	}

	return nil
}

func (s *Server) waitForBackgroundWork(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		s.dispatchWG.Wait()
		s.clientWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return fmt.Errorf("waiting for debug workbench background work: %w", ctx.Err())
	}
}

func (s *Server) serve() {
	defer close(s.serveErr)
	err := s.httpServer.Serve(s.listener)
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		select {
		case s.serveErr <- fmt.Errorf("serving debug workbench: %w", err):
		default:
		}
		s.beginShutdown()
	}
}

func (s *Server) beginShutdown() *http.Server {
	s.mu.Lock()
	s.stopped = true
	active := s.active
	httpServer := s.httpServer
	s.mu.Unlock()
	if active != nil {
		active.cancel()
	}
	s.stopOnce.Do(func() {
		close(s.done)
	})

	return httpServer
}

func (s *Server) serveIndex(response http.ResponseWriter, request *http.Request) {
	if !s.validRequestHost(request) {
		http.Error(response, "invalid host", http.StatusMisdirectedRequest)

		return
	}
	if request.URL.Path != "/" && request.URL.Path != "/logs" {
		http.NotFound(response, request)

		return
	}
	index, err := fs.ReadFile(s.assets, "index.html")
	if err != nil {
		http.Error(response, "workbench assets unavailable", http.StatusInternalServerError)

		return
	}
	token := s.token
	if request.URL.Path == "/logs" {
		token = s.logToken
	}
	page := strings.Replace(string(index), "__U5R_WORKBENCH_TOKEN__", token, 1)
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set(
		"Content-Security-Policy",
		fmt.Sprintf("default-src 'self'; connect-src 'self' ws://%s; frame-ancestors 'none'; base-uri 'none'", request.Host),
	)
	response.Header().Set("Content-Type", "text/html; charset=utf-8")
	response.Header().Set("Referrer-Policy", "no-referrer")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	if _, err := response.Write([]byte(page)); err != nil {
		logging.Debugf(logging.CategorySystem, "Workbench page response closed: %v", err)
	}
}

func (s *Server) serveAsset(response http.ResponseWriter, request *http.Request) {
	if !s.validRequestHost(request) {
		http.Error(response, "invalid host", http.StatusMisdirectedRequest)

		return
	}
	path := strings.TrimPrefix(request.URL.Path, "/")
	if path == "" || strings.Contains(path, "..") {
		http.NotFound(response, request)

		return
	}
	content, err := fs.ReadFile(s.assets, path)
	if err != nil {
		http.NotFound(response, request)

		return
	}
	response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	http.ServeContent(response, request, path, time.Time{}, bytes.NewReader(content))
}

func (s *Server) serveHealth(response http.ResponseWriter, request *http.Request) {
	if !s.validRequestHost(request) {
		http.Error(response, "invalid host", http.StatusMisdirectedRequest)

		return
	}
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Content-Type", "application/json")
	if _, err := response.Write([]byte(`{"status":"ok"}`)); err != nil {
		logging.Debugf(logging.CategorySystem, "Workbench health response closed: %v", err)
	}
}

func (s *Server) serveTileAtlas(response http.ResponseWriter, request *http.Request) {
	if !s.authorizeWorkbenchAsset(request) {
		http.Error(response, "unauthorized", http.StatusUnauthorized)

		return
	}
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Content-Type", "image/png")
	response.Header().Set("Referrer-Policy", "no-referrer")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	if _, err := response.Write(sprites.TileAtlasPNG()); err != nil {
		logging.Debugf(logging.CategorySystem, "Workbench tile atlas response closed: %v", err)
	}
}

func (s *Server) serveLogEntries(response http.ResponseWriter, request *http.Request) {
	if !s.authorizeLogRead(request) {
		http.Error(response, "unauthorized", http.StatusUnauthorized)

		return
	}
	after, limit, err := parseLogQuery(request)
	if err != nil {
		http.Error(response, err.Error(), http.StatusBadRequest)

		return
	}
	response.Header().Set("Cache-Control", "no-store")
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.Header().Set("Referrer-Policy", "no-referrer")
	response.Header().Set("X-Content-Type-Options", "nosniff")
	if err := json.NewEncoder(response).Encode(logging.GetGlobalLogger().EntriesAfter(after, limit)); err != nil {
		return
	}
}

func (s *Server) authorizeLogRead(request *http.Request) bool {
	if !s.validRequestHost(request) || len(request.Header.Values(logAuthHeader)) != 1 {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(request.Header.Get(logAuthHeader)), []byte(s.logToken)) != 1 {
		return false
	}
	if origin := request.Header.Get("Origin"); origin != "" && origin != "http://"+request.Host {
		return false
	}
	if site := request.Header.Get("Sec-Fetch-Site"); site != "" && site != "same-origin" {
		return false
	}

	return true
}

func (s *Server) authorizeWorkbenchAsset(request *http.Request) bool {
	if !s.validRequestHost(request) {
		return false
	}
	if subtle.ConstantTimeCompare([]byte(request.URL.Query().Get("token")), []byte(s.token)) != 1 {
		return false
	}
	if origin := request.Header.Get("Origin"); origin != "" && origin != "http://"+request.Host {
		return false
	}

	return request.Header.Get("Sec-Fetch-Site") == "" || request.Header.Get("Sec-Fetch-Site") == "same-origin"
}

func parseLogQuery(request *http.Request) (uint64, int, error) {
	query := request.URL.Query()
	for key, values := range query {
		if (key != "after" && key != "limit") || len(values) != 1 {
			return 0, 0, errors.New("invalid log query")
		}
	}
	after := uint64(0)
	if value := query.Get("after"); value != "" {
		parsed, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			return 0, 0, errors.New("invalid after cursor")
		}
		after = parsed
	}
	limit := defaultLogLimit
	if value := query.Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > maximumLogLimit {
			return 0, 0, errors.New("invalid log limit")
		}
		limit = parsed
	}

	return after, limit, nil
}

func (s *Server) serveWebSocket(response http.ResponseWriter, request *http.Request) {
	if !s.authorizeWebSocket(request) {
		http.Error(response, "unauthorized", http.StatusUnauthorized)

		return
	}

	clientCtx, clientCancel := context.WithCancel(request.Context())
	client := &clientConnection{
		server:    s,
		conn:      nil,
		cancel:    clientCancel,
		outbound:  make(chan envelope, s.config.OutboundCapacity),
		sessionID: "",
	}
	if err := s.acquireController(client); err != nil {
		clientCancel()
		http.Error(response, err.Error(), http.StatusConflict)

		return
	}
	defer s.clientWG.Done()

	conn, err := websocket.Accept(response, request, nil)
	if err != nil {
		s.releaseController(client)
		clientCancel()

		return
	}
	client.conn = conn
	conn.SetReadLimit(s.config.MaxMessageBytes)

	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		client.writeLoop(clientCtx)
	}()
	client.readLoop(clientCtx)
	clientCancel()
	if err := conn.CloseNow(); err != nil {
		logging.Debugf(logging.CategorySystem, "Closing Workbench connection: %v", err)
	}
	<-writerDone
	s.releaseController(client)
}

func (s *Server) authorizeWebSocket(request *http.Request) bool {
	if !s.validRequestHost(request) {
		return false
	}
	providedToken := request.URL.Query().Get("token")
	if subtle.ConstantTimeCompare([]byte(providedToken), []byte(s.token)) != 1 {
		return false
	}
	if request.Header.Get("Origin") != "http://"+request.Host {
		return false
	}

	return true
}

func (s *Server) validRequestHost(request *http.Request) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.listener != nil && request.Host == s.listener.Addr().String()
}

func (s *Server) acquireController(client *clientConnection) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return ErrServerStopping
	}
	if s.active != nil {
		return ErrControllerConnected
	}
	s.active = client
	s.clientWG.Add(1)

	return nil
}

func (s *Server) releaseController(client *clientConnection) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.active == client {
		s.active = nil
	}
}

func (c *clientConnection) readLoop(ctx context.Context) {
	for {
		var message envelope
		if err := wsjson.Read(ctx, c.conn, &message); err != nil {
			return
		}
		if err := validateEnvelope(message); err != nil {
			c.send(newEnvelope(message.ID, messageProtocolError, protocolErrorPayload{
				Code:    "invalid_envelope",
				Message: err.Error(),
			}))

			continue
		}
		c.handleMessage(message)
	}
}

func (c *clientConnection) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case message := <-c.outbound:
			if err := wsjson.Write(ctx, c.conn, message); err != nil {
				c.cancel()

				return
			}
		}
	}
}

func (c *clientConnection) enqueue(message envelope) error {
	select {
	case c.outbound <- message:
		return nil
	default:
		c.cancel()

		return ErrOutboundQueueFull
	}
}

// send closes an overloaded connection through enqueue's cancellation path.
// Command results remain retained by session and can be replayed on reconnect.
func (c *clientConnection) send(message envelope) {
	if err := c.enqueue(message); err != nil {
		logging.Debugf(logging.CategorySystem, "Workbench client disconnected before outbound delivery: %v", err)
	}
}

func (c *clientConnection) handleMessage(message envelope) {
	switch message.Type {
	case messageSessionHello:
		c.handleSessionHello(message)
	case messageCatalogRequest:
		c.handleCatalogRequest(message)
	case messageCommandExecute:
		c.handleCommandExecute(message)
	case messageCommandAck:
		c.handleCommandAck(message)
	case messageOperationStatus:
		c.handleOperationStatus(message)
	default:
		c.send(newEnvelope(message.ID, messageProtocolError, protocolErrorPayload{
			Code:    "unknown_message_type",
			Message: "unsupported message type " + strconv.Quote(message.Type),
		}))
	}
}

func (c *clientConnection) requireSession(message envelope) (*sessionState, bool) {
	c.server.mu.Lock()
	defer c.server.mu.Unlock()
	if c.sessionID == "" {
		c.send(newEnvelope(message.ID, messageProtocolError, protocolErrorPayload{
			Code:    errorCodeSessionRequired,
			Message: messageSessionRequired,
		}))

		return nil, false
	}

	return c.server.sessions[c.sessionID], true
}

func (c *clientConnection) handleSessionHello(message envelope) {
	payload, ok := c.decodeSessionHello(message)
	if !ok {
		return
	}

	c.server.mu.Lock()
	if c.sessionID != "" {
		c.server.mu.Unlock()
		c.send(newEnvelope(message.ID, messageProtocolError, protocolErrorPayload{
			Code:    "session_already_ready",
			Message: "this connection already has a session",
		}))

		return
	}
	session, resumed := c.server.sessions[payload.SessionID]
	if !resumed {
		if len(c.server.sessions) >= maxSessions && !c.server.evictIdleSessionLocked() {
			c.server.mu.Unlock()
			c.send(newEnvelope(message.ID, messageProtocolError, protocolErrorPayload{
				Code:    "session_limit",
				Message: "too many retained workbench sessions",
			}))

			return
		}
		sessionID, err := randomID(sessionIDBytes)
		if err != nil {
			c.server.mu.Unlock()
			panic("generating debug workbench session ID: " + err.Error())
		}
		session = &sessionState{
			id:         sessionID,
			operations: make(map[string]*operationRecord),
			lastUsed:   time.Now(),
		}
		c.server.sessions[sessionID] = session
	}
	session.lastUsed = time.Now()
	c.sessionID = session.id
	pending := make([]envelope, 0, len(session.operations)+1)
	pending = append(pending, newEnvelope(message.ID, messageSessionReady, sessionReadyPayload{
		SessionID:       session.id,
		SceneGeneration: c.server.control.DebugSceneGeneration(),
		Resumed:         resumed,
	}))
	for _, operation := range session.operations {
		pending = append(pending, operationEnvelope(operation))
	}
	c.server.mu.Unlock()

	for _, outgoing := range pending {
		if err := c.enqueue(outgoing); err != nil {
			return
		}
	}
}

func (c *clientConnection) decodeSessionHello(message envelope) (sessionHelloPayload, bool) {
	var payload sessionHelloPayload
	if len(message.Payload) > 0 {
		if err := json.Unmarshal(message.Payload, &payload); err != nil {
			c.send(newEnvelope(message.ID, messageProtocolError, protocolErrorPayload{
				Code:    errorCodeInvalidPayload,
				Message: err.Error(),
			}))

			var empty sessionHelloPayload

			return empty, false
		}
	}

	return payload, true
}

func (c *clientConnection) handleCatalogRequest(message envelope) {
	if _, ok := c.requireSession(message); !ok {
		return
	}
	discovery := c.server.control.DebugDiscoverySnapshot()
	c.send(newEnvelope(message.ID, messageCatalogSnapshot, catalogSnapshotPayload{
		SceneGeneration: discovery.SceneGeneration,
		OptionsRevision: discovery.OptionsRevision,
		Commands:        discovery.Commands,
		OptionSets:      discovery.OptionSets,
	}))
}

func (c *clientConnection) handleCommandExecute(message envelope) {
	payload, ok := c.decodeCommandExecute(message)
	if !ok {
		return
	}

	c.server.mu.Lock()
	if c.sessionID == "" {
		c.server.mu.Unlock()
		c.send(newEnvelope(message.ID, messageProtocolError, protocolErrorPayload{
			Code:    errorCodeSessionRequired,
			Message: messageSessionRequired,
		}))

		return
	}
	session := c.server.sessions[c.sessionID]
	if existing, found := session.operations[message.ID]; found {
		outgoing := operationEnvelope(existing)
		c.server.mu.Unlock()
		c.send(outgoing)

		return
	}
	if len(session.operations) >= maxSessionOperations {
		c.server.mu.Unlock()
		c.send(newEnvelope(message.ID, messageCommandFailed, debugging.FailureResult(
			debugging.Request{
				SessionID:               session.id,
				ID:                      message.ID,
				CommandID:               payload.CommandID,
				ExpectedSceneGeneration: 0,
				Arguments:               nil,
			},
			"session_capacity",
			"acknowledge completed commands before submitting more",
		)))

		return
	}
	request := debugging.Request{
		SessionID:               session.id,
		ID:                      message.ID,
		CommandID:               payload.CommandID,
		ExpectedSceneGeneration: payload.ExpectedSceneGeneration,
		ExpectedOptionsRevision: payload.ExpectedOptionsRevision,
		Arguments:               payload.Arguments,
	}
	record := &operationRecord{request: request, result: nil}
	session.operations[message.ID] = record
	if err := c.server.control.SubmitDebugRequest(request); err != nil {
		result := debugging.FailureResult(request, "request_queue_full", err.Error())
		record.result = &result
		outgoing := operationEnvelope(record)
		c.send(outgoing)
		c.server.mu.Unlock()

		return
	}
	outgoing := operationEnvelope(record)
	c.send(outgoing)
	c.server.mu.Unlock()
}

func (c *clientConnection) decodeCommandExecute(message envelope) (commandExecutePayload, bool) {
	if message.ID == "" {
		c.send(newEnvelope(message.ID, messageProtocolError, protocolErrorPayload{
			Code:    "request_id_required",
			Message: "command.execute requires an envelope ID",
		}))

		var empty commandExecutePayload

		return empty, false
	}
	var payload commandExecutePayload
	if err := decodePayload(message, &payload); err != nil {
		c.send(newEnvelope(message.ID, messageProtocolError, protocolErrorPayload{
			Code:    errorCodeInvalidPayload,
			Message: err.Error(),
		}))

		var empty commandExecutePayload

		return empty, false
	}

	return payload, true
}

func (c *clientConnection) handleCommandAck(message envelope) {
	var payload commandAckPayload
	if err := decodePayload(message, &payload); err != nil {
		c.send(newEnvelope(message.ID, messageProtocolError, protocolErrorPayload{
			Code:    errorCodeInvalidPayload,
			Message: err.Error(),
		}))

		return
	}

	c.server.mu.Lock()
	defer c.server.mu.Unlock()
	session := c.server.sessions[c.sessionID]
	if session == nil {
		return
	}
	operation := session.operations[payload.RequestID]
	if operation != nil && operation.result != nil {
		delete(session.operations, payload.RequestID)
		session.lastUsed = time.Now()
	}
}

func (c *clientConnection) handleOperationStatus(message envelope) {
	var payload operationStatusPayload
	if err := decodePayload(message, &payload); err != nil {
		c.send(newEnvelope(message.ID, messageProtocolError, protocolErrorPayload{
			Code:    errorCodeInvalidPayload,
			Message: err.Error(),
		}))

		return
	}

	c.server.mu.Lock()
	session := c.server.sessions[c.sessionID]
	if session == nil {
		c.server.mu.Unlock()
		c.send(newEnvelope(message.ID, messageProtocolError, protocolErrorPayload{
			Code:    errorCodeSessionRequired,
			Message: messageSessionRequired,
		}))

		return
	}
	operation := session.operations[payload.RequestID]
	if operation == nil {
		c.server.mu.Unlock()
		c.send(newEnvelope(message.ID, messageProtocolError, protocolErrorPayload{
			Code:    "operation_not_found",
			Message: "no retained operation has that request ID",
		}))

		return
	}
	outgoing := operationEnvelope(operation)
	c.server.mu.Unlock()
	c.send(outgoing)
}

func (s *Server) dispatchResults() {
	defer s.dispatchWG.Done()
	ticker := time.NewTicker(s.config.ResultPoll)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			for {
				result, ok := s.control.NextDebugResult()
				if !ok {
					break
				}
				s.retainAndPublish(result)
			}
		}
	}
}

func (s *Server) retainAndPublish(result debugging.Result) {
	s.mu.Lock()
	session := s.sessions[result.SessionID]
	if session == nil {
		s.mu.Unlock()

		return
	}
	operation := session.operations[result.RequestID]
	if operation == nil {
		operation = &operationRecord{
			request: debugging.Request{
				SessionID:               result.SessionID,
				ID:                      result.RequestID,
				CommandID:               result.CommandID,
				ExpectedSceneGeneration: 0,
				Arguments:               nil,
			},
			result: nil,
		}
		session.operations[result.RequestID] = operation
	}
	operation.result = &result
	session.lastUsed = time.Now()
	active := s.active
	if active == nil || active.sessionID != result.SessionID {
		s.mu.Unlock()

		return
	}
	outgoing := operationEnvelope(operation)
	s.mu.Unlock()
	active.send(outgoing)
}

// evictIdleSessionLocked removes the oldest acknowledged, disconnected session.
// The caller must hold s.mu.
func (s *Server) evictIdleSessionLocked() bool {
	var candidate *sessionState
	for _, session := range s.sessions {
		if len(session.operations) != 0 {
			continue
		}
		if s.active != nil && s.active.sessionID == session.id {
			continue
		}
		if candidate == nil || session.lastUsed.Before(candidate.lastUsed) {
			candidate = session
		}
	}
	if candidate == nil {
		return false
	}
	delete(s.sessions, candidate.id)

	return true
}

func operationEnvelope(operation *operationRecord) envelope {
	if operation.result == nil {
		return newEnvelope(operation.request.ID, messageCommandAccepted, commandAcceptedPayload{
			RequestID: operation.request.ID,
			CommandID: operation.request.CommandID,
		})
	}
	messageType := messageCommandCompleted
	if !operation.result.Success {
		messageType = messageCommandFailed
	}

	return newEnvelope(operation.request.ID, messageType, operation.result)
}

func validateConfig(config Config) error {
	host, port, err := net.SplitHostPort(config.Address)
	if err != nil {
		return fmt.Errorf("%w: address: %w", ErrInvalidConfig, err)
	}
	if host != "127.0.0.1" {
		return fmt.Errorf("%w: address must bind to 127.0.0.1", ErrInvalidConfig)
	}
	parsedPort, err := strconv.Atoi(port)
	if err != nil || parsedPort < 0 || parsedPort > 65535 {
		return fmt.Errorf("%w: invalid port %q", ErrInvalidConfig, port)
	}
	if config.OutboundCapacity <= 0 || config.MaxMessageBytes <= 0 || config.ResultPoll <= 0 {
		return fmt.Errorf("%w: capacities, message size, and polling interval must be positive", ErrInvalidConfig)
	}

	return nil
}

func randomID(size int) (string, error) {
	buffer := make([]byte, size)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buffer), nil
}
