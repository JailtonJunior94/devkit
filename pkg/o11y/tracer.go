package o11y

import "context"

// Tracer creates spans and propagates trace context.
type Tracer interface {
	Start(ctx context.Context, spanName string, opts ...SpanOption) (context.Context, Span)
	SpanFromContext(ctx context.Context) Span
	ContextWithSpan(ctx context.Context, span Span) context.Context
}
