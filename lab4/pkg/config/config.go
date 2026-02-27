package config

import (
	"NSSaDS/lab4/internal/domain"
	"time"
)

type Config struct {
	Services   map[domain.ServiceType]*ServiceConfig
	ThreadPool *ThreadPoolConfig
	Server     *ServerConfig
}

type ServiceConfig struct {
	Port        int           `json:"port" yaml:"port"`
	Enabled     bool          `json:"enabled" yaml:"enabled"`
	MaxRequests int64         `json:"max_requests" yaml:"max_requests"`
	Timeout     time.Duration `json:"timeout" yaml:"timeout"`
}

type ThreadPoolConfig struct {
	MinWorkers      int           `json:"min_workers" yaml:"min_workers"`
	MaxWorkers      int           `json:"max_workers" yaml:"max_workers"`
	QueueSize       int           `json:"queue_size" yaml:"queue_size"`
	WorkerTimeout   time.Duration `json:"worker_timeout" yaml:"worker_timeout"`
	ExpandThreshold float64       `json:"expand_threshold" yaml:"expand_threshold"`
}

type ServerConfig struct {
	Host          string        `json:"host" yaml:"host"`
	ReadBuffer    int           `json:"read_buffer" yaml:"read_buffer"`
	WriteBuffer   int           `json:"write_buffer" yaml:"write_buffer"`
	MaxPacketSize int           `json:"max_packet_size" yaml:"max_packet_size"`
	IdleTimeout   time.Duration `json:"idle_timeout" yaml:"idle_timeout"`
}

func NewConfig() *Config {
	return &Config{
		Services: map[domain.ServiceType]*ServiceConfig{
			domain.EchoService: {
				Port:        8081,
				Enabled:     true,
				MaxRequests: 1000,
				Timeout:     5 * time.Second,
			},
			domain.TimeService: {
				Port:        8082,
				Enabled:     true,
				MaxRequests: 1000,
				Timeout:     5 * time.Second,
			},
			domain.FileService: {
				Port:        8083,
				Enabled:     true,
				MaxRequests: 500,
				Timeout:     30 * time.Second,
			},
			domain.CalcService: {
				Port:        8084,
				Enabled:     true,
				MaxRequests: 1000,
				Timeout:     10 * time.Second,
			},
			domain.StatsService: {
				Port:        8085,
				Enabled:     true,
				MaxRequests: 100,
				Timeout:     5 * time.Second,
			},
		},
		ThreadPool: &ThreadPoolConfig{
			MinWorkers:      5,
			MaxWorkers:      50,
			QueueSize:       1000,
			WorkerTimeout:   30 * time.Second,
			ExpandThreshold: 0.8,
		},
		Server: &ServerConfig{
			Host:          "localhost",
			ReadBuffer:    4096,
			WriteBuffer:   4096,
			MaxPacketSize: 64 * 1024,
			IdleTimeout:   60 * time.Second,
		},
	}
}
