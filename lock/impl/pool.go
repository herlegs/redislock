package impl

import (
	"sync"
)

var lockPool *sync.Pool

func init() {
	lockPool = &sync.Pool{
		New: func() interface{} {
			return &lockerImpl{}
		},
	}
}

func poolGetLocker() *lockerImpl {
	return lockPool.Get().(*lockerImpl)
}

func reset(locker *lockerImpl) {
	locker.reset()
}

func poolReturnLocker(locker *lockerImpl) {
	reset(locker)
	lockPool.Put(locker)
}
