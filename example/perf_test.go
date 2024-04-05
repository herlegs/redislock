package example

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/require"
	"github.com/yuguang.xiao/redislock/db/redis"
	"github.com/yuguang.xiao/redislock/lock"
	"github.com/yuguang.xiao/redislock/lock/impl"
	"sync"
	"testing"
	"time"
)

func workerDoJob(lockManager lock.LockManager, numOfThread int, jobTime time.Duration) {
	var unprotected int64
	adder := func() {
		unprotected = unprotected + 1
		if jobTime > 0 {
			time.Sleep(jobTime)
		}
	}
	wg := sync.WaitGroup{}
	lockKey := fmt.Sprint("workerDoJob")
	start := time.Now()
	for i := 0; i < numOfThread; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			locker, err := lockManager.Lock(context.Background(), lockKey)
			if err != nil || locker == nil {
				fmt.Printf("err:%v\n", err)
				return
			}
			adder()
			_, _ = locker.Release(context.Background())
		}()
	}
	wg.Wait()
	fmt.Printf("total time use for %v jobs (job time: %v): %v\n", numOfThread, jobTime, time.Since(start))
}

func TestConcurrentSimpleJob(t *testing.T) {
	redisStore := redis.NewLocalHostRedisStore()
	lockManager, err := impl.NewLockManager(redisStore)
	require.NoError(t, err)
	workerDoJob(lockManager, 10000, 0)
}

func BenchmarkComplexJob(b *testing.B) {
	redisStore := redis.NewLocalHostRedisStore()
	lockManager, err := impl.NewLockManager(redisStore,
		impl.WithMinRetryInterval(time.Millisecond*10),
		impl.WithMaxRetryInterval(time.Millisecond*100))
	require.NoError(b, err)
	for i := 0; i < b.N; i++ {
		workerDoJob(lockManager, 1000, time.Millisecond*5)
	}
}
