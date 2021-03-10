package ctxutils

import (
	"context"
	"github.com/Prepodavan/goca/internal/models"
)

const keyRequestID = "request_id"

func WithRequestID(ctx context.Context, rid models.RequestID) context.Context {
	return context.WithValue(ctx, keyRequestID, rid)
}

func MustRequestID(ctx context.Context) models.RequestID {
	return ctx.Value(keyRequestID).(models.RequestID)
}
