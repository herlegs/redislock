package example

import (
	"context"
	"github.com/stretchr/testify/require"
	"github.com/yuguang.xiao/redislock/db/redis"
	"github.com/yuguang.xiao/redislock/lock/impl"
	"testing"
)

func TestBasic(t *testing.T) {
	redisStore := redis.NewLocalHostRedisStore()
	lockManager, err := impl.NewLockManager(redisStore)
	require.NoError(t, err)
	locker, err := lockManager.Lock(context.Background(), "a")
	require.NoError(t, err)
	released, err := locker.Release(context.Background())
	require.NoError(t, err)
	require.True(t, released)
}
