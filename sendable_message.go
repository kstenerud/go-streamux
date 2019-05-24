package streamux

import (
	"fmt"
)

func boolToUint32(value bool) uint32 {
	if value {
		return 1
	}
	return 0
}

type SendableMessage struct {
	protocol           *Protocol
	id                 int
	responseBitShifted uint32
	idShifted          uint32
	lengthShift        uint
	headerLength       int
	maxChunkLength     int
	data               []byte
	isClosed           bool
}

func (this *SendableMessage) dataLength() int {
	return len(this.data) - this.headerLength
}

func (this *SendableMessage) fillHeader(terminationBit uint32) {
	headerFields := terminationBit |
		this.responseBitShifted |
		this.idShifted |
		uint32(this.dataLength())<<this.lengthShift

	for i := 0; i < this.headerLength; i++ {
		this.data[i] = byte(headerFields)
		headerFields >>= 8
	}
}

func (this *SendableMessage) sendCurrentChunk(terminationBit uint32) {
	this.fillHeader(terminationBit)
	this.protocol.sendMessageChunk(this.data)
}

func newSendableMessage(protocol *Protocol, id int, idBits int, lengthBits int, isResponse bool) *SendableMessage {
	this := new(SendableMessage)
	this.protocol = protocol
	this.id = id
	this.idShifted = uint32(id) << shiftId
	this.headerLength = protocol.getHeaderLength()
	this.maxChunkLength = 1<<uint(lengthBits) - 1
	this.data = make([]byte, this.headerLength, this.maxChunkLength)
	this.responseBitShifted = boolToUint32(isResponse) << shiftResponseBit
	this.lengthShift = uint(shiftId + idBits)

	return this
}

func (this *SendableMessage) AddData(data []byte) {
	if this.isClosed {
		panic(fmt.Errorf("Message has been closed"))
	}

	for len(data) > 0 {
		currentLength := this.dataLength()
		lengthToAdd := len(data)
		if currentLength+lengthToAdd > this.maxChunkLength {
			lengthToAdd = this.maxChunkLength - currentLength
		}
		this.data = append(this.data, data[:lengthToAdd]...)
		currentLength = this.dataLength()
		if currentLength == this.maxChunkLength {
			this.sendCurrentChunk(0)
			data = data[lengthToAdd:]
		}
	}
}

func (this *SendableMessage) Complete() {
	if this.isClosed {
		panic(fmt.Errorf("Message has been closed"))
	}

	this.sendCurrentChunk(1)
	this.Close()
}

func (this *SendableMessage) Close() {
	if this.isClosed {
		return
	}
	this.protocol.disposeId(this.id)
	this.isClosed = true
}
