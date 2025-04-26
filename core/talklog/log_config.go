package talklog

const (
	ColorReset  = "\033[0m"
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorCyan   = "\033[36m"
)

type LogConfig struct {
	LogToFile bool
	FilePath  string
	WithTime  bool
}
