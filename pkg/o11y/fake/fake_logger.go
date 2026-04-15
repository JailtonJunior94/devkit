package fake

import (
	"context"
	"sync"
	"time"

	observability "devkit/pkg/o11y"
)

// FakeLogger accumulates log entries in memory for assertions.
type FakeLogger struct {
	mu      *sync.RWMutex
	entries *[]LogEntry
	fields  []observability.Field
}

// NewFakeLogger creates an in-memory logger for tests.
func NewFakeLogger() *FakeLogger {
	entries := make([]LogEntry, 0)
	return &FakeLogger{
		mu:      &sync.RWMutex{},
		entries: &entries,
		fields:  make([]observability.Field, 0),
	}
}

// Debug records a debug-level log entry.
func (l *FakeLogger) Debug(ctx context.Context, msg string, fields ...observability.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()
	*l.entries = append(*l.entries, LogEntry{
		Level:     observability.LogLevelDebug,
		Message:   msg,
		Fields:    mergeFields(l.fields, fields),
		Timestamp: time.Now(),
	})
}

// Info records an info-level log entry.
func (l *FakeLogger) Info(ctx context.Context, msg string, fields ...observability.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()
	*l.entries = append(*l.entries, LogEntry{
		Level:     observability.LogLevelInfo,
		Message:   msg,
		Fields:    mergeFields(l.fields, fields),
		Timestamp: time.Now(),
	})
}

// Warn records a warn-level log entry.
func (l *FakeLogger) Warn(ctx context.Context, msg string, fields ...observability.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()
	*l.entries = append(*l.entries, LogEntry{
		Level:     observability.LogLevelWarn,
		Message:   msg,
		Fields:    mergeFields(l.fields, fields),
		Timestamp: time.Now(),
	})
}

// Error records an error-level log entry.
func (l *FakeLogger) Error(ctx context.Context, msg string, fields ...observability.Field) {
	l.mu.Lock()
	defer l.mu.Unlock()
	*l.entries = append(*l.entries, LogEntry{
		Level:     observability.LogLevelError,
		Message:   msg,
		Fields:    mergeFields(l.fields, fields),
		Timestamp: time.Now(),
	})
}

// With returns a child logger that includes the given fields in every entry.
func (l *FakeLogger) With(fields ...observability.Field) observability.Logger {
	return &FakeLogger{
		mu:      l.mu,
		entries: l.entries,
		fields:  append(cloneFakeFields(l.fields), fields...),
	}
}

// GetEntries returns a copy of the captured log entries.
func (l *FakeLogger) GetEntries() []LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]LogEntry, len(*l.entries))
	for i, entry := range *l.entries {
		result[i] = LogEntry{
			Level:     entry.Level,
			Message:   entry.Message,
			Fields:    cloneFakeFields(entry.Fields),
			Timestamp: entry.Timestamp,
		}
	}
	return result
}

// Reset clears the captured log entries.
func (l *FakeLogger) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	*l.entries = make([]LogEntry, 0)
}

// LogEntry represents a captured log entry.
type LogEntry struct {
	Level     observability.LogLevel
	Message   string
	Fields    []observability.Field
	Timestamp time.Time
}
