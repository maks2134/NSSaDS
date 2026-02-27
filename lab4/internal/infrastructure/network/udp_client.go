package network

import (
	"NSSaDS/lab4/internal/domain"
	"NSSaDS/lab4/pkg/config"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
)

type UDPClient struct {
	conn       *net.UDPConn
	config     *config.Config
	timeout    time.Duration
	mutex      sync.RWMutex
	responses  map[string]chan *domain.Response
	responseMu sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewUDPClient(cfg *config.Config) domain.UDPClient {
	ctx, cancel := context.WithCancel(context.Background())

	return &UDPClient{
		config:    cfg,
		timeout:   10 * time.Second,
		responses: make(map[string]chan *domain.Response),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (c *UDPClient) Connect(ctx context.Context, addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to resolve UDP address: %w", err)
	}

	conn, err := net.DialUDP("udp", nil, udpAddr)
	if err != nil {
		return fmt.Errorf("failed to dial UDP: %w", err)
	}

	c.mutex.Lock()
	c.conn = conn
	c.mutex.Unlock()

	go c.listenResponses()

	return nil
}

func (c *UDPClient) Disconnect() error {
	c.cancel()

	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}

	return nil
}

func (c *UDPClient) SendRequest(service domain.ServiceType, command string, data []byte) (*domain.Response, error) {
	c.mutex.RLock()
	conn := c.conn
	c.mutex.RUnlock()

	if conn == nil {
		return nil, fmt.Errorf("client not connected")
	}

	requestID := uuid.New().String()

	request := map[string]interface{}{
		"id":      requestID,
		"command": command,
		"data":    string(data),
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	responseChan := make(chan *domain.Response, 1)
	c.responseMu.Lock()
	c.responses[requestID] = responseChan
	c.responseMu.Unlock()

	defer func() {
		c.responseMu.Lock()
		delete(c.responses, requestID)
		c.responseMu.Unlock()
		close(responseChan)
	}()

	_, err = conn.Write(requestData)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	select {
	case response := <-responseChan:
		return response, nil
	case <-time.After(c.timeout):
		return nil, fmt.Errorf("request timeout")
	case <-c.ctx.Done():
		return nil, fmt.Errorf("client shutdown")
	}
}

func (c *UDPClient) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
}

func (c *UDPClient) listenResponses() {
	buffer := make([]byte, 4096)

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			c.mutex.RLock()
			conn := c.conn
			c.mutex.RUnlock()

			if conn == nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
			n, err := conn.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue
				}
				if c.ctx.Err() != nil {
					return
				}
				continue
			}

			response, err := c.parseResponse(buffer[:n])
			if err != nil {
				continue
			}

			c.responseMu.RLock()
			if responseChan, exists := c.responses[response.ID]; exists {
				select {
				case responseChan <- response:
				default:
				}
			}
			c.responseMu.RUnlock()
		}
	}
}

func (c *UDPClient) parseResponse(data []byte) (*domain.Response, error) {
	var resp struct {
		ID        string `json:"id"`
		Service   string `json:"service"`
		Data      string `json:"data"`
		Error     string `json:"error"`
		Timestamp int64  `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	response := &domain.Response{
		ID:        resp.ID,
		Service:   domain.ServiceType(resp.Service),
		Data:      []byte(resp.Data),
		Timestamp: time.Unix(resp.Timestamp, 0),
	}

	if resp.Error != "" {
		response.Error = fmt.Errorf(resp.Error)
	}

	return response, nil
}
