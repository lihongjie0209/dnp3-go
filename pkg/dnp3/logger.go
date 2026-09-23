package dnp3

import (
	"github.com/lihongjie0209/dnp3-go/pkg/internal/logger"
)

// LogLevel represents logging level
type LogLevel int

const (
	// LevelDebug shows all log messages (most verbose)
	LevelDebug LogLevel = iota
	// LevelInfo shows info, warn, and error messages (default)
	LevelInfo
	// LevelWarn shows warn and error messages
	LevelWarn
	// LevelError shows only error messages
	LevelError
)

// SetLogLevel sets the global logging level
// Use this to enable/disable different levels of logging output
func SetLogLevel(level LogLevel) {
	debugLogger := logger.NewDefaultLogger(logger.Level(level))
	logger.SetDefault(debugLogger)
}

// EnableFrameDebug enables or disables detailed frame debugging
// When enabled, shows hex dumps of all DNP3 frames sent and received
func EnableFrameDebug(enable bool) {
	logger.SetFrameDebug(enable)
}
