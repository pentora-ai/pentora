// pkg/server/server.go
package server

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"time"

	"github.com/pentora-ai/pentora/pkg/safe"
	"github.com/rs/zerolog/log"
)

type Server struct {
	signals  chan os.Signal
	stopChan chan bool

	routinesPool *safe.Pool
}

func NewServer(routinesPool *safe.Pool) *Server {
	srv := &Server{
		signals:      make(chan os.Signal, 1),
		stopChan:     make(chan bool, 1),
		routinesPool: routinesPool,
	}

	srv.configureSignals()

	return srv
}

func (s *Server) Start(ctx context.Context) {
	go func() {
		<-ctx.Done()
		logger := log.Ctx(ctx)
		logger.Info().Msg("I have to go...")
		logger.Info().Msg("Stopping server gracefully...")
		s.Stop()
	}()

	s.routinesPool.GoCtx(s.listenSignals)
}

func (s *Server) Wait() {
	<-s.stopChan
}

func (s *Server) Stop() {
	defer log.Info().Msg("Server stopped")

	s.stopChan <- true
}

func (s *Server) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	go func(ctx context.Context) {
		<-ctx.Done()
		if errors.Is(ctx.Err(), context.Canceled) {
			return
		} else if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			panic("Timeout while stopping pentora, killing instance ✝")
		}
	}(ctx)

	s.routinesPool.Stop()

	signal.Stop(s.signals)
	close(s.signals)

	close(s.stopChan)

	cancel()
}
