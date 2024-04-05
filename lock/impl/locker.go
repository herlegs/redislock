package impl

import (
	"context"
	"fmt"
	"github.com/yuguang.xiao/redislock/db"
	"math/rand"
	"sync/atomic"
	"time"
)

type lockerImpl struct {
	dbClient db.Store
	key      string
	uuid     string
	setting  *Setting
	retryCnt uint64
	// for keep alive routine
	stopChan chan struct{}
	closed   chan struct{}
}

// for re-use
func (l *lockerImpl) reset() {
	l.dbClient = nil
	l.key = ""
	l.uuid = ""
	l.setting = nil
	l.retryCnt = 0
	l.stopChan = nil
	l.closed = nil
}

func newLocker(l *lockManagerImpl, key, uuid string) *lockerImpl {
	locker := poolGetLocker()
	locker.dbClient = l.dbClient
	locker.key = l.setting.LockKeyPrefix + key
	locker.uuid = uuid
	locker.setting = l.setting
	locker.retryCnt = 0
	locker.stopChan = make(chan struct{})
	locker.closed = make(chan struct{})
	return locker
}

func (l *lockerImpl) lockWithRetry(ctx context.Context, start time.Time, retry bool) (locked bool, err error) {
	var ticker *time.Ticker
	defer func() {
		if ticker != nil {
			ticker.Stop()
		}
	}()

	// exponential backoff retry
	for {
		if retry && atomic.LoadUint64(&l.retryCnt) > 0 && time.Since(start) > l.setting.MaxWaitTime {
			return false, failedToLock(fmt.Errorf("max wait time reached"))
		}
		locked, err := l.lockDB(ctx)
		if locked {
			return true, nil
		}
		if !retry {
			return false, err
		}
		// cal back off time
		cnt := atomic.AddUint64(&l.retryCnt, 1)
		backOffMs := 1 << cnt * time.Millisecond
		if backOffMs < l.setting.retryMinInterval {
			backOffMs = l.setting.retryMinInterval
		}
		if backOffMs > l.setting.retryMaxInterval {
			backOffMs = l.setting.retryMaxInterval
		}
		backOffMs = backOffMs/2 + time.Duration(rand.Int63n(int64(backOffMs/2)))
		if ticker == nil {
			ticker = time.NewTicker(backOffMs)
		} else {
			ticker.Reset(backOffMs)
		}

		<-ticker.C
	}
}

func (l *lockerImpl) lockDB(ctx context.Context) (locked bool, err error) {
	//start := time.Now()
	//defer func() {
	//	fmt.Printf("lockdb used: %v\n", time.Since(start))
	//}()
	locked, err = l.dbClient.SetNX(ctx, l.key, l.uuid, l.setting.Lease)
	//if (!locked || err != nil) && l.alreadyLockedDB(ctx) {
	//	// already locked by self
	//	return true, nil
	//}
	if locked {
		return true, nil
	}
	return false, err
}

func (l *lockerImpl) HasLock(ctx context.Context) bool {
	val, err := l.dbClient.Get(ctx, l.key)
	if err == nil && val == l.uuid {
		return true
	}
	return false
}

func (l *lockerImpl) startKeepAlive(ctx context.Context) {
	defer func() {
		close(l.closed)
	}()
	var renewTimer <-chan time.Time
	var releaseTicker <-chan time.Time
	if l.setting.keepAliveInterval > 0 {
		keepAliveTicker := time.NewTicker(l.setting.keepAliveInterval)
		defer keepAliveTicker.Stop()
		renewTimer = keepAliveTicker.C
	}
	if l.setting.MaxLockTime > 0 {
		maxLockTimer := time.NewTimer(l.setting.MaxLockTime)
		defer maxLockTimer.Stop()
		releaseTicker = maxLockTimer.C
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-l.stopChan:
			return
		case <-releaseTicker:
			go l.Release(ctx)
			return
		case <-renewTimer:
			if renewed, err := l.renew(ctx); err == nil && !renewed {
				fmt.Printf("error renewing: %v\n", err)
				return
			}
		}
	}
}

func (l *lockerImpl) renew(ctx context.Context) (renewed bool, err error) {
	for i := 0; i < 3; i++ {
		randomSleep(i == 0, 5, 5)
		renewed, err = l.dbClient.CompareAndSet(ctx, l.key, l.uuid, l.setting.Lease)
		if err == nil {
			break
		}
	}
	return renewed, err
}

func (l *lockerImpl) Release(ctx context.Context) (released bool, err error) {
	//start := time.Now()
	//defer func() {
	//	fmt.Printf("release used: %v\n", time.Since(start))
	//}()
	defer func() {
		// safe close
		if l.stopChan != nil {
			select {
			// already closed
			case <-l.stopChan:
			default:
				// stop keepalive
				close(l.stopChan)
			}
			// wait fully closed
			<-l.closed
		}
	}()
	for i := 0; i < 3; i++ {
		randomSleep(i == 0, 5, 5)
		released, err = l.dbClient.CompareAndDelete(ctx, l.key, l.uuid)
		if err != nil {
			continue
		}
		if !released {
			err = failedToRelease(fmt.Errorf("lock cannot be released"))
		}
		return released, err
	}
	return false, failedToRelease(err)
}
