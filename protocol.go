package streamux

// TODO:
// - OOB messages
// - ???

import (
	"math"
)

const ProtocolVersion = 1

const PriorityMax = math.MaxInt32
const PriorityOOB = PriorityMax

type MessageReceiveCallbacks interface {
	OnRequestChunkReceived(messageId int, isEnd bool, data []byte) error
	OnResponseChunkReceived(messageId int, isEnd bool, data []byte) error
}

type MessageSendCallbacks interface {
	OnMessageChunkToSend(priority int, data []byte) error
}

type Protocol struct {
	hasStarted bool
	negotiator negotiator_
	decoder    messageDecoder_
	idPool     IdPool
	callbacks  MessageSendCallbacks
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

func (this *Protocol) sendMessageChunk(priority int, chunk []byte) error {
	return this.callbacks.OnMessageChunkToSend(priority, chunk)
}

func (this *Protocol) Start() error {
	if !this.hasStarted {
		this.hasStarted = true
		return this.sendMessageChunk(PriorityOOB, this.negotiator.BuildInitializeMessage())
	}
	return nil
}

func (this *Protocol) SendMessage(priority int, contents []byte) error {
	isEndOfData := true
	message, err := this.BeginMessage(priority)
	if err != nil {
		return err
	}
	defer message.Close()
	return message.AddData(contents, isEndOfData)
}

func (this *Protocol) SendResponseMessage(priority int, responseToId int, contents []byte) error {
	isEndOfData := true
	message, err := this.BeginResponseMessage(priority, responseToId)
	if err != nil {
		return err
	}
	defer message.Close()
	return message.AddData(contents, isEndOfData)
}

func (this *Protocol) BeginMessage(priority int) (*SendableMessage, error) {
	if err := this.Start(); err != nil {
		return nil, err
	}

	isResponse := false
	return newSendableMessage(this, priority, this.allocateId(),
		this.negotiator.HeaderLength, this.negotiator.LengthBits,
		this.negotiator.IdBits, isResponse), nil
}

func (this *Protocol) BeginResponseMessage(priority int, responseToId int) (*SendableMessage, error) {
	if err := this.Start(); err != nil {
		return nil, err
	}

	isResponse := true
	return newSendableMessage(this, priority, this.allocateId(),
		this.negotiator.HeaderLength, this.negotiator.LengthBits,
		this.negotiator.IdBits, isResponse), nil
}

func (this *Protocol) Feed(incomingStreamData []byte) error {
	if err := this.Start(); err != nil {
		return err
	}

	if !this.negotiator.IsNegotiated {
		var err error
		if incomingStreamData, err = this.negotiator.Feed(incomingStreamData); err != nil {
			return err
		}
		if this.negotiator.IsNegotiated {
			this.decoder.Init(this.negotiator.HeaderLength, this.negotiator.LengthBits, this.negotiator.IdBits)
		}
	}

	return this.decoder.Feed(incomingStreamData)
}
