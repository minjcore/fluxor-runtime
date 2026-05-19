package concurrency

import (
	"fmt"
	"log"
	"os"
)

// simpleLogger is a minimal logger interface to avoid import cycles
// This allows concurrency package to log errors without importing core
type simpleLogger interface {
	Errorf(format string, args ...interface{})
}

// defaultSimpleLogger implements simpleLogger using standard log
type defaultSimpleLogger struct {
	logger *log.Logger
}

func newDefaultSimpleLogger() simpleLogger {
	return &defaultSimpleLogger{
		logger: log.New(os.Stderr, "[ERROR] ", log.LstdFlags|log.Lshortfile),
	}
}

func (l *defaultSimpleLogger) Errorf(format string, args ...interface{}) {
	if err := l.logger.Output(3, fmt.Sprintf(format, args...)); err != nil {
		// Best-effort logging; ignore on error.
	}
}
