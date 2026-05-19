package managers

import (
	"github.com/fluxorio/fluxor/pkg/core"
)

// CreateLogger creates a logger based on Managers configuration
func (m *Managers) CreateLogger() (core.Logger, error) {
	m.mu.RLock()
	config := m.config
	m.mu.RUnlock()

	if config.LogJSON {
		return core.NewJSONLogger(), nil
	}

	loggerConfig := core.LoggerConfig{
		JSONOutput:       config.LogJSON,
		Level:            config.LogLevel,
		AppendLogEnabled: false,
	}

	return core.NewLogger(loggerConfig), nil
}

// CreateJSONLogger creates a JSON logger
func (m *Managers) CreateJSONLogger() core.Logger {
	return core.NewJSONLogger()
}
