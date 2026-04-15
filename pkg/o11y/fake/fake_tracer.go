package fake

import (
	"context"
	"sync"
	"time"

	observability "devkit/pkg/o11y"
)

type fakeSpanContextKey struct{}

// FakeTracer stores spans created during a test run.
type FakeTracer struct {
	mu    sync.RWMutex
	spans []*FakeSpan
}

// NewFakeTracer creates an in-memory tracer for tests.
func NewFakeTracer() *FakeTracer {
	return &FakeTracer{
		spans: make([]*FakeSpan, 0),
	}
}

// Start creates and records a new span, injecting it into the returned context.
func (t *FakeTracer) Start(ctx context.Context, spanName string, opts ...observability.SpanOption) (context.Context, observability.Span) {
	ctx = ensureContext(ctx)
	config := observability.NewSpanConfig(opts)

	span := &FakeSpan{
		Name:       spanName,
		StartTime:  time.Now(),
		Attributes: config.Attributes(),
		Events:     make([]FakeEvent, 0),
	}

	t.mu.Lock()
	t.spans = append(t.spans, span)
	t.mu.Unlock()

	return context.WithValue(ctx, fakeSpanContextKey{}, span), span
}

// SpanFromContext retrieves the span previously injected into ctx.
func (t *FakeTracer) SpanFromContext(ctx context.Context) observability.Span {
	ctx = ensureContext(ctx)
	if span, ok := ctx.Value(fakeSpanContextKey{}).(observability.Span); ok && span != nil {
		return span
	}

	return &FakeSpan{}
}

// ContextWithSpan injects span into ctx for later retrieval.
func (t *FakeTracer) ContextWithSpan(ctx context.Context, span observability.Span) context.Context {
	ctx = ensureContext(ctx)
	if span == nil {
		return ctx
	}

	return context.WithValue(ctx, fakeSpanContextKey{}, span)
}

// GetSpans returns a copy of the captured spans.
func (t *FakeTracer) GetSpans() []*FakeSpan {
	t.mu.RLock()
	defer t.mu.RUnlock()
	result := make([]*FakeSpan, len(t.spans))
	for i, span := range t.spans {
		result[i] = span.clone()
	}
	return result
}

// Reset clears the captured spans.
func (t *FakeTracer) Reset() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.spans = make([]*FakeSpan, 0)
}

// FakeSpan represents a span recorded in memory.
type FakeSpan struct {
	mu          sync.RWMutex
	Name        string
	StartTime   time.Time
	EndTime     *time.Time
	Attributes  []observability.Field
	Events      []FakeEvent
	Status      observability.StatusCode
	StatusDesc  string
	RecordedErr error
}

// End marks the span as finished.
func (s *FakeSpan) End() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	s.EndTime = &now
}

// SetAttributes appends structured attributes to the span.
func (s *FakeSpan) SetAttributes(fields ...observability.Field) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Attributes = append(s.Attributes, fields...)
}

// SetStatus records the final status on the span.
func (s *FakeSpan) SetStatus(code observability.StatusCode, description string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = code
	s.StatusDesc = description
}

// RecordError records an error and optional attributes on the span.
func (s *FakeSpan) RecordError(err error, fields ...observability.Field) {
	if isNilError(err) {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.RecordedErr = err
	s.Attributes = append(s.Attributes, cloneFakeFields(fields)...)
}

// AddEvent records a named event with optional attributes.
func (s *FakeSpan) AddEvent(name string, fields ...observability.Field) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Events = append(s.Events, FakeEvent{
		Name:      name,
		Timestamp: time.Now(),
		Fields:    cloneFakeFields(fields),
	})
}

// Context returns a FakeSpanContext with deterministic IDs for assertions.
func (s *FakeSpan) Context() observability.SpanContext {
	return &FakeSpanContext{
		traceID: "fake-trace-id",
		spanID:  "fake-span-id",
		sampled: true,
	}
}

// FakeEvent represents an event recorded on a fake span.
type FakeEvent struct {
	Name      string
	Timestamp time.Time
	Fields    []observability.Field
}

// FakeSpanContext exposes fixed IDs to simplify assertions.
type FakeSpanContext struct {
	traceID string
	spanID  string
	sampled bool
}

// TraceID returns the deterministic trace identifier.
func (c *FakeSpanContext) TraceID() string {
	return c.traceID
}

// SpanID returns the deterministic span identifier.
func (c *FakeSpanContext) SpanID() string {
	return c.spanID
}

// IsSampled always returns true for the fake span context.
func (c *FakeSpanContext) IsSampled() bool {
	return c.sampled
}

func (s *FakeSpan) clone() *FakeSpan {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cloned := &FakeSpan{
		Name:       s.Name,
		StartTime:  s.StartTime,
		Attributes: cloneFakeFields(s.Attributes),
		Events:     cloneFakeEvents(s.Events),
		Status:     s.Status,
		StatusDesc: s.StatusDesc,
	}
	if s.EndTime != nil {
		end := *s.EndTime
		cloned.EndTime = &end
	}
	if !isNilError(s.RecordedErr) {
		cloned.RecordedErr = s.RecordedErr
	}

	return cloned
}

func cloneFakeEvents(events []FakeEvent) []FakeEvent {
	if len(events) == 0 {
		return nil
	}

	cloned := make([]FakeEvent, len(events))
	for i, event := range events {
		cloned[i] = FakeEvent{
			Name:      event.Name,
			Timestamp: event.Timestamp,
			Fields:    cloneFakeFields(event.Fields),
		}
	}

	return cloned
}
