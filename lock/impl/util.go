package impl

import (
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"math/rand"
	"os"
	"sync/atomic"
	"time"
)

var (
	machineID = getMachineID()
	processID = os.Getpid()
	counter   uint32
)

func getMachineID() []byte {
	var sum [3]byte
	id := sum[:]
	hostname, err1 := os.Hostname()
	if err1 != nil {
		n := uint32(time.Now().UnixNano())
		sum[0] = byte(n >> 0)
		sum[1] = byte(n >> 8)
		sum[2] = byte(n >> 16)
		return id
	}
	hw := md5.New()
	hw.Write([]byte(hostname))
	copy(id, hw.Sum(nil))
	return id
}

// based on timestamp, machine id, process id, and in mem counter
func newRandID() string {
	var b [12]byte
	binary.BigEndian.PutUint32(b[:], uint32(time.Now().Unix()))
	b[4] = machineID[0]
	b[5] = machineID[1]
	b[6] = machineID[2]
	b[7] = byte(processID >> 8)
	b[8] = byte(processID)
	i := atomic.AddUint32(&counter, 1)
	b[9] = byte(i >> 16)
	b[10] = byte(i >> 8)
	b[11] = byte(i)
	return hex.EncodeToString(b[:])
}

func randomSleep(firstTime bool, fixMs, randMs int) {
	if firstTime {
		return
	}
	time.Sleep(time.Millisecond * time.Duration(fixMs+rand.Intn(randMs+1)))
}
