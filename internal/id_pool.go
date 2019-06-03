package internal

import (
	"crypto/rand"
	"fmt"
	mr "math/rand"
	"time"
)

// TODO: implement a kind of circular buffer for freed ids?
// have minimum number of freed ids before recycling them to prevent thrashing?

type IdPool struct {
	maxIds        uint32
	idMask        uint32
	salt          uint32
	highestUsedId uint32
	freedIds      []uint32
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

func NewIdPool(idBits int) *IdPool {
	this := new(IdPool)
	this.Init(idBits)
	return this
}

func (this *IdPool) Init(idBits int) {
	if idBits < 0 || idBits > 30 {
		panic(fmt.Errorf("idBits (%v) out of allowed range 0-30", idBits))
	}
	this.salt = randomUint32()
	this.maxIds = 1 << uint(idBits)
	this.idMask = this.maxIds - 1
	this.highestUsedId = 0
	this.highestUsedId--
}

func (this *IdPool) AllocateId() (id int, ok bool) {
	var newId uint32

	if freedIdsCount := len(this.freedIds); freedIdsCount > 0 {
		newId = this.freedIds[freedIdsCount-1]
		this.freedIds = this.freedIds[:freedIdsCount-1]
	} else {
		newId = this.highestUsedId + 1
		if newId >= this.maxIds {
			return 0, false
		}
		this.highestUsedId = newId
	}

	return int((newId + this.salt) & this.idMask), true
}

// This method is not idempotent. Calling it with a not allocated ID will break things.
func (this *IdPool) DeallocateId(id int) {
	this.freedIds = append(this.freedIds, (uint32(id)-this.salt)&this.idMask)
}
