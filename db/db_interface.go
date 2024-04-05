package db

import (
	"context"
	"time"
)

type Store interface {
	SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error)

	Get(ctx context.Context, key string) (string, error)

	CompareAndSet(ctx context.Context, key string, value string, expiration time.Duration) (renewed bool, err error)

	CompareAndDelete(ctx context.Context, key string, value string) (released bool, err error)
}
