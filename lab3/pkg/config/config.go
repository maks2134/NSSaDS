package config

import (
	"time"
)

type Config struct {
	Host               string        `json:"host"`
	Port               string        `json:"port"`
	MaxConnections     int           `json:"max_connections"`
	ChunkSize          int           `json:"chunk_size"`
	InteractiveTimeout time.Duration `json:"interactive_timeout"`
	SelectTimeout      time.Duration `json:"select_timeout"`
	SessionTimeout     time.Duration `json:"session_timeout"`
	BufferSize         int           `json:"buffer_size"`
}

func NewConfig() *Config {
	return &Config{
		Host:               "localhost",
		Port:               "8080",
		MaxConnections:     1000,
		ChunkSize:          512,                    // Small for interactive response (ping * 10 = 5ms for 512 bytes)
		InteractiveTimeout: 100 * time.Millisecond, // ping * 10ms
		SelectTimeout:      10 * time.Millisecond,
		SessionTimeout:     5 * time.Minute,
		BufferSize:         8192,
	}
}
