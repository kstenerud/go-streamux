package buffer

type FeedableBuffer struct {
	Data         []byte
	minByteCount int
	maxByteCount int
}

// API

func New(minByteCount, maxByteCount, initialCapacity int) *FeedableBuffer {
	this := new(FeedableBuffer)
	this.Init(minByteCount, maxByteCount, initialCapacity)
	return this
}

func (this *FeedableBuffer) Init(minByteCount, maxByteCount, initialCapacity int) {
	this.minByteCount = minByteCount
	this.maxByteCount = maxByteCount
	this.Data = make([]byte, minByteCount, initialCapacity)
}

func (this *FeedableBuffer) IsFull() bool {
	return len(this.Data) == this.maxByteCount
}

func (this *FeedableBuffer) IsEmpty() bool {
	return len(this.Data) == this.minByteCount
}

// Shrink the buffer to its minimum size.
func (this *FeedableBuffer) Minimize() {
	this.Data = this.Data[:this.minByteCount]
}

// Expand the buffer to its maximum size. The expanded portion is not cleared.
func (this *FeedableBuffer) Maximize() {
	this.ExpandTo(this.maxByteCount)
}

// Expand the buffer to the specified size. The expanded portion is not cleared.
func (this *FeedableBuffer) ExpandTo(length int) {
	if cap(this.Data) < length {
		old := this.Data
		this.Data = make([]byte, length)
		copy(this.Data, old)
	} else {
		this.Data = this.Data[:length]
	}
}

func (this *FeedableBuffer) GetFreeByteCount() int {
	return this.maxByteCount - len(this.Data)
}

// Get the number of extra bytes filled in this buffer over the minimum
func (this *FeedableBuffer) GetUsedByteCountOverMinimum() int {
	return len(this.Data) - this.minByteCount
}

// Overwrite the head of this buffer with the specified bytes
func (this *FeedableBuffer) OverwriteHead(bytes []byte) {
	copy(this.Data, bytes)
}

// Feed as many bytes as will fit into this buffer (based on maxByteCount), returning whatever remains.
func (this *FeedableBuffer) Feed(bytesToFeed []byte) (bytesRemaining []byte) {
	byteCount := this.maxByteCount - len(this.Data)
	consumedPortion, bytesRemaining := ConsumeBytes(byteCount, bytesToFeed)
	this.Data = append(this.Data, consumedPortion...)
	return bytesRemaining
}

// Split a slice into a "consumed" portion buffer[:byteCount] and "remaining" portion buffer[byteCount:len(buffer)].
func ConsumeBytes(byteCount int, buffer []byte) (consumedPortion []byte, remainingPortion []byte) {
	if byteCount > len(buffer) {
		byteCount = len(buffer)
	}

	consumedPortion = buffer[:byteCount]
	remainingPortion = buffer[byteCount:len(buffer)]
	return consumedPortion, remainingPortion
}
