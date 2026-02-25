package config

import (
	"time"
)

type Config struct {
	Server ServerConfig `json:"server"`
	Client ClientConfig `json:"client"`
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
	}
}
