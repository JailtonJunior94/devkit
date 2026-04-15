package logging

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
)

type stubHandler struct {
	enabled bool
	err     error
	records []slog.Record
}

func (h *stubHandler) Enabled(context.Context, slog.Level) bool { return h.enabled }

func (h *stubHandler) Handle(_ context.Context, record slog.Record) error {
	h.records = append(h.records, record.Clone())
	return h.err
}

func (h *stubHandler) WithAttrs([]slog.Attr) slog.Handler { return h }
func (h *stubHandler) WithGroup(string) slog.Handler      { return h }

func TestNewMultiHandlerFiltersNilHandlers(t *testing.T) {
	t.Parallel()

	handler := &stubHandler{enabled: true}
	got := newMultiHandler(nil, handler, nil)
	if got != handler {
		t.Fatalf("newMultiHandler() = %T, want original handler", got)
	}
}

func TestMultiHandlerEnabledReturnsTrueWhenAnyChildIsEnabled(t *testing.T) {
	t.Parallel()

	handler := newMultiHandler(
		&stubHandler{enabled: false},
		&stubHandler{enabled: true},
	)

	if !handler.Enabled(context.Background(), slog.LevelInfo) {
		t.Fatal("Enabled() = false, want true")
	}
}

func TestMultiHandlerHandleClonesRecordAndJoinsErrors(t *testing.T) {
	t.Parallel()

	first := &stubHandler{enabled: true, err: errors.New("first")}
	second := &stubHandler{enabled: true}
	handler := newMultiHandler(first, second)

	record := slog.NewRecord(time.Now(), slog.LevelInfo, "msg", 0)

	err := handler.Handle(context.Background(), record)
	if err == nil || err.Error() != "first" {
		t.Fatalf("Handle() error = %v, want first", err)
	}
	if len(first.records) != 1 || len(second.records) != 1 {
		t.Fatalf("records handled = %d/%d, want 1/1", len(first.records), len(second.records))
	}
	if record.Message != "msg" {
		t.Fatalf("original record mutated to %q", record.Message)
	}
}
