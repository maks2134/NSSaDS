package config

import "time"

const (
	DefaultPort    = 9000
	DefaultGroup   = "239.255.42.42"
	ReadBufferSize = 65535
	HelloInterval  = 2 * time.Second
	PeerTimeout    = 8 * time.Second
	RecvTimeout    = 200 * time.Millisecond
	MaxPayloadSize = 1400
	MulticastTTL   = 1
)

type Config struct {
	Port      int
	IfaceName string
	Group     string
	Nick      string
}

func New() *Config {
	return &Config{
		Port:  DefaultPort,
		Group: DefaultGroup,
	}
}
