package ctxlog

import (
	"context"
	"log/slog"
)

type ctxKey int

const loggerCtxKey ctxKey = iota

// NewContext returns a new context with the given logger stored as a value.
func NewContext(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerCtxKey, logger)
}

// GetContextLogger retrieves the logger stored in the context, falling back to slog.Default().
func GetContextLogger(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerCtxKey).(*slog.Logger); ok {
		return logger
	}
	return slog.Default()
}
