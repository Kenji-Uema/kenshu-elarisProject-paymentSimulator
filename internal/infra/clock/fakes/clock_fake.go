package fakes

import (
	"context"
	"time"
)

type FakeClock struct {
	NowFn func(ctx context.Context) (*time.Time, error)
}

func (f *FakeClock) Now(ctx context.Context) (*time.Time, error) {
	if f.NowFn != nil {
		return f.NowFn(ctx)
	}

	now := time.Now().UTC()
	return &now, nil
}

func (f *FakeClock) Close() error {
	return nil
}
