package log4go

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Formatter formats log entries into strings
type Formatter interface {
	Format(entry *LogEntry) string
}

// TextFormatter formats logs as plain text
type TextFormatter struct {
	TimestampFormat string
	UseColors       bool
	ShowCaller      bool
	CallerDepth     int
}

// NewTextFormatter creates a new text formatter
func NewTextFormatter() *TextFormatter {
	return &TextFormatter{
		TimestampFormat: time.RFC3339,
		UseColors:       true,
		ShowCaller:      true,
		CallerDepth:     2,
	}
}

// ANSI color codes
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[37m"
	colorWhite  = "\033[97m"
)

// getLevelColor returns the ANSI color code for a level
func getLevelColor(level Level) string {
	switch level {
	case TRACE:
		return colorGray
	case DEBUG:
		return colorCyan
	case INFO:
		return colorGreen
	case WARN:
		return colorYellow
	case ERROR:
		return colorRed
	case FATAL:
		return colorPurple
	default:
		return colorWhite
	}
}

// Format formats a log entry as text
func (f *TextFormatter) Format(entry *LogEntry) string {
	var builder strings.Builder

	// Timestamp
	timestamp := entry.Timestamp.Format(f.TimestampFormat)
	builder.WriteString(timestamp)
	builder.WriteString(" ")

	// Level with color
	levelStr := entry.Level.String()
	if f.UseColors {
		color := getLevelColor(entry.Level)
		builder.WriteString(color)
		builder.WriteString(fmt.Sprintf("%-5s", levelStr))
		builder.WriteString(colorReset)
	} else {
		builder.WriteString(fmt.Sprintf("[%-5s]", levelStr))
	}
	builder.WriteString(" ")

	// Logger name
	builder.WriteString("[")
	builder.WriteString(entry.Logger)
	builder.WriteString("] ")

	// Caller info
	if f.ShowCaller && entry.File != "" {
		file := filepath.Base(entry.File)
		builder.WriteString(fmt.Sprintf("%s:%d ", file, entry.Line))
	}

	// Message
	builder.WriteString(entry.Message)

	// Fields
	if len(entry.Fields) > 0 {
		builder.WriteString(" ")
		builder.WriteString(formatFields(entry.Fields))
	}

	return builder.String()
}

// formatFields formats fields as key=value pairs
func formatFields(fields Fields) string {
	var pairs []string
	for k, v := range fields {
		pairs = append(pairs, fmt.Sprintf("%s=%v", k, v))
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

// JSONFormatter formats logs as JSON
type JSONFormatter struct {
	TimestampFormat string
	PrettyPrint     bool
}

// NewJSONFormatter creates a new JSON formatter
func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{
		TimestampFormat: time.RFC3339,
		PrettyPrint:     false,
	}
}

// Format formats a log entry as JSON
func (f *JSONFormatter) Format(entry *LogEntry) string {
	data := map[string]interface{}{
		"timestamp": entry.Timestamp.Format(f.TimestampFormat),
		"level":     entry.Level.String(),
		"logger":    entry.Logger,
		"message":   entry.Message,
	}

	if entry.File != "" {
		data["file"] = entry.File
		data["line"] = entry.Line
		data["function"] = entry.Function
	}

	if len(entry.Fields) > 0 {
		data["fields"] = entry.Fields
	}

	var jsonData []byte
	var err error

	if f.PrettyPrint {
		jsonData, err = json.MarshalIndent(data, "", "  ")
	} else {
		jsonData, err = json.Marshal(data)
	}

	if err != nil {
		return fmt.Sprintf("ERROR: Failed to marshal log entry: %v", err)
	}

	return string(jsonData)
}

// PatternFormatter formats logs using a pattern string
// Pattern syntax:
//   %d - timestamp
//   %l - level
//   %n - logger name
//   %m - message
//   %f - file name
//   %L - line number
//   %F - function name
//   %% - literal %
type PatternFormatter struct {
	Pattern         string
	TimestampFormat string
}

// NewPatternFormatter creates a new pattern formatter
func NewPatternFormatter(pattern string) *PatternFormatter {
	return &PatternFormatter{
		Pattern:         pattern,
		TimestampFormat: time.RFC3339,
	}
}

// Format formats a log entry using the pattern
func (f *PatternFormatter) Format(entry *LogEntry) string {
	result := f.Pattern

	// Replace placeholders
	result = strings.ReplaceAll(result, "%d", entry.Timestamp.Format(f.TimestampFormat))
	result = strings.ReplaceAll(result, "%l", entry.Level.String())
	result = strings.ReplaceAll(result, "%n", entry.Logger)
	result = strings.ReplaceAll(result, "%m", entry.Message)
	result = strings.ReplaceAll(result, "%f", filepath.Base(entry.File))
	result = strings.ReplaceAll(result, "%L", fmt.Sprintf("%d", entry.Line))
	result = strings.ReplaceAll(result, "%F", entry.Function)
	result = strings.ReplaceAll(result, "%%", "%")

	// Add fields if present
	if len(entry.Fields) > 0 {
		result += " " + formatFields(entry.Fields)
	}

	return result
}

// DefaultFormatter returns the default formatter (text)
func DefaultFormatter() Formatter {
	return NewTextFormatter()
}
