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
	idPool    *IdPool
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

func (this *SendableMessage) fillHeader(terminationBit uint32) {
	headerFields := terminationBit |
		this.responseBitShifted |
		this.idShifted |
		uint32(this.dataLength())<<this.shiftLength

	for i := 0; i < this.headerLength; i++ {
		this.data[i] = byte(headerFields)
		headerFields >>= 8
	}
}

func (this *SendableMessage) sendCurrentChunk(terminationBit uint32) error {
	this.fillHeader(terminationBit)
	if err := this.callbacks.OnMessageChunkToSend(this.priority, this.data); err != nil {
		return err
	}
	this.data = this.data[0:this.headerLength]
	return nil
}

func newSendableMessage(idPool *IdPool, callbacks MessageSendCallbacks,
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
	this.data = make([]byte, this.headerLength, this.maxChunkLength)

	return this
}

func (this *SendableMessage) AddData(bytesToSend []byte, isEndOfData bool) error {
	if this.isClosed {
		return fmt.Errorf("Message has been closed")
	}

	for len(bytesToSend) > 0 {
		filledByteCount := this.dataLength()
		toSendByteCount := len(bytesToSend)
		if filledByteCount+toSendByteCount > this.maxChunkLength {
			toSendByteCount = this.maxChunkLength - filledByteCount
		}
		bytesToAppend := bytesToSend[:toSendByteCount]
		bytesToSend = bytesToSend[toSendByteCount:]
		this.data = append(this.data, bytesToAppend...)
		if this.dataLength() == this.maxChunkLength {
			dontTerminateMessage := uint32(0)
			if err := this.sendCurrentChunk(dontTerminateMessage); err != nil {
				return err
			}
		}
	}

	if isEndOfData {
		terminateMessage := uint32(1)
		if err := this.sendCurrentChunk(terminateMessage); err != nil {
			return err
		}
		this.Close()
	}

	return nil
}

func (this *SendableMessage) Close() {
	if this.isClosed {
		return
	}
	this.idPool.DeallocateId(this.Id)
	this.isClosed = true
}
