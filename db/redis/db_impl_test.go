package redis

import (
	"code.byted.org/kv/redis-v6"
	"context"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestLocalRedis(t *testing.T) {
	key := "test"
	val := "val"

	ctx := context.Background()
	db := NewLocalHostRedisStore()
	res, err := db.redisClient.Set(key, val, 0).Result()
	require.NoError(t, err)
	res, err = db.Get(ctx, key)
	require.Equal(t, val, res)
	require.NoError(t, err)

	renewed, err := db.CompareAndSet(ctx, key, val, time.Millisecond*100)
	require.NoError(t, err)
	require.True(t, renewed)
	time.Sleep(time.Millisecond * 200)

	res, err = db.Get(ctx, key)
	require.Equal(t, redis.Nil, err)

	res, err = db.redisClient.Set(key, val, 0).Result()
	require.NoError(t, err)
	released, err := db.CompareAndDelete(ctx, key, val)
	require.NoError(t, err)
	require.True(t, released)

	res, err = db.Get(ctx, key)
	require.Equal(t, redis.Nil, err)
}
