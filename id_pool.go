package streamux

// TODO:
// - algorithm
// - thread safety
// - start value randomization
// - blocking?

type IdPool struct {
	idMask int
	lastId int
}

func NewIdPool(idBits int) *IdPool {
	this := new(IdPool)
	this.idMask = 1<<uint(idBits) - 1
	this.lastId = -1
	return this
}

func (this *IdPool) AllocateId() int {
	// TODO
	this.lastId = (this.lastId + 1) & this.idMask
	return this.lastId
}

func (this *IdPool) DeallocateId(id int) {
	// TODO
}
