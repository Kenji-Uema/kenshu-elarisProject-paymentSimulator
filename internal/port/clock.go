package port

import (
	"context"
	"time"
)

type Clock interface {
	Now(ctx context.Context) (*time.Time, error)
	Close() error
}
