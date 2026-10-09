package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/bradhannah/UltimaVRedux/internal/debugging/workbench"
	"github.com/bradhannah/UltimaVRedux/internal/logging"
)

const debugWorkbenchShutdownTimeout = 2 * time.Second

var (
	errInvalidDebugWorkbenchPort  = errors.New("invalid debug workbench port")
	errDebugWorkbenchGameRequired = errors.New("starting debug workbench requires a game")
)

type runningDebugWorkbench struct {
	server *workbench.Server
	game   *Game
}

func startDebugWorkbench(game *Game, port int) (*runningDebugWorkbench, error) {
	if game == nil {
		return nil, errDebugWorkbenchGameRequired
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("%w: %d", errInvalidDebugWorkbenchPort, port)
	}

	config := workbench.DefaultConfig()
	config.Address = net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	server, err := workbench.NewServer(game, config)
	if err != nil {
		return nil, fmt.Errorf("creating debug workbench: %w", err)
	}
	// Startup runs before RunGame; publish fresh choices before HTTP readers
	// can connect. All later provider reads stay on Game.Update.
	game.refreshDebugDiscoverySnapshot()
	if err := server.Start(); err != nil {
		return nil, fmt.Errorf("starting debug workbench: %w", err)
	}
	game.debugDiscoveryPolling.Store(true)

	logging.Info(logging.CategorySystem, "Debug workbench available at %s", server.URL())
	go func() {
		defer game.stopDebugDiscoveryPolling()
		for serveErr := range server.Errors() {
			logging.Error(logging.CategorySystem, "Debug workbench server failed: %v", serveErr)
		}
	}()

	return &runningDebugWorkbench{server: server, game: game}, nil
}

func shutdownDebugWorkbench(runtime *runningDebugWorkbench) error {
	if runtime == nil {
		return nil
	}
	defer runtime.game.stopDebugDiscoveryPolling()

	ctx, cancel := context.WithTimeout(context.Background(), debugWorkbenchShutdownTimeout)
	defer cancel()

	if err := runtime.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("stopping debug workbench: %w", err)
	}

	return nil
}
