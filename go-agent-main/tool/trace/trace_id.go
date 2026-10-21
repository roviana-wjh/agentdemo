package trace

import (
	"context"
	"crypto/rand"
	"encoding/hex"
)

type traceIDKeyType struct{}

var traceIDKey = traceIDKeyType{}

// GenerateTraceID generates a random 16-byte trace ID.
func GenerateTraceID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback: should not happen in practice
		return "fallback-trace-id"
	}
	return hex.EncodeToString(b)
}

// WithTraceID attaches a trace ID to the context.
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID retrieves the trace ID from context, or generates a new one.
func GetTraceID(ctx context.Context) string {
	if id, ok := ctx.Value(traceIDKey).(string); ok && id != "" {
		return id
	}
	return GenerateTraceID()
}
