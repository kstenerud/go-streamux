package streamux

// TODO: blocking?

import (
	"crypto/rand"
	mr "math/rand"
	"sync"
	"time"
)

const idPoolPrime = 1001353

type idPool struct {
	idMask   uint32
	lastId   uint32
	maxIds   int
	inFlight map[uint32]bool
	returned []uint32
	mutex    sync.Mutex
}

func randomId() uint32 {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err == nil {
		return uint32(bytes[0])<<24 |
			uint32(bytes[1])<<16 |
			uint32(bytes[2])<<8 |
			uint32(bytes[3])
	} else {
		mr.Seed(time.Now().UnixNano())
		return mr.Uint32()
	}
}

func newIdPool(idBits int) *idPool {
	this := new(idPool)
	this.maxIds = 1 << uint(idBits)
	this.idMask = 1<<uint(idBits) - 1
	this.inFlight = make(map[uint32]bool)
	this.lastId = randomId()
	return this
}

func (this *idPool) AllocateId() int {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	var newId uint32
	returnedLength := len(this.returned)
	if returnedLength > 0 {
		newId = this.returned[returnedLength-1]
		this.returned = this.returned[:returnedLength-1]
	} else {
		// TODO: This can deadlock since deallocate won't be able to unlock
		for newId = (this.lastId + idPoolPrime) & this.idMask; true; newId = (newId + idPoolPrime) & this.idMask {
			if _, exists := this.inFlight[newId]; exists {
				if len(this.inFlight) >= this.maxIds {
					// TODO: Full
				}
				continue
			} else {
				break
			}
		}
	}

	this.lastId = newId
	this.inFlight[newId] = true
	return int(this.lastId)
}

func (this *idPool) DeallocateId(id int) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	if _, exists := this.inFlight[uint32(id)]; exists {
		delete(this.inFlight, uint32(id))
		this.returned = append(this.returned, uint32(id))
	}
	// TODO: Error if it doesn't exist? Or no?
}
