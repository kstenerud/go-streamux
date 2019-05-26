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

	isClosed bool
	protocol *Protocol
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

func (this *SendableMessage) sendCurrentChunk(terminationBit uint32) {
	this.fillHeader(terminationBit)
	this.protocol.sendMessageChunk(this.priority, this.data)
	this.data = this.data[0:this.headerLength]
}

func newSendableMessage(protocol *Protocol, priority int, id int, headerLength int, lengthBits int, idBits int, isResponse bool) *SendableMessage {
	this := new(SendableMessage)
	this.protocol = protocol
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

func (this *SendableMessage) AddData(bytesToSend []byte, isEndOfData bool) {
	if this.isClosed {
		panic(fmt.Errorf("Message has been closed"))
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
			this.sendCurrentChunk(dontTerminateMessage)
		}
	}

	if isEndOfData {
		terminateMessage := uint32(1)
		this.sendCurrentChunk(terminateMessage)
		this.Close()
	}
}

func (this *SendableMessage) Close() {
	if this.isClosed {
		return
	}
	this.protocol.deallocateId(this.Id)
	this.isClosed = true
}
