package streamux

import (
	"fmt"
)

type SendableMessage struct {
	Id int

	// Per-message constants
	maxChunkLength     int
	headerLength       int
	responseBitShifted uint32
	idShifted          uint32
	shiftLength        uint

	priority int

	data []byte

	isClosed  bool
	idPool    *idPool
	callbacks MessageSendCallbacks
}

func boolToUint32(value bool) uint32 {
	if value {
		return 1
	}
	return 0
}

func (this *SendableMessage) dataLength() int {
	return len(this.data) - this.headerLength
}

func (this *SendableMessage) freeChunkSpace() int {
	return this.maxChunkLength - this.dataLength()
}

func (this *SendableMessage) clearData() {
	this.data = this.data[0:this.headerLength]
}

func (this *SendableMessage) fillHeader(terminationBit uint32) {
	headerFields := terminationBit |
		this.responseBitShifted |
		this.idShifted |
		uint32(this.dataLength())<<this.shiftLength

	for i := 0; i < this.headerLength; i++ {
		this.data[i] = byte(headerFields)
		headerFields >>= 8
	}
	// fmt.Printf("### SM %p: lshift %v, ishift %v\n", this, this.shiftLength, shiftId)
	// fmt.Printf("### SM %p: Decode header len %v, id %v, resp %v, term %v\n", this, this.dataLength(), this.idShifted>>shiftId, this.responseBitShifted>>shiftResponseBit, terminationBit)
}

func (this *SendableMessage) sendCurrentChunk(terminationBit uint32) error {
	this.fillHeader(terminationBit)
	if err := this.callbacks.OnMessageChunkToSend(this.priority, this.data); err != nil {
		return err
	}
	this.clearData()
	return nil
}

func newSendableMessage(idPool *idPool, callbacks MessageSendCallbacks,
	priority int, id int, headerLength int, lengthBits int, idBits int,
	isResponse bool) *SendableMessage {

	this := new(SendableMessage)
	this.idPool = idPool
	this.callbacks = callbacks
	this.priority = priority
	this.Id = id
	this.idShifted = uint32(id) << shiftId
	this.responseBitShifted = boolToUint32(isResponse) << shiftResponseBit
	this.shiftLength = uint(shiftId + idBits)
	this.headerLength = headerLength
	this.maxChunkLength = 1<<uint(lengthBits) - 1

	initialBufferCapacity := this.maxChunkLength
	if initialBufferCapacity > 1028 {
		initialBufferCapacity = 1028
	}
	this.data = make([]byte, this.headerLength, initialBufferCapacity)

	return this
}

func (this *SendableMessage) AddData(bytesToSend []byte, isEndOfData bool) error {
	if this.isClosed {
		return fmt.Errorf("Message has been closed")
	}

	// TODO: Don't allow end of data if no bytes have been added since the beginning
	// except for empty response...

	for len(bytesToSend) > this.freeChunkSpace() {
		// fmt.Printf("### calc: max chunk length %v, data %v, header length %v\n", this.maxChunkLength, len(this.data), this.headerLength)
		appendByteCount := this.freeChunkSpace()
		bytesToAppend := bytesToSend[:appendByteCount]
		// fmt.Printf("### add data: free space %v, bytes to append %v, bytes to send %v, data %v\n",
		// 	this.freeChunkSpace(), len(bytesToAppend), len(bytesToSend), len(this.data))

		bytesToSend = bytesToSend[appendByteCount:]
		this.data = append(this.data, bytesToAppend...)
		// fmt.Printf("### bytes to send now %v, data now %v\n", len(bytesToSend), len(this.data))
		doNotTerminateMessage := uint32(0)
		if err := this.sendCurrentChunk(doNotTerminateMessage); err != nil {
			// fmt.Printf("### ERROR %v\n", err)
			return err
		}
	}

	this.data = append(this.data, bytesToSend...)
	termination := uint32(0)
	if isEndOfData {
		termination = uint32(1)
	}
	if err := this.sendCurrentChunk(termination); err != nil {
		return err
	}

	if isEndOfData {
		this.Close()
	}

	return nil
}

func (this *SendableMessage) Close() {
	if this.isClosed {
		return
	}
	if this.idPool != nil {
		this.idPool.DeallocateId(this.Id)
	}
	this.isClosed = true
}
