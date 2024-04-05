package example

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/yuguang.xiao/redislock/db/redis"
	"github.com/yuguang.xiao/redislock/lock/impl"
	"sync"
	"testing"
	"time"
)

/*
基本场景，多个线程同时写一个resource，能保证线程安全以及正确性
*/
func TestMultiReadWriteOnCommonVar(t *testing.T) {
	var unprotected int64

	adder := func() {
		unprotected = unprotected + 1
	}
	// firstly, test if without lock:
	unprotected = 0
	var wg sync.WaitGroup
	numOfThread := 1000
	for i := 0; i < numOfThread; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			adder()
		}()
	}
	wg.Wait()
	fmt.Printf("unprotected: %v, should be: %v\n", unprotected, numOfThread)
	require.NotEqual(t, int64(numOfThread), unprotected)
	fmt.Printf("but without lock it's not equal\n")

	// then test with lock
	redisStore := redis.NewLocalHostRedisStore()
	lockManager, err := impl.NewLockManager(redisStore)
	require.NoError(t, err)

	unprotected = 0
	lockKey := "TestMultiReadWriteOnCommonVar"
	for i := 0; i < numOfThread; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			locker, err := lockManager.Lock(context.Background(), lockKey)
			if err != nil || locker == nil {
				fmt.Printf("err:%v\n", err)
				return
			}
			require.NoError(t, err)
			adder()
			released, err := locker.Release(context.Background())
			require.NoError(t, err)
			require.True(t, released)
		}()
	}
	wg.Wait()

	fmt.Printf("unprotected: %v, should be: %v\n", unprotected, numOfThread)
	require.Equal(t, int64(numOfThread), unprotected)
	fmt.Printf("now with lock it's equal")
}

func TestLockAndTryLock(t *testing.T) {
	key := "TestLockAndTryLock"
	leaseTime := time.Millisecond * 50
	maxLockTime := time.Millisecond * 150
	redisStore := redis.NewLocalHostRedisStore()
	lockManager, err := impl.NewLockManager(redisStore,
		impl.WithLease(leaseTime),
		impl.WithMaxLockTime(maxLockTime))
	require.NoError(t, err)

	lockA, err := lockManager.Lock(context.Background(), key)
	_ = lockA

	start := time.Now()
	tryLocker, err := lockManager.TryLock(context.Background(), key)
	fmt.Printf("try lock time: %v\n", time.Since(start))
	require.Nil(t, tryLocker)

	start = time.Now()
	locker, err := lockManager.Lock(context.Background(), key)
	fmt.Printf("lock time: %v\n", time.Since(start))
	require.NotNil(t, locker)
}

/*
任务自动续期, 但不超过maxLockTime 防止死锁
*/
func TestLeaseAndDeadLock(t *testing.T) {
	key := "TestLeaseAndDeadLock"
	leaseTime := time.Millisecond * 500
	maxLockTime := leaseTime * 3
	redisStore := redis.NewLocalHostRedisStore()
	lockManager, err := impl.NewLockManager(redisStore,
		impl.WithLease(leaseTime),
		impl.WithMaxLockTime(maxLockTime))
	require.NoError(t, err)

	deadLocker, err := lockManager.Lock(context.Background(), key)
	require.NoError(t, err)
	_ = deadLocker
	stillHasLock := deadLocker.HasLock(context.Background())
	require.True(t, stillHasLock)
	<-time.After(leaseTime * 2)
	stillHasLock = deadLocker.HasLock(context.Background())
	require.True(t, stillHasLock)
	<-time.After(maxLockTime)
	stillHasLock = deadLocker.HasLock(context.Background())
	require.False(t, stillHasLock)
}

func TestUUIDRelease(t *testing.T) {
	key := "TestUUIDRelease"
	leaseTime := time.Millisecond * 50
	maxLockTime := time.Millisecond * 150
	redisStore := redis.NewLocalHostRedisStore()
	lockManager, err := impl.NewLockManager(redisStore,
		impl.WithLease(leaseTime),
		impl.WithMaxLockTime(maxLockTime))
	require.NoError(t, err)

	threadA, err := lockManager.Lock(context.Background(), key)
	require.NoError(t, err)
	_ = threadA
	time.Sleep(leaseTime * 2)

	threadB, err := lockManager.Lock(context.Background(), key)
	require.NoError(t, err)

	released, err := threadA.Release(context.Background())
	require.False(t, released)
	require.Error(t, err)

	released, err = threadB.Release(context.Background())
	require.True(t, released)
	require.NoError(t, err)
}
