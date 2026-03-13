package collector

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	"github.com/LambdaLabs/redfish_exporter/internal/ctxlog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// capturingHandler is an slog.Handler that collects log records for inspection.
type capturingHandler struct {
	mu      sync.Mutex
	records []slog.Record
}

func (h *capturingHandler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *capturingHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.records = append(h.records, r)
	return nil
}

func (h *capturingHandler) WithAttrs(_ []slog.Attr) slog.Handler { return h }
func (h *capturingHandler) WithGroup(_ string) slog.Handler      { return h }

func (h *capturingHandler) HasMessage(substr string) bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, r := range h.records {
		if strings.Contains(r.Message, substr) {
			return true
		}
	}
	return false
}

func TestRecoverGroup(t *testing.T) {
	tests := []struct {
		name          string
		fn            func() error
		wantErr       bool
		wantErrSubstr string
		wantLogMsg    string
	}{
		{
			name: "normal function returns nil error",
			fn:   func() error { return nil },
		},
		{
			name:          "panics with string returns error containing panic value",
			fn:            func() error { panic("something went wrong") },
			wantErr:       true,
			wantErrSubstr: "something went wrong",
			wantLogMsg:    "panic recovered",
		},
		{
			name:          "panics with error returns error containing panic value",
			fn:            func() error { panic(errors.New("wrapped panic error")) },
			wantErr:       true,
			wantErrSubstr: "wrapped panic error",
			wantLogMsg:    "panic recovered",
		},
		{
			name:          "panics with integer returns error containing panic value",
			fn:            func() error { panic(42) },
			wantErr:       true,
			wantErrSubstr: "42",
			wantLogMsg:    "panic recovered",
		},
		{
			name:          "context logger is used on panic",
			fn:            func() error { panic("ctx panic value") },
			wantErr:       true,
			wantErrSubstr: "ctx panic value",
			wantLogMsg:    "panic recovered",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			handler := &capturingHandler{}
			ctx := ctxlog.NewContext(context.Background(), slog.New(handler))

			eg := newRecoverGroup(ctx)
			eg.Go(tc.fn)
			err := eg.Wait()

			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErrSubstr)
			} else {
				assert.NoError(t, err)
			}
			if tc.wantLogMsg != "" {
				assert.True(t, handler.HasMessage(tc.wantLogMsg))
			}
		})
	}
}
