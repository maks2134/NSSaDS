package domain

import (
	"context"
	"errors"
	"net"
	"time"
)

type ServiceType string

const (
	EchoService  ServiceType = "echo"
	TimeService  ServiceType = "time"
	FileService  ServiceType = "file"
	CalcService  ServiceType = "calc"
	StatsService ServiceType = "stats"
)

type Service interface {
	Name() ServiceType
	HandleRequest(ctx context.Context, req *Request) (*Response, error)
	Port() int
}

type Request struct {
	ID         string
	Service    ServiceType
	Command    string
	Data       []byte
	ClientAddr net.Addr
	Timestamp  time.Time
}

type Response struct {
	ID        string
	Service   ServiceType
	Data      []byte
	Error     error
	Timestamp time.Time
}

type ServiceRegistry interface {
	RegisterService(service Service) error
	GetService(serviceType ServiceType) (Service, error)
	ListServices() []ServiceType
	GetServicePort(serviceType ServiceType) (int, error)
}

type ThreadPool interface {
	Submit(task func()) error
	Start(ctx context.Context) error
	Stop() error
	Stats() *PoolStats
}

type PoolStats struct {
	ActiveWorkers  int
	QueuedTasks    int
	CompletedTasks int64
	MinWorkers     int
	MaxWorkers     int
	CurrentWorkers int
}

type UDPServer interface {
	Start(ctx context.Context) error
	Stop() error
	RegisterService(service Service) error
	GetStats() map[ServiceType]*ServiceStats
}

type ServiceStats struct {
	RequestsReceived  int64
	RequestsProcessed int64
	Errors            int64
	AvgResponseTime   time.Duration
	LastRequest       time.Time
}

type UDPClient interface {
	Connect(ctx context.Context, addr string) error
	Disconnect() error
	SendRequest(service ServiceType, command string, data []byte) (*Response, error)
	SetTimeout(timeout time.Duration)
}

type Config struct {
	Services   map[ServiceType]*ServiceConfig
	ThreadPool *ThreadPoolConfig
	Server     *ServerConfig
}

type ServiceConfig struct {
	Port        int
	Enabled     bool
	MaxRequests int64
	Timeout     time.Duration
}

type ThreadPoolConfig struct {
	MinWorkers      int
	MaxWorkers      int
	QueueSize       int
	WorkerTimeout   time.Duration
	ExpandThreshold float64
}

type ServerConfig struct {
	Host          string
	ReadBuffer    int
	WriteBuffer   int
	MaxPacketSize int
	IdleTimeout   time.Duration
}

var (
	ErrQueueFull       = errors.New("task queue is full")
	ErrServiceNotFound = errors.New("service not found")
	ErrInvalidRequest  = errors.New("invalid request")
	ErrServiceDisabled = errors.New("service is disabled")
	ErrServiceExists   = errors.New("service already exists")
	ErrPortInUse       = errors.New("port already in use")
)
