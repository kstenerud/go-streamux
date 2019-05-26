package streamux

// TODO:
// - OOB messages
// - Quick init
// - ???

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
	hasStarted bool
	negotiator negotiator_
	decoder    messageDecoder_
	idPool     IdPool
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
	this.Init(lengthMinBits, lengthMaxBits, lengthRecommendBits,
		idMinBits, idMaxBits, idRecommendBits,
		requestQuickInit, allowQuickInit,
		sendCallbacks, receiveCallbacks)

	return this
}

func (this *Protocol) Init(lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	idMinBits int, idMaxBits int, idRecommendBits int, requestQuickInit bool,
	allowQuickInit bool, sendCallbacks MessageSendCallbacks,
	receiveCallbacks MessageReceiveCallbacks) {

	this.negotiator.Init(lengthMinBits, lengthMaxBits, lengthRecommendBits,
		idMinBits, idMaxBits, idRecommendBits,
		requestQuickInit, allowQuickInit)
	this.callbacks = sendCallbacks
	this.decoder.callbacks = receiveCallbacks
}

func (this *Protocol) Start() {
	if !this.hasStarted {
		this.hasStarted = true
		this.sendMessageChunk(priorityOOB, this.negotiator.BuildInitializeMessage())
	}
}

func (this *Protocol) SendMessage(priority int, contents []byte) {
	isEndOfData := true
	message := this.BeginMessage(priority)
	defer message.Close()
	message.AddData(contents, isEndOfData)
}

func (this *Protocol) SendResponseMessage(priority int, responseToId int, contents []byte) {
	isEndOfData := true
	message := this.BeginResponseMessage(priority, responseToId)
	defer message.Close()
	message.AddData(contents, isEndOfData)
}

func (this *Protocol) BeginMessage(priority int) *SendableMessage {
	this.Start()
	isResponse := false
	return newSendableMessage(this, priority, this.allocateId(),
		this.negotiator.HeaderLength, this.negotiator.LengthBits,
		this.negotiator.IdBits, isResponse)
}

func (this *Protocol) BeginResponseMessage(priority int, responseToId int) *SendableMessage {
	this.Start()
	isResponse := true
	return newSendableMessage(this, priority, this.allocateId(),
		this.negotiator.HeaderLength, this.negotiator.LengthBits,
		this.negotiator.IdBits, isResponse)
}

func (this *Protocol) Feed(incomingStreamData []byte) error {
	this.Start()
	if !this.negotiator.IsNegotiated {
		var err error
		if incomingStreamData, err = this.negotiator.Feed(incomingStreamData); err != nil {
			return err
		}
		if this.negotiator.IsNegotiated {
			this.decoder.Initialize(this.negotiator.HeaderLength, this.negotiator.LengthBits, this.negotiator.IdBits)
		}
	}

	return this.decoder.Feed(incomingStreamData)
}
