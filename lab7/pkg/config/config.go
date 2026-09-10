package config

// Defaults for matrix multiply MPI lab.
const (
	DefaultN         = 2560
	DefaultPanel     = 64
	DefaultMode      = "blocking"
	MinProcessesHint = 3
)

// Config holds CLI parameters for a run.
type Config struct {
	N     int
	Panel int
	Mode  string // "blocking" | "nonblocking"
}
