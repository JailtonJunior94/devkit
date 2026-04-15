package oteltest

import (
	"context"
	"log/slog"
	"sync"
)

// FakeLogger provides an in-memory slog handler for test log inspection.
type FakeLogger struct {
	root   *fakeLogRoot
	logger *slog.Logger
}

type fakeLogRoot struct {
	mu      sync.Mutex
	records []slog.Record
}

type fakeHandler struct {
	root        *fakeLogRoot
	topAttrs    []slog.Attr
	attrs       []slog.Attr
	groupPrefix string
}

func (h *fakeHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *fakeHandler) Handle(_ context.Context, r slog.Record) error {
	h.root.mu.Lock()
	defer h.root.mu.Unlock()

	out := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)
	if len(h.topAttrs) > 0 {
		out.AddAttrs(h.topAttrs...)
	}

	if h.groupPrefix == "" {
		if len(h.attrs) > 0 {
			out.AddAttrs(h.attrs...)
		}
		r.Attrs(func(a slog.Attr) bool {
			out.AddAttrs(a)
			return true
		})
	} else {
		grouped := append([]slog.Attr(nil), h.attrs...)
		r.Attrs(func(a slog.Attr) bool {
			grouped = append(grouped, a)
			return true
		})
		if len(grouped) > 0 {
			out.AddAttrs(slog.Group(h.groupPrefix, attrsToAny(grouped)...))
		}
	}

	h.root.records = append(h.root.records, out)
	return nil
}

func (h *fakeHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	merged := make([]slog.Attr, len(h.attrs)+len(attrs))
	copy(merged, h.attrs)
	copy(merged[len(h.attrs):], attrs)
	return &fakeHandler{root: h.root, topAttrs: h.topAttrs, attrs: merged, groupPrefix: h.groupPrefix}
}

func (h *fakeHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	newTop := make([]slog.Attr, len(h.topAttrs)+len(h.attrs))
	copy(newTop, h.topAttrs)
	copy(newTop[len(h.topAttrs):], h.attrs)

	prefix := name
	if h.groupPrefix != "" {
		prefix = h.groupPrefix + "." + name
	}
	return &fakeHandler{root: h.root, topAttrs: newTop, groupPrefix: prefix}
}

func attrsToAny(attrs []slog.Attr) []any {
	result := make([]any, len(attrs))
	for i, a := range attrs {
		result[i] = a
	}
	return result
}

// NewFakeLogger creates a FakeLogger that collects log records in memory.
func NewFakeLogger() *FakeLogger {
	root := &fakeLogRoot{}
	h := &fakeHandler{root: root}
	return &FakeLogger{
		root:   root,
		logger: slog.New(h),
	}
}

// Logger returns the slog logger that writes to memory.
func (f *FakeLogger) Logger() *slog.Logger {
	return f.logger
}

// Records returns a copy of all collected log records.
func (f *FakeLogger) Records() []slog.Record {
	f.root.mu.Lock()
	defer f.root.mu.Unlock()
	result := make([]slog.Record, len(f.root.records))
	copy(result, f.root.records)
	return result
}

// Reset clears collected records.
func (f *FakeLogger) Reset() {
	f.root.mu.Lock()
	defer f.root.mu.Unlock()
	f.root.records = f.root.records[:0]
}
