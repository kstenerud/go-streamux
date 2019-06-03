package streamux

import (
	"fmt"
	"math"

	"github.com/kstenerud/go-streamux/internal"
)

const ProtocolVersion = 1

const PriorityMax = math.MaxInt32
const PriorityOOB = PriorityMax

// Protocol encapsulates the top level API of the streamux protocol.
type Protocol struct {
	hasBegunInitialization         bool
	hasFinishedEarlyInitialization bool
	negotiator                     internal.ProtocolNegotiator
	decoder                        internal.MessageDecoder
	requestRules                   internal.RequestFlightRules
	sender                         MessageSender
	receiver                       MessageReceiver
}

// API

func NewProtocol(lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	idMinBits int, idMaxBits int, idRecommendBits int, requestQuickInit bool,
	allowQuickInit bool, sender MessageSender,
	receiver MessageReceiver) *Protocol {

	this := new(Protocol)
	this.Init(lengthMinBits, lengthMaxBits, lengthRecommendBits,
		idMinBits, idMaxBits, idRecommendBits,
		requestQuickInit, allowQuickInit,
		sender, receiver)

	return this
}

func (this *Protocol) Init(lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	idMinBits int, idMaxBits int, idRecommendBits int, requestQuickInit bool,
	allowQuickInit bool, sender MessageSender,
	receiver MessageReceiver) {

	this.negotiator.Init(ProtocolVersion, lengthMinBits, lengthMaxBits, lengthRecommendBits,
		idMinBits, idMaxBits, idRecommendBits,
		requestQuickInit, allowQuickInit)
	this.sender = sender
	this.receiver = receiver
}

func (this *Protocol) SendInitialization() error {
	if !this.hasBegunInitialization {
		this.hasBegunInitialization = true
		if this.negotiator.CanSendMessages() {
			this.finishEarlyInitialization()
		}
		// fmt.Printf("### P %p: Sending init message\n", this)
		return this.sendRawMessage(PriorityOOB, this.negotiator.BuildInitializeMessage())
	}
	return nil
}

func (this *Protocol) SendRequest(priority int, contents []byte) (messageId int, err error) {
	message, err := this.BeginRequest(priority)
	if err != nil {
		return 0, err
	}
	if err = message.Feed(contents); err != nil {
		return message.Id, err
	}
	return message.Id, message.End()
}

func (this *Protocol) SendResponse(priority int, responseToId int, contents []byte) error {
	message, err := this.BeginResponse(priority, responseToId)
	if err != nil {
		return err
	}
	if err = message.Feed(contents); err != nil {
		return err
	}
	return message.End()
}

// Advanced API. The SendableMessage returned by this method can be used to incrementally
// add data to the message being sent. Data will be queued and sent as it fills the maximum chunk length.
func (this *Protocol) BeginRequest(priority int) (message *SendableMessage, err error) {
	if !this.negotiator.CanSendMessages() {
		return nil, fmt.Errorf("Can't send messages: %v", this.negotiator.ExplainFailure())
	}

	err = this.requestRules.TryBeginRequest(func(id int) {
		isResponse := false
		message = newSendableMessage(this, priority, id,
			this.negotiator.LengthBits, this.negotiator.IdBits, isResponse)
	})
	return message, err
}

// Advanced API. The SendableMessage returned by this method can be used to incrementally
// add data to the message being sent. Data will be queued and sent as it fills the maximum chunk length.
func (this *Protocol) BeginResponse(priority int, responseToId int) (*SendableMessage, error) {
	if !this.negotiator.CanSendMessages() {
		return nil, fmt.Errorf("Can't send messages: %v", this.negotiator.ExplainFailure())
	}

	isResponse := true
	return newSendableMessage(this, priority, responseToId,
		this.negotiator.LengthBits, this.negotiator.IdBits, isResponse), nil
}

func (this *Protocol) Cancel(messageId int) (err error) {
	var innerErr error
	err = this.requestRules.TryCancelRequest(messageId, func(id int) {
		innerErr = this.OnMessageChunkToSend(PriorityOOB, id, this.newEmptyMessageHeader(id, internal.MessageTypeCancel))
	})
	if err == nil {
		err = innerErr
	}
	return err
}

func (this *Protocol) Ping() (id int, err error) {
	this.requestRules.TryPing(func(newId int) {
		id = newId
		err = this.OnMessageChunkToSend(PriorityOOB, id, this.newEmptyMessageHeader(id, internal.MessageTypeRequestEmptyTermination))
	})

	return id, err
}

func (this *Protocol) Feed(incomingStreamData []byte) (err error) {
	// fmt.Printf("### P %p: Feed %v bytes. Negotiation complete: %v\n", this, len(incomingStreamData), this.negotiator.IsNegotiationComplete())
	remainingData := incomingStreamData

	if !this.negotiator.IsNegotiationComplete() {
		if remainingData, err = this.feedNegotiator(remainingData); err != nil {
			return err
		}
	}

	if len(remainingData) > 0 && !this.negotiator.CanReceiveMessages() {
		return fmt.Errorf("Can't receive messages: %v", this.negotiator.ExplainFailure())
	}

	for len(remainingData) > 0 {
		var err error
		if remainingData, err = this.decoder.Feed(remainingData); err != nil {
			return err
		}
	}

	return nil
}

// Callbacks

func (this *Protocol) OnMessageChunkToSend(priority int, messageId int, data []byte) error {
	return this.sendRawMessage(priority, data)
}

func (this *Protocol) OnRequestChunkReceived(messageId int, isEnd bool, data []byte) error {
	// TODO: keep track of chunks-in-progress so that we can detect ping and ping response
	return this.receiver.OnRequestChunkReceived(messageId, isEnd, data)
}

func (this *Protocol) OnResponseChunkReceived(messageId int, isEnd bool, data []byte) (err error) {
	this.requestRules.TryReceiveResponseChunk(messageId, isEnd, func(id int, isTerminated bool) {
		err = this.receiver.OnResponseChunkReceived(id, isTerminated, data)
	})
	return err
}

func (this *Protocol) OnZeroLengthMessageReceived(messageId int, messageType internal.MessageType) (err error) {
	switch messageType {
	case internal.MessageTypeCancel:
		return this.receiver.OnCancelReceived(messageId)
	case internal.MessageTypeCancelAck:
		this.requestRules.TryReceiveCancelAck(messageId, func(id int) {
			err = this.receiver.OnCancelAckReceived(messageId)
		})
	case internal.MessageTypeEmptyResponse:
		// TODO: Check for ping response
		return this.receiver.OnEmptyResponseReceived(messageId)
	case internal.MessageTypeRequestEmptyTermination:
		// TODO: Check for ping
		return this.receiver.OnPingReceived(messageId)
	default:
		// TODO: Bug
	}
	return err
}

// Internal

func (this *Protocol) feedNegotiator(incomingStreamData []byte) (remainingData []byte, err error) {
	// fmt.Printf("### P %p: Feeding %v bytes to negotiator\n", this, len(incomingStreamData))
	remainingData = incomingStreamData
	if remainingData, err = this.negotiator.Feed(remainingData); err != nil {
		return nil, err
	}
	if !this.negotiator.IsNegotiationComplete() {
		if len(remainingData) != 0 {
			return nil, fmt.Errorf("INTERNAL BUG: %v bytes in incoming stream, but negotiation still not complete", len(incomingStreamData))
		}
		return remainingData, nil
	}

	this.finishEarlyInitialization()

	return remainingData, nil
}

func (this *Protocol) finishEarlyInitialization() {
	if !this.hasFinishedEarlyInitialization {
		this.hasFinishedEarlyInitialization = true
		this.decoder.Init(this.negotiator.LengthBits, this.negotiator.IdBits, this)
		this.requestRules.Init(internal.NewIdPool(this.negotiator.IdBits))
		this.sender.OnAbleToSend()
	}
}

func (this *Protocol) sendRawMessage(priority int, data []byte) error {
	return this.sender.OnMessageChunkToSend(priority, data)
}

func (this *Protocol) newEmptyMessageHeader(id int, messageType internal.MessageType) []byte {
	var header internal.MessageHeader
	header.Init(this.negotiator.LengthBits, this.negotiator.IdBits)
	header.SetIdAndType(id, messageType)
	return header.Encoded.Data
}

func (this *Protocol) cancelAck(messageId int) error {
	return this.OnMessageChunkToSend(PriorityOOB, messageId, this.newEmptyMessageHeader(messageId, internal.MessageTypeCancelAck))
}

func (this *Protocol) pingAck(messageId int) error {
	return this.OnMessageChunkToSend(PriorityOOB, messageId, this.newEmptyMessageHeader(messageId, internal.MessageTypeEmptyResponse))
}
