package network

import (
	"NSSaDS/lab4/internal/domain"
	"NSSaDS/lab4/pkg/config"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type UDPServer struct {
	config     *config.Config
	registry   domain.ServiceRegistry
	threadPool domain.ThreadPool
	listeners  map[int]*net.UDPConn
	stats      map[domain.ServiceType]*domain.ServiceStats
	statsMutex sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

func NewUDPServer(cfg *config.Config, registry domain.ServiceRegistry, threadPool domain.ThreadPool) domain.UDPServer {
	ctx, cancel := context.WithCancel(context.Background())

	return &UDPServer{
		config:     cfg,
		registry:   registry,
		threadPool: threadPool,
		listeners:  make(map[int]*net.UDPConn),
		stats:      make(map[domain.ServiceType]*domain.ServiceStats),
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (s *UDPServer) Start(ctx context.Context) error {
	if err := s.threadPool.Start(ctx); err != nil {
		return fmt.Errorf("failed to start thread pool: %w", err)
	}

	for serviceType, serviceConfig := range s.config.Services {
		if !serviceConfig.Enabled {
			continue
		}

		service, err := s.registry.GetService(serviceType)
		if err != nil {
			log.Printf("Service %s not found in registry: %v", serviceType, err)
			continue
		}

		if err := s.startServiceListener(service, serviceConfig); err != nil {
			log.Printf("Failed to start listener for service %s: %v", serviceType, err)
			continue
		}

		s.stats[serviceType] = &domain.ServiceStats{}
		log.Printf("Started service %s on port %d", serviceType, service.Port())
	}

	log.Printf("UDP Multiservice Server started with %d services", len(s.listeners))
	return nil
}

func (s *UDPServer) Stop() error {
	s.cancel()

	for port, listener := range s.listeners {
		if err := listener.Close(); err != nil {
			log.Printf("Error closing listener on port %d: %v", port, err)
		}
	}

	if err := s.threadPool.Stop(); err != nil {
		log.Printf("Error stopping thread pool: %v", err)
	}

	s.wg.Wait()
	log.Println("UDP Server stopped")
	return nil
}

func (s *UDPServer) RegisterService(service domain.Service) error {
	return s.registry.RegisterService(service)
}

func (s *UDPServer) GetStats() map[domain.ServiceType]*domain.ServiceStats {
	s.statsMutex.RLock()
	defer s.statsMutex.RUnlock()

	stats := make(map[domain.ServiceType]*domain.ServiceStats)
	for serviceType, stat := range s.stats {
		stats[serviceType] = &domain.ServiceStats{
			RequestsReceived:  stat.RequestsReceived,
			RequestsProcessed: stat.RequestsProcessed,
			Errors:            stat.Errors,
			AvgResponseTime:   stat.AvgResponseTime,
			LastRequest:       stat.LastRequest,
		}
	}

	return stats
}

func (s *UDPServer) startServiceListener(service domain.Service, serviceConfig *config.ServiceConfig) error {
	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", s.config.Server.Host, serviceConfig.Port))
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on UDP: %w", err)
	}

	if err := conn.SetReadBuffer(s.config.Server.ReadBuffer); err != nil {
		log.Printf("Warning: failed to set read buffer: %v", err)
	}

	if err := conn.SetWriteBuffer(s.config.Server.WriteBuffer); err != nil {
		log.Printf("Warning: failed to set write buffer: %v", err)
	}

	s.listeners[serviceConfig.Port] = conn
	s.wg.Add(1)

	go s.handleServiceConnections(service, conn, serviceConfig)
	return nil
}

func (s *UDPServer) handleServiceConnections(service domain.Service, conn *net.UDPConn, config *config.ServiceConfig) {
	defer s.wg.Done()

	buffer := make([]byte, s.config.Server.MaxPacketSize)

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			conn.SetReadDeadline(time.Now().Add(s.config.Server.IdleTimeout))
			n, clientAddr, err := conn.ReadFromUDP(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if s.ctx.Err() != nil {
					return
				}
				log.Printf("Error reading from UDP: %v", err)
				continue
			}

			s.wg.Add(1)
			err = s.threadPool.Submit(func() {
				defer s.wg.Done()
				s.handleRequest(service, conn, clientAddr, buffer[:n], config)
			})

			if err != nil {
				s.wg.Done()
				log.Printf("Failed to submit task to thread pool: %v", err)
				atomic.AddInt64(&s.stats[service.Name()].Errors, 1)
			}
		}
	}
}

func (s *UDPServer) handleRequest(service domain.Service, conn *net.UDPConn, clientAddr *net.UDPAddr, data []byte, config *config.ServiceConfig) {
	startTime := time.Now()

	s.updateStats(service.Name(), func(stats *domain.ServiceStats) {
		stats.RequestsReceived++
		stats.LastRequest = startTime
	})

	request, err := s.parseRequest(data, clientAddr)
	if err != nil {
		s.sendError(conn, clientAddr, request.ID, service.Name(), err)
		atomic.AddInt64(&s.stats[service.Name()].Errors, 1)
		return
	}

	ctx, cancel := context.WithTimeout(s.ctx, config.Timeout)
	defer cancel()

	response := s.processRequest(ctx, service, request)

	responseTime := time.Since(startTime)
	s.updateStats(service.Name(), func(stats *domain.ServiceStats) {
		stats.RequestsProcessed++
		stats.AvgResponseTime = (stats.AvgResponseTime*time.Duration(stats.RequestsProcessed-1) + responseTime) / time.Duration(stats.RequestsProcessed)
	})

	s.sendResponse(conn, clientAddr, response)
}

func (s *UDPServer) parseRequest(data []byte, clientAddr net.Addr) (*domain.Request, error) {
	var req struct {
		ID      string `json:"id"`
		Command string `json:"command"`
		Data    string `json:"data"`
	}

	if err := json.Unmarshal(data, &req); err != nil {
		return &domain.Request{
			ID:         uuid.New().String(),
			Service:    "",
			Command:    "",
			Data:       data,
			ClientAddr: clientAddr,
			Timestamp:  time.Now(),
		}, nil
	}

	if req.ID == "" {
		req.ID = uuid.New().String()
	}

	return &domain.Request{
		ID:         req.ID,
		Service:    "",
		Command:    req.Command,
		Data:       []byte(req.Data),
		ClientAddr: clientAddr,
		Timestamp:  time.Now(),
	}, nil
}

func (s *UDPServer) processRequest(ctx context.Context, service domain.Service, request *domain.Request) *domain.Response {
	response, err := service.HandleRequest(ctx, request)
	if err != nil {
		return &domain.Response{
			ID:        request.ID,
			Service:   service.Name(),
			Error:     err,
			Timestamp: time.Now(),
		}
	}
	return response
}

func (s *UDPServer) sendResponse(conn *net.UDPConn, clientAddr *net.UDPAddr, response *domain.Response) {
	var responseData []byte
	var err error

	if response.Error != nil {
		responseData = []byte(fmt.Sprintf("ERROR: %s", response.Error.Error()))
	} else {
		responseData = response.Data
	}

	responseJSON := map[string]interface{}{
		"id":        response.ID,
		"service":   response.Service,
		"data":      string(responseData),
		"timestamp": response.Timestamp.Unix(),
	}

	if response.Error != nil {
		responseJSON["error"] = response.Error.Error()
	}

	data, err := json.Marshal(responseJSON)
	if err != nil {
		log.Printf("Error marshaling response: %v", err)
		return
	}

	_, err = conn.WriteToUDP(data, clientAddr)
	if err != nil {
		log.Printf("Error sending response: %v", err)
	}
}

func (s *UDPServer) sendError(conn *net.UDPConn, clientAddr *net.UDPAddr, requestID string, serviceType domain.ServiceType, err error) {
	errorResponse := map[string]interface{}{
		"id":        requestID,
		"service":   serviceType,
		"error":     err.Error(),
		"timestamp": time.Now().Unix(),
	}

	data, _ := json.Marshal(errorResponse)
	conn.WriteToUDP(data, clientAddr)
}

func (s *UDPServer) updateStats(serviceType domain.ServiceType, updateFunc func(*domain.ServiceStats)) {
	s.statsMutex.Lock()
	defer s.statsMutex.Unlock()

	if stats, exists := s.stats[serviceType]; exists {
		updateFunc(stats)
	}
}
