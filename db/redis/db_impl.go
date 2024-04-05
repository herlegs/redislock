package redis

import (
	"code.byted.org/kv/redis-v6"
	"context"
	"time"
)

const (
	localHost = "localhost:6379"
)

type redisStoreImpl struct {
	redisClient *redis.Client
}

func NewLocalHostRedisStore() *redisStoreImpl {
	return NewRedisStore(localHost)
}

func NewRedisStore(addr string) *redisStoreImpl {
	redisClient := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &redisStoreImpl{
		redisClient: redisClient,
	}
}

func (r *redisStoreImpl) SetNX(ctx context.Context, key string, value string, expiration time.Duration) (bool, error) {
	return r.redisClient.WithContext(ctx).SetNX(key, value, expiration).Result()
}

func (r *redisStoreImpl) Get(ctx context.Context, key string) (string, error) {
	return r.redisClient.WithContext(ctx).Get(key).Result()
}

func (r *redisStoreImpl) CompareAndSet(ctx context.Context, key string, value string, expiration time.Duration) (renewed bool, err error) {
	const script = `local val = redis.call('GET', KEYS[1]); 
if val == false or val ~=  ARGV[1] then
	return 0 end; 
redis.call('PEXPIRE', KEYS[1], tonumber(ARGV[2]))
return 1;`
	val, err := r.redisClient.WithContext(ctx).Eval(script, []string{key}, value, expiration.Milliseconds()).Result()
	if err != nil {
		return false, err
	}
	renewed = numberString2Bool(val)
	return renewed, nil
}

func (r *redisStoreImpl) CompareAndDelete(ctx context.Context, key string, value string) (released bool, err error) {
	const script = `local val = redis.call('GET', KEYS[1]);
if val == false or val ~=  ARGV[1] then 
	return 0 end;
return redis.call('DEL', KEYS[1]);`
	val, err := r.redisClient.WithContext(ctx).Eval(script, []string{key}, value).Result()
	if err != nil {
		return false, err
	}
	released = numberString2Bool(val)
	return released, nil
}

func numberString2Bool(v interface{}) bool {
	switch v := v.(type) {
	case bool:
		return v
	case string:
		return v == "1"
	case int:
		return v == 1
	case int8:
		return v == 1
	case int16:
		return v == 1
	case int32:
		return v == 1
	case int64:
		return v == 1
	case uint:
		return v == 1
	case uint8:
		return v == 1
	case uint16:
		return v == 1
	case uint32:
		return v == 1
	case uint64:
		return v == 1
	}
	return false
}
