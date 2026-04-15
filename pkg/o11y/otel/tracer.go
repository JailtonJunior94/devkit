package otel

import (
	"context"

	observability "devkit/pkg/o11y"
	"go.opentelemetry.io/otel/codes"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type otelTracer struct {
	tracer oteltrace.Tracer
}

func newOtelTracer(tracer oteltrace.Tracer) *otelTracer {
	return &otelTracer{tracer: tracer}
}

func (t *otelTracer) Start(ctx context.Context, spanName string, opts ...observability.SpanOption) (context.Context, observability.Span) {
	ctx = normalizeContext(ctx)
	cfg := observability.NewSpanConfig(opts)

	initialCap := 1
	if len(cfg.Attributes()) > 0 {
		initialCap++
	}

	otelOpts := make([]oteltrace.SpanStartOption, 0, initialCap)
	otelOpts = append(otelOpts, oteltrace.WithSpanKind(convertSpanKind(cfg.Kind())))

	attrs := convertFieldsToAttributes(cfg.Attributes())
	if attrs != nil {
		otelOpts = append(otelOpts, oteltrace.WithAttributes(attrs...))
	}

	ctx, otelSpan := t.tracer.Start(ctx, spanName, otelOpts...)
	return ctx, &otelSpanImpl{span: otelSpan}
}

// When there is no active span, OTel returns a non-recording span.
func (t *otelTracer) SpanFromContext(ctx context.Context) observability.Span {
	ctx = normalizeContext(ctx)
	span := oteltrace.SpanFromContext(ctx)
	return &otelSpanImpl{span: span}
}

func (t *otelTracer) ContextWithSpan(ctx context.Context, span observability.Span) context.Context {
	ctx = normalizeContext(ctx)
	otelSpan, ok := span.(*otelSpanImpl)
	if !ok {
		return ctx
	}

	return oteltrace.ContextWithSpan(ctx, otelSpan.span)
}

type otelSpanImpl struct {
	span oteltrace.Span
}

func (s *otelSpanImpl) End() {
	s.span.End()
}

func (s *otelSpanImpl) SetAttributes(fields ...observability.Field) {
	attrs := convertFieldsToAttributes(fields)
	if attrs == nil {
		return
	}

	s.span.SetAttributes(attrs...)
}

func (s *otelSpanImpl) SetStatus(code observability.StatusCode, description string) {
	s.span.SetStatus(convertStatusCode(code), description)
}

func (s *otelSpanImpl) RecordError(err error, fields ...observability.Field) {
	if isNilInterfaceValue(err) {
		return
	}

	attrs := convertFieldsToAttributes(fields)
	if attrs == nil {
		s.span.RecordError(err)
		return
	}

	s.span.RecordError(err, oteltrace.WithAttributes(attrs...))
}

func (s *otelSpanImpl) AddEvent(name string, fields ...observability.Field) {
	attrs := convertFieldsToAttributes(fields)
	if attrs == nil {
		s.span.AddEvent(name)
		return
	}

	s.span.AddEvent(name, oteltrace.WithAttributes(attrs...))
}

func (s *otelSpanImpl) Context() observability.SpanContext {
	return &otelSpanContext{ctx: s.span.SpanContext()}
}

type otelSpanContext struct {
	ctx oteltrace.SpanContext
}

func (c *otelSpanContext) TraceID() string {
	return c.ctx.TraceID().String()
}

func (c *otelSpanContext) SpanID() string {
	return c.ctx.SpanID().String()
}

func (c *otelSpanContext) IsSampled() bool {
	return c.ctx.IsSampled()
}

func convertSpanKind(kind observability.SpanKind) oteltrace.SpanKind {
	switch kind {
	case observability.SpanKindInternal:
		return oteltrace.SpanKindInternal
	case observability.SpanKindServer:
		return oteltrace.SpanKindServer
	case observability.SpanKindClient:
		return oteltrace.SpanKindClient
	case observability.SpanKindProducer:
		return oteltrace.SpanKindProducer
	case observability.SpanKindConsumer:
		return oteltrace.SpanKindConsumer
	default:
		return oteltrace.SpanKindInternal
	}
}

func convertStatusCode(code observability.StatusCode) codes.Code {
	switch code {
	case observability.StatusCodeOK:
		return codes.Ok
	case observability.StatusCodeError:
		return codes.Error
	default:
		return codes.Unset
	}
}
