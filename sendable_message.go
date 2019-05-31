package streamux

import (
	"fmt"
)

const maxInitialBufferCapacity = 1024 + 4

type SendableMessage struct {
	Id int

	priority   int
	header     messageHeader
	chunkData  feedableBuffer
	isEnded    bool
	chunksSent int

	messageSender internalMessageSender
}

// API

func newSendableMessage(messageSender internalMessageSender, priority int, id int,
	lengthBits int, idBits int, isResponse bool) *SendableMessage {

	this := new(SendableMessage)
	this.Init(messageSender, priority, id, lengthBits, idBits, isResponse)
	return this
}

func (this *SendableMessage) Init(messageSender internalMessageSender,
	priority int, id int, lengthBits int, idBits int, isResponse bool) {

	this.Id = id
	this.messageSender = messageSender
	this.priority = priority
	this.header.Init(lengthBits, idBits)
	this.header.SetIdAndResponseNoEncode(id, isResponse)

	initialBufferCapacity := this.header.HeaderLength + this.header.MaxChunkLength
	if initialBufferCapacity > maxInitialBufferCapacity {
		initialBufferCapacity = maxInitialBufferCapacity
	}
	this.chunkData.Init(this.header.HeaderLength,
		this.header.HeaderLength+this.header.MaxChunkLength, initialBufferCapacity)
}

func (this *SendableMessage) Add(bytesToSend []byte) error {
	if this.isEnded {
		return fmt.Errorf("Cannot add more data: message has ended")
	}

	for len(bytesToSend) > this.chunkData.GetFreeByteCount() {
		bytesToSend = this.chunkData.Feed(bytesToSend)
		if err := this.sendCurrentChunk(); err != nil {
			return err
		}
	}

	this.chunkData.Feed(bytesToSend)

	return nil
}

func (this *SendableMessage) Flush() error {
	if this.getDataLength() > 0 {
		return this.sendCurrentChunk()
	}
	return nil
}

func (this *SendableMessage) End() error {
	if this.isEnded {
		return nil
	}

	this.isEnded = true
	this.header.SetLengthAndTermination(this.getDataLength(), this.isEnded)

	switch this.header.MessageType {
	case messageTypeRequestEmptyTermination:
		if this.chunksSent == 0 {
			return fmt.Errorf("A request message must contain at least 1 byte of payload")
		}
	case messageTypeCancel, messageTypeCancelAck:
		return fmt.Errorf("Internal bug: Message type %v should not be possible", this.header.MessageType)
	case messageTypeRequest, messageTypeResponse, messageTypeEmptyResponse:
		// These are allowed
	default:
		return fmt.Errorf("Internal bug: Unhandled message type: %v", this.header.MessageType)
	}

	return this.sendCurrentChunk()
}

// Internal

func (this *SendableMessage) getDataLength() int {
	return this.chunkData.GetUsedByteCountOverMinimum()
}

func (this *SendableMessage) sendCurrentChunk() error {
	// fmt.Printf("### SM %p: Send chunk length %v, end %v\n", this, this.getDataLength(), this.isEnded)
	this.header.SetLengthAndTermination(this.getDataLength(), this.isEnded)
	this.chunkData.InsertAtHead(this.header.encoded.Data)
	err := this.messageSender.OnMessageChunkToSend(this.priority, this.Id, this.chunkData.Data)
	this.chunkData.Minimize()
	this.chunksSent++
	return err
}
