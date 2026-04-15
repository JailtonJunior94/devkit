package o11y

// Signals groups the tracing, logging, and metrics contracts exposed by the
// simplified observability API.
type Signals interface {
	Tracer() Tracer
	Logger() Logger
	Metrics() Metrics
}

// Field represents a structured attribute shared by logs, spans, and metrics.
type Field struct {
	Key   string
	Value any
}

// String creates a string field.
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

// Int creates an int field.
func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

// Int64 creates an int64 field.
func Int64(key string, value int64) Field {
	return Field{Key: key, Value: value}
}

// Float64 creates a float64 field.
func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

// Bool creates a bool field.
func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

// Error creates a field under the conventional key `error`.
func Error(err error) Field {
	return Field{Key: "error", Value: err}
}

// Any creates a field with an arbitrary value.
func Any(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// SpanContext exposes the trace identifiers needed for propagation.
type SpanContext interface {
	TraceID() string
	SpanID() string
	IsSampled() bool
}

// Span represents an instrumented unit of work.
type Span interface {
	End()
	SetAttributes(fields ...Field)
	SetStatus(code StatusCode, description string)
	RecordError(err error, fields ...Field)
	AddEvent(name string, fields ...Field)
	Context() SpanContext
}

// StatusCode represents the final status recorded on a span.
type StatusCode int

const (
	StatusCodeUnset StatusCode = iota
	StatusCodeOK
	StatusCodeError
)

// SpanKind defines the role a span plays in a distributed trace.
type SpanKind int

const (
	SpanKindInternal SpanKind = iota
	SpanKindServer
	SpanKindClient
	SpanKindProducer
	SpanKindConsumer
)

// SpanOption customizes span creation.
type SpanOption interface {
	apply(*spanConfig)
}

type spanConfig struct {
	kind       SpanKind
	attributes []Field
}

type spanOptionFunc func(*spanConfig)

func (f spanOptionFunc) apply(c *spanConfig) {
	f(c)
}

// WithSpanKind overrides the default span kind for a started span.
func WithSpanKind(kind SpanKind) SpanOption {
	return spanOptionFunc(func(c *spanConfig) {
		c.kind = kind
	})
}

// WithAttributes appends structured attributes to the started span.
func WithAttributes(fields ...Field) SpanOption {
	return spanOptionFunc(func(c *spanConfig) {
		c.attributes = append(c.attributes, fields...)
	})
}

// NewSpanConfig resolves a slice of span options into a read-only span config.
func NewSpanConfig(opts []SpanOption) SpanConfig {
	cfg := &spanConfig{
		kind:       SpanKindInternal,
		attributes: make([]Field, 0),
	}
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt.apply(cfg)
	}
	return cfg
}

// SpanConfig exposes the resolved span configuration.
type SpanConfig interface {
	Kind() SpanKind
	Attributes() []Field
}

func (c *spanConfig) Kind() SpanKind {
	return c.kind
}

func (c *spanConfig) Attributes() []Field {
	return cloneFields(c.attributes)
}

func cloneFields(fields []Field) []Field {
	if len(fields) == 0 {
		return nil
	}

	cloned := make([]Field, len(fields))
	copy(cloned, fields)
	return cloned
}
