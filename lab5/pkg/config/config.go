package config

import "time"

const (
	DefaultCount       = 4
	DefaultMaxTTL      = 30
	DefaultPayloadSize = 56
	DefaultTTL         = 64
	DefaultSmurfCount  = 3
	MaxSmurfCount      = 5
	ReadBufferSize     = 65535
)

type Config struct {
	Count       int
	Timeout     time.Duration
	Interval    time.Duration
	MaxTTL      int
	PayloadSize int
	PeekTimeout time.Duration
	SmurfCount  int
	SmurfMax    int
	DefaultTTL  int
}

func New() *Config {
	return &Config{
		Count:       DefaultCount,
		Timeout:     time.Second,
		Interval:    time.Second,
		MaxTTL:      DefaultMaxTTL,
		PayloadSize: DefaultPayloadSize,
		PeekTimeout: 100 * time.Millisecond,
		SmurfCount:  DefaultSmurfCount,
		SmurfMax:    MaxSmurfCount,
		DefaultTTL:  DefaultTTL,
	}
}
