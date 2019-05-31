package streamux

type feedableBuffer struct {
	Data         []byte
	minByteCount int
	maxByteCount int
}

// API

func newBuffer(minByteCount, maxByteCount, initialCapacity int) *feedableBuffer {
	this := new(feedableBuffer)
	this.Init(minByteCount, maxByteCount, initialCapacity)
	return this
}

func (this *feedableBuffer) Init(minByteCount, maxByteCount, initialCapacity int) {
	this.minByteCount = minByteCount
	this.maxByteCount = maxByteCount
	this.Data = make([]byte, minByteCount, initialCapacity)
}

func (this *feedableBuffer) IsFull() bool {
	return len(this.Data) == this.maxByteCount
}

func (this *feedableBuffer) IsEmpty() bool {
	return len(this.Data) == this.minByteCount
}

func (this *feedableBuffer) Minimize() {
	this.Data = this.Data[:this.minByteCount]
}

func (this *feedableBuffer) Maximize() {
	this.Data = this.Data[:this.maxByteCount]
}

func (this *feedableBuffer) GetFreeByteCount() int {
	return this.maxByteCount - len(this.Data)
}

func (this *feedableBuffer) GetUsedByteCountOverMinimum() int {
	return len(this.Data) - this.minByteCount
}

func (this *feedableBuffer) InsertAtHead(bytes []byte) {
	copy(this.Data, bytes)
}

func (this *feedableBuffer) Feed(bytesToFeed []byte) (bytesRemaining []byte) {
	byteCount := this.maxByteCount - len(this.Data)
	consumedPortion, bytesRemaining := consumeBytes(byteCount, bytesToFeed)
	this.Data = append(this.Data, consumedPortion...)
	return bytesRemaining
}

func consumeBytes(byteCount int, buffer []byte) (consumedPortion []byte, remainingPortion []byte) {
	if byteCount > len(buffer) {
		byteCount = len(buffer)
	}

	consumedPortion = buffer[:byteCount]
	remainingPortion = buffer[byteCount:len(buffer)]
	return consumedPortion, remainingPortion
}
