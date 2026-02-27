package network

import (
	"NSSaDS/lab3/internal/domain"
	"context"
)

type TCPServer struct {
	multiplexer domain.Multiplexer
	handler     domain.CommandHandler
}

func NewTCPServer(multiplexer domain.Multiplexer, handler domain.CommandHandler) *TCPServer {
	return &TCPServer{
		multiplexer: multiplexer,
		handler:     handler,
	}
}

func (s *TCPServer) Start(ctx context.Context, config *domain.ServerConfig) error {
	s.multiplexer.SetHandler(s.handler)

	return s.multiplexer.Start(ctx, config)
}

func (s *TCPServer) Stop() error {
	return s.multiplexer.Stop()
}

func (s *TCPServer) SetHandler(handler domain.CommandHandler) {
	s.handler = handler
	s.multiplexer.SetHandler(handler)
}
