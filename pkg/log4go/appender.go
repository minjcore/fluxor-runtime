package log4go

import (
	"fmt"
	"io"
	"os"
	"sync"
)

// Appender writes log entries to a destination
type Appender interface {
	// Name returns the appender name
	Name() string

	// Append writes a log entry
	Append(entry *LogEntry)

	// Close closes the appender and releases resources
	Close() error

	// SetFormatter sets the formatter for this appender
	SetFormatter(formatter Formatter)

	// GetFormatter returns the current formatter
	GetFormatter() Formatter
}

// baseAppender provides common functionality for all appenders
type baseAppender struct {
	name      string
	formatter Formatter
	mu        sync.Mutex
}

// Name returns the appender name
func (a *baseAppender) Name() string {
	return a.name
}

// SetFormatter sets the formatter
func (a *baseAppender) SetFormatter(formatter Formatter) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.formatter = formatter
}

// GetFormatter returns the formatter
func (a *baseAppender) GetFormatter() Formatter {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.formatter == nil {
		return DefaultFormatter()
	}
	return a.formatter
}

// ConsoleAppender writes to stdout/stderr
type ConsoleAppender struct {
	*baseAppender
	writer io.Writer
}

// NewConsoleAppender creates a console appender
func NewConsoleAppender() Appender {
	return &ConsoleAppender{
		baseAppender: &baseAppender{
			name:      "console",
			formatter: DefaultFormatter(),
		},
		writer: os.Stdout,
	}
}

// NewConsoleAppenderWithWriter creates a console appender with custom writer
func NewConsoleAppenderWithWriter(name string, writer io.Writer) Appender {
	return &ConsoleAppender{
		baseAppender: &baseAppender{
			name:      name,
			formatter: DefaultFormatter(),
		},
		writer: writer,
	}
}

// Append writes the log entry to console
func (a *ConsoleAppender) Append(entry *LogEntry) {
	formatter := a.GetFormatter()
	formatted := formatter.Format(entry)

	a.mu.Lock()
	defer a.mu.Unlock()

	// Write to stderr for ERROR and FATAL
	writer := a.writer
	if entry.Level >= ERROR {
		writer = os.Stderr
	}

	fmt.Fprintln(writer, formatted)
}

// Close closes the console appender
func (a *ConsoleAppender) Close() error {
	return nil
}

// FileAppender writes to a file
type FileAppender struct {
	*baseAppender
	file     *os.File
	filePath string
	mu       sync.Mutex
}

// NewFileAppender creates a file appender
func NewFileAppender(name string, filePath string) (Appender, error) {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	return &FileAppender{
		baseAppender: &baseAppender{
			name:      name,
			formatter: DefaultFormatter(),
		},
		file:     file,
		filePath: filePath,
	}, nil
}

// Append writes the log entry to file
func (a *FileAppender) Append(entry *LogEntry) {
	formatter := a.GetFormatter()
	formatted := formatter.Format(entry)

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.file != nil {
		fmt.Fprintln(a.file, formatted)
	}
}

// Close closes the file appender
func (a *FileAppender) Close() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.file != nil {
		err := a.file.Close()
		a.file = nil
		return err
	}
	return nil
}

// AsyncAppender wraps another appender and writes asynchronously
type AsyncAppender struct {
	*baseAppender
	underlying Appender
	queue      chan *LogEntry
	wg         sync.WaitGroup
	closed     bool
	closeMu    sync.Mutex
}

// NewAsyncAppender creates an async appender
func NewAsyncAppender(name string, underlying Appender, queueSize int) Appender {
	async := &AsyncAppender{
		baseAppender: &baseAppender{
			name:      name,
			formatter: underlying.GetFormatter(),
		},
		underlying: underlying,
		queue:      make(chan *LogEntry, queueSize),
		closed:     false,
	}

	// Start worker goroutine
	async.wg.Add(1)
	go async.worker()

	return async
}

// worker processes log entries asynchronously
func (a *AsyncAppender) worker() {
	defer a.wg.Done()

	for entry := range a.queue {
		a.underlying.Append(entry)
	}
}

// Append queues the log entry for async writing
func (a *AsyncAppender) Append(entry *LogEntry) {
	a.closeMu.Lock()
	closed := a.closed
	a.closeMu.Unlock()

	if !closed {
		select {
		case a.queue <- entry:
			// Successfully queued
		default:
			// Queue full, drop the message (fail-fast)
			// Could also block here with: a.queue <- entry
		}
	}
}

// Close closes the async appender and waits for pending logs
func (a *AsyncAppender) Close() error {
	a.closeMu.Lock()
	if a.closed {
		a.closeMu.Unlock()
		return nil
	}
	a.closed = true
	a.closeMu.Unlock()

	// Close queue and wait for worker to finish
	close(a.queue)
	a.wg.Wait()

	// Close underlying appender
	return a.underlying.Close()
}

// SetFormatter sets the formatter for both this and underlying appender
func (a *AsyncAppender) SetFormatter(formatter Formatter) {
	a.baseAppender.SetFormatter(formatter)
	a.underlying.SetFormatter(formatter)
}

// MultiAppender writes to multiple appenders
type MultiAppender struct {
	*baseAppender
	appenders []Appender
	mu        sync.RWMutex
}

// NewMultiAppender creates a multi appender
func NewMultiAppender(name string, appenders ...Appender) Appender {
	return &MultiAppender{
		baseAppender: &baseAppender{
			name:      name,
			formatter: DefaultFormatter(),
		},
		appenders: appenders,
	}
}

// Append writes to all underlying appenders
func (a *MultiAppender) Append(entry *LogEntry) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	for _, appender := range a.appenders {
		appender.Append(entry)
	}
}

// Close closes all underlying appenders
func (a *MultiAppender) Close() error {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var lastErr error
	for _, appender := range a.appenders {
		if err := appender.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// AddAppender adds an appender
func (a *MultiAppender) AddAppender(appender Appender) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.appenders = append(a.appenders, appender)
}

// RemoveAppender removes an appender by name
func (a *MultiAppender) RemoveAppender(name string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	for i, appender := range a.appenders {
		if appender.Name() == name {
			a.appenders = append(a.appenders[:i], a.appenders[i+1:]...)
			break
		}
	}
}
