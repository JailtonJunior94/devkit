package oteltest

import (
	"context"
	"log/slog"
	"sync"
)

// FakeLogger provides an in-memory slog handler for test log inspection.
// All records are collected in memory regardless of level.
type FakeLogger struct {
	root   *fakeLogRoot
	logger *slog.Logger
}

// fakeLogRoot holds the shared mutable state for all handler branches.
type fakeLogRoot struct {
	mu      sync.Mutex
	records []slog.Record
}

// fakeHandler is the slog.Handler implementation that writes to fakeLogRoot.
//
// topAttrs holds attributes accumulated before the current groupPrefix was
// established (via WithGroup). They are rendered at the outermost scope of
// every record, matching the stdlib slog contract where pre-group attributes
// are not nested inside the group.
//
// attrs holds attributes accumulated after the current groupPrefix was set.
// They are rendered inside the group when groupPrefix is non-empty.
type fakeHandler struct {
	root        *fakeLogRoot
	topAttrs    []slog.Attr // attrs from before the current group; always outer-scope
	attrs       []slog.Attr // attrs at the current group level
	groupPrefix string      // non-empty when inside a WithGroup call
}

func (h *fakeHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *fakeHandler) Handle(_ context.Context, r slog.Record) error {
	h.root.mu.Lock()
	defer h.root.mu.Unlock()

	out := slog.NewRecord(r.Time, r.Level, r.Message, r.PC)

	// topAttrs are always rendered at the outer scope, independent of groupPrefix.
	if len(h.topAttrs) > 0 {
		out.AddAttrs(h.topAttrs...)
	}

	if h.groupPrefix == "" {
		// No active group: stored attrs and record attrs are at the top level.
		if len(h.attrs) > 0 {
			out.AddAttrs(h.attrs...)
		}
		r.Attrs(func(a slog.Attr) bool {
			out.AddAttrs(a)
			return true
		})
	} else {
		// Active group: combine in-group stored attrs with record attrs under the group prefix.
		var grouped []slog.Attr
		grouped = append(grouped, h.attrs...)
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

// WithGroup returns a handler that scopes all future attributes under name.
// Attributes already accumulated via WithAttrs are promoted to topAttrs so they
// continue to appear at the outer scope in every record, matching the slog.Handler
// contract: pre-group attributes must not be nested inside the new group.
func (h *fakeHandler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	// Existing attrs (topAttrs + attrs) become the outer-scope attrs for the new handler.
	newTop := make([]slog.Attr, len(h.topAttrs)+len(h.attrs))
	copy(newTop, h.topAttrs)
	copy(newTop[len(h.topAttrs):], h.attrs)

	prefix := name
	if h.groupPrefix != "" {
		prefix = h.groupPrefix + "." + name
	}
	return &fakeHandler{root: h.root, topAttrs: newTop, groupPrefix: prefix}
}

// attrsToAny converts []slog.Attr to []any for use with slog.Group.
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

// Logger returns the *slog.Logger that writes to memory.
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
