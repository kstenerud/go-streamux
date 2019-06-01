package streamux

import (
	"fmt"
	"math"
	"time"

	"github.com/kstenerud/go-streamux/common"
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
	idPool                         internal.IdPool
	sender                         common.MessageSender
	receiver                       common.MessageReceiver
	pingSendTimes                  map[int]time.Time
}

// API

func NewProtocol(lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	idMinBits int, idMaxBits int, idRecommendBits int, requestQuickInit bool,
	allowQuickInit bool, sender common.MessageSender,
	receiver common.MessageReceiver) *Protocol {

	this := new(Protocol)
	this.Init(lengthMinBits, lengthMaxBits, lengthRecommendBits,
		idMinBits, idMaxBits, idRecommendBits,
		requestQuickInit, allowQuickInit,
		sender, receiver)

	return this
}

func (this *Protocol) Init(lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	idMinBits int, idMaxBits int, idRecommendBits int, requestQuickInit bool,
	allowQuickInit bool, sender common.MessageSender,
	receiver common.MessageReceiver) {

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
func (this *Protocol) BeginRequest(priority int) (*SendableMessage, error) {
	if !this.negotiator.CanSendMessages() {
		return nil, fmt.Errorf("Can't send messages: %v", this.negotiator.ExplainFailure())
	}

	isResponse := false
	return newSendableMessage(this, priority, this.idPool.AllocateId(),
		this.negotiator.LengthBits, this.negotiator.IdBits, isResponse), nil
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

func (this *Protocol) Cancel(messageId int) error {
	return this.OnMessageChunkToSend(PriorityOOB, messageId, this.newEmptyMessageHeader(messageId, internal.MessageTypeCancel))
}

func (this *Protocol) Ping() (id int, err error) {
	id = this.idPool.AllocateId()
	defer this.idPool.DeallocateId(id)

	if err := this.OnMessageChunkToSend(PriorityOOB, id, this.newEmptyMessageHeader(id, internal.MessageTypeRequestEmptyTermination)); err != nil {
		return 0, err
	}
	this.pingSendTimes[id] = time.Now()
	// TODO: Store outstanding ping for response ack
	// TODO: Record ping time?

	return id, nil
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
	err := this.sendRawMessage(priority, data)
	// TODO: Deal with response & canceling message so we get the ID back.
	// For now, do it wrong: Auto deallocate at the end of a message.
	terminationBit := data[0] & 1
	if terminationBit == 1 {
		this.idPool.DeallocateId(messageId)
	}
	return err
}

func (this *Protocol) OnRequestChunkReceived(messageId int, isEnd bool, data []byte) error {
	// TODO: ???
	return this.receiver.OnRequestChunkReceived(messageId, isEnd, data)
}

func (this *Protocol) OnResponseChunkReceived(messageId int, isEnd bool, data []byte) error {
	// TODO: move to available pool if ended?
	return this.receiver.OnResponseChunkReceived(messageId, isEnd, data)
}

func (this *Protocol) OnZeroLengthMessageReceived(messageId int, messageType internal.MessageType) error {
	switch messageType {
	case internal.MessageTypeCancel:
		// TODO: Move to cancel pool
		return this.receiver.OnCancelReceived(messageId)
	case internal.MessageTypeCancelAck:
		// TODO: Move to available pool
		return this.receiver.OnCancelAckReceived(messageId)
	case internal.MessageTypeEmptyResponse:
		// TODO: Check for ping response
		return this.receiver.OnEmptyResponseReceived(messageId)
		// var latency time.Duration
		// return this.receiver.OnPingAckReceived(messageId, latency)
	case internal.MessageTypeRequestEmptyTermination:
		// TODO: Check for ping
		return this.receiver.OnPingReceived(messageId)
	default:
		// TODO: Bug
	}
	return nil
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
		this.idPool.Init(this.negotiator.IdBits)
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
