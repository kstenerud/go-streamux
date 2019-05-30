package streamux

import (
	"crypto/rand"
	"fmt"
	mr "math/rand"
	"sync"
	"time"
)

type idPool struct {
	maxIds        uint32
	idMask        uint32
	salt          uint32
	highestUsedId uint32
	freedIds      []uint32
	mutex         sync.Mutex
}

func randomUint32() uint32 {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err == nil {
		return uint32(bytes[0])<<24 |
			uint32(bytes[1])<<16 |
			uint32(bytes[2])<<8 |
			uint32(bytes[3])
	} else {
		var random mr.Rand
		random.Seed(time.Now().UnixNano())
		return random.Uint32()
	}
}

func newIdPool(idBits int) *idPool {
	this := new(idPool)
	this.Init(idBits)
	return this
}

func (this *idPool) Init(idBits int) {
	if idBits < 0 || idBits > 30 {
		panic(fmt.Errorf("idBits (%v) out of allowed range 0-30", idBits))
	}
	this.salt = randomUint32()
	this.maxIds = 1 << uint(idBits)
	this.idMask = this.maxIds - 1
	this.highestUsedId = 0
	this.highestUsedId--
}

func (this *idPool) AllocateId() int {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	var newId uint32

	if freedIdsCount := len(this.freedIds); freedIdsCount > 0 {
		newId = this.freedIds[freedIdsCount-1]
		this.freedIds = this.freedIds[:freedIdsCount-1]
	} else {
		newId = this.highestUsedId + 1
		if newId >= this.maxIds {
			panic(fmt.Errorf("ID pool exhausted"))
		}
		this.highestUsedId = newId
	}

	return int((newId + this.salt) & this.idMask)
}

// This method is not idempotent. Calling it with a not allocated ID will break things.
func (this *idPool) DeallocateId(id int) {
	this.mutex.Lock()
	defer this.mutex.Unlock()

	this.freedIds = append(this.freedIds, (uint32(id)-this.salt)&this.idMask)
}
