package ctxlog_test

import (
	"context"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/LambdaLabs/redfish_exporter/internal/ctxlog"
)

func TestGetContextLogger(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(nil, nil))

	tests := []struct {
		name       string
		setupCtx   func() context.Context
		wantLogger *slog.Logger
	}{
		{
			name: "returns stored logger from context",
			setupCtx: func() context.Context {
				return ctxlog.NewContext(context.Background(), logger)
			},
			wantLogger: logger,
		},
		{
			name: "returns slog.Default() when no logger in context",
			setupCtx: func() context.Context {
				return context.Background()
			},
			wantLogger: slog.Default(),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ctxlog.GetContextLogger(tc.setupCtx())
			assert.Same(t, tc.wantLogger, got)
		})
	}
}
