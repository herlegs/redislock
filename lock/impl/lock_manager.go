package impl

import (
	"context"
	"github.com/yuguang.xiao/redislock/db"
	"github.com/yuguang.xiao/redislock/lock"
	"time"
)

var defaultSetting = Setting{
	retryMinInterval: time.Millisecond * 15,
	retryMaxInterval: time.Millisecond * 150,
	// will set to be 1/3 of Lease time
	keepAliveInterval: 0,
	Lease:             time.Second * 3,
	MaxLockTime:       time.Second * 10,
	MaxWaitTime:       time.Second * 10,
	LockKeyPrefix:     "dblock:",
}

type Setting struct {
	retryMinInterval  time.Duration
	retryMaxInterval  time.Duration
	keepAliveInterval time.Duration
	// expire time for lock
	Lease time.Duration
	// lock's max lifecycle time
	MaxLockTime time.Duration
	// max time waiting for Lock()
	MaxWaitTime   time.Duration
	LockKeyPrefix string
}

type Option func(setting *Setting)

// min interval for exponential backoff retrying (Lock)
func WithMinRetryInterval(t time.Duration) Option {
	return func(setting *Setting) {
		setting.retryMinInterval = t
	}
}

// max interval for exponential backoff retrying (Lock)
func WithMaxRetryInterval(t time.Duration) Option {
	return func(setting *Setting) {
		setting.retryMaxInterval = t
	}
}

// set expire time for lock
func WithLease(t time.Duration) Option {
	return func(setting *Setting) {
		setting.Lease = t
	}
}

// set lock's max lifecycle time
func WithMaxLockTime(t time.Duration) Option {
	return func(setting *Setting) {
		setting.MaxLockTime = t
	}
}

// set max waiting time for Lock()
func WithMaxWaitTime(t time.Duration) Option {
	return func(setting *Setting) {
		setting.MaxWaitTime = t
	}
}

func WithLockKeyPrefix(prefix string) Option {
	return func(setting *Setting) {
		setting.LockKeyPrefix = prefix
	}
}

func (s *Setting) Validate() error {
	if s.Lease <= 0 {
		return SettingErrorLeaseInvalid
	}
	if s.MaxLockTime < s.Lease {
		return SettingErrorMaxLockTimeInvalid
	}
	if s.retryMinInterval <= 0 {
		return SettingErrorRetryMinIntervalInvalid
	}
	if s.retryMaxInterval < s.retryMinInterval {
		return SettingErrorRetryMaxIntervalInvalid
	}
	return nil
}

type lockManagerImpl struct {
	dbClient db.Store
	setting  *Setting
}

func NewLockManager(dbClient db.Store, options ...Option) (lock.LockManager, error) {
	setting := defaultSetting
	for _, option := range options {
		option(&setting)
	}
	setting.keepAliveInterval = setting.Lease / 3
	if err := setting.Validate(); err != nil {
		return nil, err
	}

	return &lockManagerImpl{
		dbClient: dbClient,
		setting:  &setting,
	}, nil
}

func (l *lockManagerImpl) lock(ctx context.Context, key string, retry bool) (lock.Locker, error) {
	locker := newLocker(l, key, newRandID())
	locked, err := locker.lockWithRetry(ctx, time.Now(), retry)
	if locked {
		go locker.startKeepAlive(ctx)
		return locker, nil
	}
	poolReturnLocker(locker)
	return nil, err
}

func (l *lockManagerImpl) Lock(ctx context.Context, key string) (lock.Locker, error) {
	//start := time.Now()
	//defer func() {
	//	fmt.Printf("lock used: %v\n", time.Since(start))
	//}()
	return l.lock(ctx, key, true)
}

func (l *lockManagerImpl) TryLock(ctx context.Context, key string) (lock.Locker, error) {
	return l.lock(ctx, key, false)
}
