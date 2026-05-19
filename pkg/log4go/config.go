package log4go

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the logging configuration
type Config struct {
	Loggers map[string]LoggerConfig `json:"loggers" yaml:"loggers"`
}

// LoggerConfig represents configuration for a single logger
type LoggerConfig struct {
	Level     string           `json:"level" yaml:"level"`
	Appenders []AppenderConfig `json:"appenders" yaml:"appenders"`
}

// AppenderConfig represents configuration for an appender
type AppenderConfig struct {
	Type   string                 `json:"type" yaml:"type"`
	Name   string                 `json:"name" yaml:"name"`
	Config map[string]interface{} `json:"config" yaml:"config"`
}

// LoadConfigFromFile loads configuration from a file (JSON or YAML)
func LoadConfigFromFile(filePath string) (*Config, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Try YAML first
	var config Config
	if err := yaml.Unmarshal(data, &config); err == nil {
		return &config, nil
	}

	// Try JSON
	if err := json.Unmarshal(data, &config); err == nil {
		return &config, nil
	}

	return nil, fmt.Errorf("failed to parse config file as YAML or JSON")
}

// ApplyConfig applies configuration to a logger
func ApplyConfig(logger Logger, config LoggerConfig) error {
	// Set level
	level := ParseLevel(config.Level)
	logger.SetLevel(level)

	// Create and add appenders
	for _, appenderCfg := range config.Appenders {
		appender, err := createAppender(appenderCfg)
		if err != nil {
			return fmt.Errorf("failed to create appender %s: %w", appenderCfg.Name, err)
		}
		logger.AddAppender(appender)
	}

	return nil
}

// createAppender creates an appender from configuration
func createAppender(config AppenderConfig) (Appender, error) {
	switch config.Type {
	case "console":
		return createConsoleAppender(config)
	case "file":
		return createFileAppender(config)
	case "rolling":
		return createRollingFileAppender(config)
	case "daily":
		return createDailyRollingFileAppender(config)
	case "async":
		return createAsyncAppender(config)
	default:
		return nil, fmt.Errorf("unknown appender type: %s", config.Type)
	}
}

// createConsoleAppender creates a console appender from configuration
func createConsoleAppender(config AppenderConfig) (Appender, error) {
	appender := NewConsoleAppender()

	// Apply formatter if specified
	if formatterType, ok := config.Config["formatter"].(string); ok {
		formatter := createFormatter(formatterType, config.Config)
		appender.SetFormatter(formatter)
	}

	return appender, nil
}

// createFileAppender creates a file appender from configuration
func createFileAppender(config AppenderConfig) (Appender, error) {
	filePath, ok := config.Config["path"].(string)
	if !ok {
		return nil, fmt.Errorf("file appender requires 'path' config")
	}

	appender, err := NewFileAppender(config.Name, filePath)
	if err != nil {
		return nil, err
	}

	// Apply formatter if specified
	if formatterType, ok := config.Config["formatter"].(string); ok {
		formatter := createFormatter(formatterType, config.Config)
		appender.SetFormatter(formatter)
	}

	return appender, nil
}

// createRollingFileAppender creates a rolling file appender from configuration
func createRollingFileAppender(config AppenderConfig) (Appender, error) {
	filePath, ok := config.Config["path"].(string)
	if !ok {
		return nil, fmt.Errorf("rolling file appender requires 'path' config")
	}

	maxSize := 10 // Default 10MB
	if size, ok := config.Config["max_size"].(int); ok {
		maxSize = size
	}

	maxBackups := 5 // Default 5 backups
	if backups, ok := config.Config["max_backups"].(int); ok {
		maxBackups = backups
	}

	policy := NewSizeBasedRollingPolicy(maxSize)
	appender, err := NewRollingFileAppender(config.Name, filePath, policy, maxBackups)
	if err != nil {
		return nil, err
	}

	// Apply formatter if specified
	if formatterType, ok := config.Config["formatter"].(string); ok {
		formatter := createFormatter(formatterType, config.Config)
		appender.SetFormatter(formatter)
	}

	return appender, nil
}

// createDailyRollingFileAppender creates a daily rolling file appender from configuration
func createDailyRollingFileAppender(config AppenderConfig) (Appender, error) {
	filePath, ok := config.Config["path"].(string)
	if !ok {
		return nil, fmt.Errorf("daily rolling file appender requires 'path' config")
	}

	maxBackups := 30 // Default 30 days
	if backups, ok := config.Config["max_backups"].(int); ok {
		maxBackups = backups
	}

	appender, err := NewDailyRollingFileAppender(config.Name, filePath, maxBackups)
	if err != nil {
		return nil, err
	}

	// Apply formatter if specified
	if formatterType, ok := config.Config["formatter"].(string); ok {
		formatter := createFormatter(formatterType, config.Config)
		appender.SetFormatter(formatter)
	}

	return appender, nil
}

// createAsyncAppender creates an async appender from configuration
func createAsyncAppender(config AppenderConfig) (Appender, error) {
	underlyingCfg, ok := config.Config["appender"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("async appender requires 'appender' config")
	}

	// Convert to AppenderConfig
	underlying := AppenderConfig{
		Type:   underlyingCfg["type"].(string),
		Name:   underlyingCfg["name"].(string),
		Config: underlyingCfg,
	}

	underlyingAppender, err := createAppender(underlying)
	if err != nil {
		return nil, err
	}

	queueSize := 1000 // Default queue size
	if size, ok := config.Config["queue_size"].(int); ok {
		queueSize = size
	}

	return NewAsyncAppender(config.Name, underlyingAppender, queueSize), nil
}

// createFormatter creates a formatter from configuration
func createFormatter(formatterType string, config map[string]interface{}) Formatter {
	switch formatterType {
	case "json":
		formatter := NewJSONFormatter()
		if prettyPrint, ok := config["pretty_print"].(bool); ok {
			formatter.PrettyPrint = prettyPrint
		}
		return formatter
	case "pattern":
		pattern := "%d %l [%n] %m"
		if p, ok := config["pattern"].(string); ok {
			pattern = p
		}
		return NewPatternFormatter(pattern)
	case "text":
		fallthrough
	default:
		formatter := NewTextFormatter()
		if useColors, ok := config["use_colors"].(bool); ok {
			formatter.UseColors = useColors
		}
		if showCaller, ok := config["show_caller"].(bool); ok {
			formatter.ShowCaller = showCaller
		}
		return formatter
	}
}

// Example configuration:
//
// loggers:
//   root:
//     level: INFO
//     appenders:
//       - type: console
//         name: console
//         config:
//           formatter: text
//           use_colors: true
//       - type: rolling
//         name: file
//         config:
//           path: /var/log/app.log
//           max_size: 10
//           max_backups: 5
//           formatter: json
//   myapp:
//     level: DEBUG
//     appenders:
//       - type: daily
//         name: daily
//         config:
//           path: /var/log/myapp.log
//           max_backups: 30
