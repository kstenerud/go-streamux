package streamux

// TODO:
// - OOB messages
// - ???

import (
	"fmt"
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
	OnAbleToSend()
	OnMessageChunkToSend(priority int, data []byte) error
}

type Protocol struct {
	hasStarted       bool
	negotiator       *negotiator_
	decoder          *messageDecoder_
	idPool           *idPool
	callbacks        MessageSendCallbacks
	decoderCallbacks MessageReceiveCallbacks
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

	this.negotiator = newNegotiator(lengthMinBits, lengthMaxBits, lengthRecommendBits,
		idMinBits, idMaxBits, idRecommendBits,
		requestQuickInit, allowQuickInit)
	this.callbacks = sendCallbacks
	this.decoderCallbacks = receiveCallbacks
}

func (this *Protocol) feedNegotiator(incomingStreamData []byte) ([]byte, error) {
	if !this.negotiator.IsNegotiationComplete() {
		var err error
		if incomingStreamData, err = this.negotiator.Feed(incomingStreamData); err != nil {
			return nil, err
		}
		if !this.negotiator.IsNegotiationComplete() {
			if len(incomingStreamData) != 0 {
				return nil, fmt.Errorf("INTERNAL BUG: %v bytes in incoming stream, but negotiation still not complete", len(incomingStreamData))
			}
			return incomingStreamData, nil
		}

		this.decoder = newMessageDecoder(this.negotiator.HeaderLength, this.negotiator.LengthBits, this.negotiator.IdBits, this.decoderCallbacks)
		this.idPool = newIdPool(this.negotiator.IdBits)
		this.callbacks.OnAbleToSend()
	}
	return incomingStreamData, nil
}

func (this *Protocol) feedDecoder(incomingStreamData []byte) error {
	return this.decoder.Feed(incomingStreamData)
}

func (this *Protocol) sendMessageChunk(priority int, chunk []byte) error {
	return this.callbacks.OnMessageChunkToSend(priority, chunk)
}

func (this *Protocol) Start() error {
	if !this.hasStarted {
		this.hasStarted = true
		return this.sendMessageChunk(PriorityOOB, this.negotiator.BuildInitializeMessage())
	}
	if this.negotiator.CanSendMessages() {
		this.callbacks.OnAbleToSend()
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
	if !this.negotiator.CanSendMessages() {
		return nil, fmt.Errorf("Can't send messages: %v", this.negotiator.ExplainFailure())
	}

	isResponse := false
	return newSendableMessage(this.idPool, this.callbacks, priority, this.allocateId(),
		this.negotiator.HeaderLength, this.negotiator.LengthBits,
		this.negotiator.IdBits, isResponse), nil
}

func (this *Protocol) BeginResponseMessage(priority int, responseToId int) (*SendableMessage, error) {
	if !this.negotiator.CanSendMessages() {
		return nil, fmt.Errorf("Can't send messages: %v", this.negotiator.ExplainFailure())
	}

	isResponse := true
	return newSendableMessage(this.idPool, this.callbacks, priority, this.allocateId(),
		this.negotiator.HeaderLength, this.negotiator.LengthBits,
		this.negotiator.IdBits, isResponse), nil
}

func (this *Protocol) Feed(incomingStreamData []byte) error {
	if err := this.Start(); err != nil {
		return err
	}

	var err error
	if incomingStreamData, err = this.feedNegotiator(incomingStreamData); err != nil {
		return err
	}

	if this.negotiator.CanReceiveMessages() {
		return this.feedDecoder(incomingStreamData)
	} else if len(incomingStreamData) > 0 {
		return fmt.Errorf("Can't receive messages: %v", this.negotiator.ExplainFailure())
	}

	return nil
}
