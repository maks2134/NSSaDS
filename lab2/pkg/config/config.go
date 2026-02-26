package config

import (
	"time"
)

type Config struct {
	Server ServerConfig `json:"server"`
	Client ClientConfig `json:"client"`
	UDP    UDPConfig    `json:"udp"`
}

type ServerConfig struct {
	Host           string        `json:"host"`
	Port           string        `json:"port"`
	KeepAlive      bool          `json:"keep_alive"`
	KeepAliveIdle  time.Duration `json:"keep_alive_idle"`
	KeepAliveCount int           `json:"keep_alive_count"`
	KeepAliveIntvl time.Duration `json:"keep_alive_intvl"`
	BufferSize     int           `json:"buffer_size"`
	UploadDir      string        `json:"upload_dir"`
	SessionTimeout time.Duration `json:"session_timeout"`
}

type ClientConfig struct {
	KeepAlive      bool          `json:"keep_alive"`
	KeepAliveIdle  time.Duration `json:"keep_alive_idle"`
	KeepAliveCount int           `json:"keep_alive_count"`
	KeepAliveIntvl time.Duration `json:"keep_alive_intvl"`
	BufferSize     int           `json:"buffer_size"`
	Timeout        time.Duration `json:"timeout"`
}

type UDPConfig struct {
	WindowSize            uint16        `json:"window_size"`
	PacketTimeout         time.Duration `json:"packet_timeout"`
	RetransmissionTimeout time.Duration `json:"retransmission_timeout"`
	MaxRetransmissions    int           `json:"max_retransmissions"`
	BufferSizes           []int         `json:"buffer_sizes"`
	TestDuration          time.Duration `json:"test_duration"`
	MinBufferSize         int           `json:"min_buffer_size"`
	MaxBufferSize         int           `json:"max_buffer_size"`
	BufferStep            int           `json:"buffer_step"`
}

func NewConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:           "localhost",
			Port:           "8080",
			KeepAlive:      true,
			KeepAliveIdle:  30 * time.Second,
			KeepAliveCount: 3,
			KeepAliveIntvl: 10 * time.Second,
			BufferSize:     8192,
			UploadDir:      "./uploads",
			SessionTimeout: 5 * time.Minute,
		},
		Client: ClientConfig{
			KeepAlive:      true,
			KeepAliveIdle:  30 * time.Second,
			KeepAliveCount: 3,
			KeepAliveIntvl: 10 * time.Second,
			BufferSize:     8192,
			Timeout:        30 * time.Second,
		},
		UDP: UDPConfig{
			WindowSize:            64,
			PacketTimeout:         100 * time.Millisecond,
			RetransmissionTimeout: 500 * time.Millisecond,
			MaxRetransmissions:    5,
			BufferSizes:           []int{512, 1024, 2048, 4096, 8192, 16384, 32768},
			TestDuration:          30 * time.Second,
			MinBufferSize:         256,
			MaxBufferSize:         65536,
			BufferStep:            256,
		},
	}
}
