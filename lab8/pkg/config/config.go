package config

// Defaults for matrix multiply MPI lab with groups and files.
const (
	DefaultN         = 2560
	DefaultPanel     = 64
	DefaultMode      = "compare"
	DefaultGroups    = 2
	DefaultSeed      = 1
	DefaultAPath     = "data/A.bin"
	DefaultBPath     = "data/B.bin"
	DefaultOutDir    = "data"
	MinProcessesHint = 3
)

// Config holds CLI parameters for a run.
type Config struct {
	N      int
	Panel  int
	Mode   string // "blocking" | "nonblocking" | "collective" | "compare"
	Groups int
	Seed   int64
	APath  string
	BPath  string
	OutDir string
	Gen    bool
}
