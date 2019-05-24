package streamux

import (
	"math"
)

const protocolVersion = 1

const priorityOOB = math.MaxInt32

type MessageReceiveCallbacks interface {
	OnRequestChunkReceived(messageId int, isEnd bool, data []byte)
	OnResponseChunkReceived(messageId int, isEnd bool, data []byte)
}

type MessageSendCallbacks interface {
	OnMessageChunkToSend(priority int, data []byte)
}

type Protocol struct {
	negotiator negotiator_
	decoder    messageDecoder_
	idPool     *IdPool
	callbacks  MessageSendCallbacks
}

func (this *Protocol) sendMessageChunk(priority int, message []byte) {
	this.callbacks.OnMessageChunkToSend(priority, message)
}

func (this *Protocol) allocateId() int {
	return this.idPool.AllocateId()
}

func (this *Protocol) deallocateId(id int) {
	this.idPool.DeallocateId(id)
}

func NewProtocol(lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	idMinBits int, idMaxBits int, idRecommendBits int, requestQuickInit bool,
	allowQuickInit bool, sendCallbacks MessageSendCallbacks,
	receiveCallbacks MessageReceiveCallbacks) *Protocol {

	this := new(Protocol)
	this.negotiator.Initialize(lengthMinBits, lengthMaxBits, lengthRecommendBits,
		idMinBits, idMaxBits, idRecommendBits, requestQuickInit, allowQuickInit)
	this.callbacks = sendCallbacks
	this.decoder.callbacks = receiveCallbacks

	this.sendMessageChunk(priorityOOB, this.negotiator.BuildInitializeMessage())

	return this
}

func (this *Protocol) BeginMessage(priority int) *SendableMessage {
	isResponse := false
	return newSendableMessage(this, priority, this.allocateId(),
		this.negotiator.HeaderLength, this.negotiator.IdBits,
		this.negotiator.LengthBits, isResponse)
}

func (this *Protocol) BeginResponseMessage(priority int, responseToId int) *SendableMessage {
	isResponse := true
	return newSendableMessage(this, priority, this.allocateId(),
		this.negotiator.HeaderLength, this.negotiator.IdBits,
		this.negotiator.LengthBits, isResponse)
}

func (this *Protocol) Feed(incomingStreamData []byte) error {
	if !this.negotiator.IsNegotiated {
		var err error
		if incomingStreamData, err = this.negotiator.Negotiate(incomingStreamData); err != nil {
			return err
		}
		if this.negotiator.IsNegotiated {
			this.decoder.Initialize(this.negotiator.HeaderLength, this.negotiator.IdBits, this.negotiator.LengthBits)
		}
	}

	return this.decoder.Feed(incomingStreamData)
}
