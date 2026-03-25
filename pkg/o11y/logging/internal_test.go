package logging

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
)

func TestWithHandler_setsHandler(t *testing.T) {
	t.Parallel()

	handler := newStubHandler(true)
	cfg := Config{ServiceName: "svc"}

	if err := WithHandler(handler)(context.Background(), &cfg); err != nil {
		t.Fatalf("WithHandler() error = %v", err)
	}
	if cfg.Handler != handler {
		t.Fatal("WithHandler() did not store the provided handler")
	}
}

func TestNewMultiHandler_filtersNil(t *testing.T) {
	t.Parallel()

	handler := newStubHandler(true)
	got := newMultiHandler(nil, handler, nil)
	if got != handler {
		t.Fatalf("newMultiHandler() = %T, want original handler", got)
	}
}

func TestMultiHandlerEnabled_reportsTrueWhenAnyHandlerAccepts(t *testing.T) {
	t.Parallel()

	disabled := newStubHandler(false)
	enabled := newStubHandler(true)
	handler := newMultiHandler(disabled, enabled)

	if !handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("Enabled() = false, want true")
	}
}

func TestMultiHandlerHandle_joinsErrorsAndClonesRecord(t *testing.T) {
	t.Parallel()

	first := newStubHandler(true)
	first.err = errors.New("first")
	second := newStubHandler(true)
	second.mutateMessage = "mutated"
	handler := newMultiHandler(first, second)
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "original", 0)

	err := handler.Handle(context.Background(), record)
	if err == nil || err.Error() != "first" {
		t.Fatalf("Handle() error = %v, want first", err)
	}
	if first.root.handled[0].Message != "original" {
		t.Fatalf("first handler message = %q, want original", first.root.handled[0].Message)
	}
	if second.root.handled[0].Message != "mutated" {
		t.Fatalf("second handler message = %q, want mutated", second.root.handled[0].Message)
	}
	if record.Message != "original" {
		t.Fatalf("original record message = %q, want original", record.Message)
	}
}

func TestMultiHandlerWithAttrsAndGroup_delegatesToChildren(t *testing.T) {
	t.Parallel()

	first := newStubHandler(true)
	second := newStubHandler(true)
	handler := newMultiHandler(first, second)

	withAttrs := handler.WithAttrs([]slog.Attr{slog.String("k", "v")})
	withGroup := withAttrs.WithGroup("group")
	record := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)

	if err := withGroup.Handle(context.Background(), record); err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	derived, ok := withGroup.(*multiHandler)
	if !ok {
		t.Fatalf("WithGroup() returned %T, want *multiHandler", withGroup)
	}
	firstDerived, ok := derived.handlers[0].(*stubHandler)
	if !ok {
		t.Fatalf("first derived handler = %T, want *stubHandler", derived.handlers[0])
	}
	secondDerived, ok := derived.handlers[1].(*stubHandler)
	if !ok {
		t.Fatalf("second derived handler = %T, want *stubHandler", derived.handlers[1])
	}
	if len(firstDerived.root.attrs) != 1 || firstDerived.root.attrs[0].Key != "k" {
		t.Fatalf("first attrs = %+v, want key k", firstDerived.root.attrs)
	}
	if len(secondDerived.root.attrs) != 1 || secondDerived.root.attrs[0].Key != "k" {
		t.Fatalf("second attrs = %+v, want key k", secondDerived.root.attrs)
	}
	if firstDerived.root.group != "group" || secondDerived.root.group != "group" {
		t.Fatalf("groups = %q/%q, want group", firstDerived.root.group, secondDerived.root.group)
	}
}

type stubHandler struct {
	root          *stubHandlerRoot
	enabled       bool
	err           error
	mutateMessage string
}

type stubHandlerRoot struct {
	handled []slog.Record
	attrs   []slog.Attr
	group   string
}

func newStubHandler(enabled bool) *stubHandler {
	return &stubHandler{
		root:    &stubHandlerRoot{},
		enabled: enabled,
	}
}

func (h *stubHandler) Enabled(context.Context, slog.Level) bool {
	return h.enabled
}

func (h *stubHandler) Handle(_ context.Context, record slog.Record) error {
	if h.mutateMessage != "" {
		record.Message = h.mutateMessage
	}
	h.root.handled = append(h.root.handled, record.Clone())
	return h.err
}

func (h *stubHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	next := *h
	next.root = &stubHandlerRoot{
		handled: h.root.handled,
		attrs:   append(append([]slog.Attr(nil), h.root.attrs...), attrs...),
		group:   h.root.group,
	}
	return &next
}

func (h *stubHandler) WithGroup(name string) slog.Handler {
	next := *h
	next.root = &stubHandlerRoot{
		handled: h.root.handled,
		attrs:   append([]slog.Attr(nil), h.root.attrs...),
		group:   name,
	}
	return &next
}
