package streamux

import (
	"fmt"
)

const maxInitialBufferCapacity = 1024 + 4

type SendableMessage struct {
	Id int

	priority   int
	header     messageHeader_
	chunkData  []byte
	isEnded    bool
	chunksSent int

	messageSender MessageSender
}

func (this *SendableMessage) dataLength() int {
	return len(this.chunkData) - this.header.HeaderLength
}

func (this *SendableMessage) freeChunkSpace() int {
	return this.header.MaxChunkLength - this.dataLength()
}

func (this *SendableMessage) sendCurrentChunk() error {
	this.header.SetLengthAndTermination(this.dataLength(), this.isEnded)
	copy(this.chunkData, this.header.Encoded)
	err := this.messageSender.OnMessageChunkToSend(this.priority, this.Id, this.chunkData)
	this.chunkData = this.chunkData[0:this.header.HeaderLength]
	this.chunksSent++
	return err
}

func newSendableMessage(messageSender MessageSender, priority int, id int,
	lengthBits int, idBits int, isResponse bool) *SendableMessage {

	this := new(SendableMessage)
	this.Init(messageSender, priority, id, lengthBits, idBits, isResponse)
	return this
}

func (this *SendableMessage) Init(messageSender MessageSender,
	priority int, id int, lengthBits int, idBits int, isResponse bool) {

	this.Id = id
	this.messageSender = messageSender
	this.priority = priority
	this.header.Init(lengthBits, idBits)
	this.header.SetIdAndResponseNoEncode(id, isResponse)

	initialBufferCapacity := this.header.MaxChunkLength
	if initialBufferCapacity > maxInitialBufferCapacity {
		initialBufferCapacity = maxInitialBufferCapacity
	}
	this.chunkData = make([]byte, this.header.HeaderLength, initialBufferCapacity)
}

func (this *SendableMessage) Add(bytesToSend []byte) error {
	if this.isEnded {
		return fmt.Errorf("Message has been ended")
	}

	for len(bytesToSend) > this.freeChunkSpace() {
		this.chunkData, bytesToSend = fillBuffer(this.header.HeaderLength+this.header.MaxChunkLength, this.chunkData, bytesToSend)
		if err := this.sendCurrentChunk(); err != nil {
			// fmt.Printf("### ERROR %v\n", err)
			return err
		}
	}

	this.chunkData = append(this.chunkData, bytesToSend...)

	return nil
}

func (this *SendableMessage) Flush() error {
	if this.dataLength() > 0 {
		return this.sendCurrentChunk()
	}
	return nil

}

func (this *SendableMessage) End() error {
	if this.isEnded {
		return nil
	}

	// Force and update to the message type
	this.isEnded = true
	this.header.SetLengthAndTermination(this.dataLength(), this.isEnded)

	// if this.chunksSent == 0 && this.dataLength() == 0 {
	if this.chunksSent == 0 {
		switch this.header.MessageType {
		case messageTypeCancel, messageTypeCancelAck, messageTypePing:
			return fmt.Errorf("Use OOB message methods to send OOB messages")
		case messageTypeEmptyResponse:
			// This is allowed
		case messageTypeNormal:
			// This is allowed
		}
	}

	return this.sendCurrentChunk()

}
