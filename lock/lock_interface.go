package lock

import "context"

// LockManager is distributed lock system based on database
// for detailed config, please refer to available Option
type LockManager interface {
	// Lock is blocking getting lock, wait time can be configured
	Lock(ctx context.Context, key string) (Locker, error)
	// TryLock is non-blocking, will directly return if didn't get lock
	TryLock(ctx context.Context, key string) (Locker, error)
}

// Locker is the result of Lock or TryLock, providing Release API
type Locker interface {
	Release(ctx context.Context) (released bool, err error)
	HasLock(ctx context.Context) bool
}
