package streamux

// TODO:
// - algorithm
// - thread safety
// - start value randomization
// - blocking?

type idPool struct {
	idMask int
	lastId int
}

func newIdPool(idBits int) *idPool {
	this := new(idPool)
	this.idMask = 1<<uint(idBits) - 1
	this.lastId = -1
	return this
}

func (this *idPool) AllocateId() int {
	// TODO
	this.lastId = (this.lastId + 1) & this.idMask
	return this.lastId
}

func (this *idPool) DeallocateId(id int) {
	// TODO
}
