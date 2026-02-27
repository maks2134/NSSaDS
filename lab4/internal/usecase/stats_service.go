package usecase

import (
	"NSSaDS/lab4/internal/domain"
	"context"
	"encoding/json"
	"fmt"
	"time"
)

type StatsService struct {
	port   int
	server domain.UDPServer
}

func NewStatsService(port int, server domain.UDPServer) domain.Service {
	return &StatsService{
		port:   port,
		server: server,
	}
}

func (s *StatsService) Name() domain.ServiceType { return domain.StatsService }
func (s *StatsService) Port() int                { return s.port }

func (s *StatsService) HandleRequest(ctx context.Context, req *domain.Request) (*domain.Response, error) {
	command := req.Command
	stats := s.server.GetStats()

	switch command {
	case "ALL":
		return s.handleAllStats(stats)
	case "SERVICE":
		return s.handleServiceStats(stats, req.Data)
	case "POOL":
		return s.handlePoolStats()
	case "HELP":
		return s.handleHelp()
	default:
		return &domain.Response{
			ID:        req.ID,
			Service:   s.Name(),
			Error:     fmt.Errorf("unknown command: %s", command),
			Timestamp: time.Now(),
		}, nil
	}
}

func (s *StatsService) handleAllStats(stats map[domain.ServiceType]*domain.ServiceStats) (*domain.Response, error) {
	response := make(map[string]interface{})

	for serviceType, serviceStats := range stats {
		response[string(serviceType)] = map[string]interface{}{
			"requests_received":  serviceStats.RequestsReceived,
			"requests_processed": serviceStats.RequestsProcessed,
			"errors":             serviceStats.Errors,
			"avg_response_time":  serviceStats.AvgResponseTime.String(),
			"last_request":       serviceStats.LastRequest.Format(time.RFC3339),
		}
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return &domain.Response{
			ID:        "",
			Service:   s.Name(),
			Error:     fmt.Errorf("failed to marshal stats: %w", err),
			Timestamp: time.Now(),
		}, nil
	}

	return &domain.Response{
		ID:        "",
		Service:   s.Name(),
		Data:      data,
		Timestamp: time.Now(),
	}, nil
}

func (s *StatsService) handleServiceStats(stats map[domain.ServiceType]*domain.ServiceStats, data []byte) (*domain.Response, error) {
	serviceType := domain.ServiceType(string(data))

	serviceStats, exists := stats[serviceType]
	if !exists {
		return &domain.Response{
			ID:        "",
			Service:   s.Name(),
			Error:     fmt.Errorf("service %s not found", serviceType),
			Timestamp: time.Now(),
		}, nil
	}

	response := map[string]interface{}{
		"service":            string(serviceType),
		"requests_received":  serviceStats.RequestsReceived,
		"requests_processed": serviceStats.RequestsProcessed,
		"errors":             serviceStats.Errors,
		"avg_response_time":  serviceStats.AvgResponseTime.String(),
		"last_request":       serviceStats.LastRequest.Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return &domain.Response{
			ID:        "",
			Service:   s.Name(),
			Error:     fmt.Errorf("failed to marshal stats: %w", err),
			Timestamp: time.Now(),
		}, nil
	}

	return &domain.Response{
		ID:        "",
		Service:   s.Name(),
		Data:      data,
		Timestamp: time.Now(),
	}, nil
}

func (s *StatsService) handlePoolStats() (*domain.Response, error) {
	return &domain.Response{
		ID:        "",
		Service:   s.Name(),
		Data:      []byte("Pool stats not implemented in this version"),
		Timestamp: time.Now(),
	}, nil
}

func (s *StatsService) handleHelp() (*domain.Response, error) {
	help := `Stats Service Commands:
ALL - Show statistics for all services
SERVICE <service_name> - Show statistics for specific service
POOL - Show thread pool statistics
HELP - Show this help message

Available services: echo, time, calc, stats`

	return &domain.Response{
		ID:        "",
		Service:   s.Name(),
		Data:      []byte(help),
		Timestamp: time.Now(),
	}, nil
}
