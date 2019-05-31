package streamux

import (
	"fmt"
	"math"
	"time"
)

const ProtocolVersion = 1

const PriorityMax = math.MaxInt32
const PriorityOOB = PriorityMax

type MessageReceiveCallbacks interface {
	OnRequestChunkReceived(messageId int, isEnd bool, data []byte) error
	OnResponseChunkReceived(messageId int, isEnd bool, data []byte) error
	OnPingReceived(id int)
	OnPingAckReceived(id int, latency time.Duration)
	OnCancelReceived(messageId int)
	OnCancelAckReceived(messageId int)
	OnEmptyResponseReceived(id int)
}

type MessageSendCallbacks interface {
	OnNegotiationFailed()
	OnAbleToSend()
	OnMessageChunkToSend(priority int, data []byte) error
}

type MessageSender interface {
	OnMessageChunkToSend(priority int, messageId int, data []byte) error
}

type Protocol struct {
	hasBegunInitialization         bool
	hasFinishedEarlyInitialization bool
	negotiator                     *negotiator_
	decoder                        *messageDecoder_
	idPool                         *idPool
	callbacks                      MessageSendCallbacks
	decoderCallbacks               MessageReceiveCallbacks
	pingSendTimes                  map[int]time.Time
}

// Forwarding MessageReceiveCallbacks interface
func (this *Protocol) OnRequestChunkReceived(messageId int, isEnd bool, data []byte) error {
	return this.decoderCallbacks.OnRequestChunkReceived(messageId, isEnd, data)
}

func (this *Protocol) OnResponseChunkReceived(messageId int, isEnd bool, data []byte) error {
	return this.decoderCallbacks.OnResponseChunkReceived(messageId, isEnd, data)
}

func (this *Protocol) OnPingReceived(id int) {
	this.decoderCallbacks.OnPingReceived(id)
}

func (this *Protocol) OnPingAckReceived(id int, latency time.Duration) {
	this.decoderCallbacks.OnPingAckReceived(id, latency)
}

func (this *Protocol) OnCancelReceived(messageId int) {
	this.decoderCallbacks.OnCancelReceived(messageId)
}

func (this *Protocol) OnCancelAckReceived(messageId int) {
	this.decoderCallbacks.OnCancelAckReceived(messageId)
}

func (this *Protocol) OnEmptyResponseReceived(id int) {
	this.decoderCallbacks.OnEmptyResponseReceived(id)
}

//

func (this *Protocol) sendRawMessage(priority int, data []byte) error {
	return this.callbacks.OnMessageChunkToSend(priority, data)
}

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
	// fmt.Printf("### P %p: Feeding %v bytes to negotiator\n", this, len(incomingStreamData))
	var err error
	if incomingStreamData, err = this.negotiator.Feed(incomingStreamData); err != nil {
		this.callbacks.OnNegotiationFailed()
		return nil, err
	}
	if !this.negotiator.IsNegotiationComplete() {
		if len(incomingStreamData) != 0 {
			return nil, fmt.Errorf("INTERNAL BUG: %v bytes in incoming stream, but negotiation still not complete", len(incomingStreamData))
		}
		return incomingStreamData, nil
	}

	this.finishEarlyInitialization()

	return incomingStreamData, nil
}

func (this *Protocol) finishEarlyInitialization() {
	if !this.hasFinishedEarlyInitialization {
		this.hasFinishedEarlyInitialization = true
		this.decoder = newMessageDecoder(this.negotiator.LengthBits, this.negotiator.IdBits, this)
		this.idPool = newIdPool(this.negotiator.IdBits)
		this.callbacks.OnAbleToSend()
	}
}

func (this *Protocol) BeginInitialization() error {
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

func (this *Protocol) SendRequest(priority int, contents []byte) (id int, err error) {
	message, err := this.BeginRequest(priority)
	if err != nil {
		return 0, err
	}
	if err = message.Add(contents); err != nil {
		return message.Id, err
	}
	return message.Id, message.End()
}

func (this *Protocol) SendResponse(priority int, responseToId int, contents []byte) error {
	message, err := this.BeginResponse(priority, responseToId)
	if err != nil {
		return err
	}
	if err = message.Add(contents); err != nil {
		return err
	}
	return message.End()
}

func (this *Protocol) BeginRequest(priority int) (*SendableMessage, error) {
	if !this.negotiator.CanSendMessages() {
		return nil, fmt.Errorf("Can't send messages: %v", this.negotiator.ExplainFailure())
	}

	isResponse := false
	return newSendableMessage(this, priority, this.idPool.AllocateId(),
		this.negotiator.LengthBits, this.negotiator.IdBits, isResponse), nil
}

func (this *Protocol) BeginResponse(priority int, responseToId int) (*SendableMessage, error) {
	if !this.negotiator.CanSendMessages() {
		return nil, fmt.Errorf("Can't send messages: %v", this.negotiator.ExplainFailure())
	}

	isResponse := true
	return newSendableMessage(this, priority, responseToId,
		this.negotiator.LengthBits, this.negotiator.IdBits, isResponse), nil
}

func (this *Protocol) newEmptyMessageHeader(id int, messageType messageType) []byte {
	var header messageHeader_
	header.Init(this.negotiator.LengthBits, this.negotiator.IdBits)
	header.SetIdAndType(id, messageType)
	return header.Encoded
}

func (this *Protocol) Cancel(messageId int) error {
	return this.OnMessageChunkToSend(PriorityOOB, messageId, this.newEmptyMessageHeader(messageId, messageTypeCancel))
}

func (this *Protocol) cancelAck(messageId int) error {
	return this.OnMessageChunkToSend(PriorityOOB, messageId, this.newEmptyMessageHeader(messageId, messageTypeCancelAck))
}

func (this *Protocol) Ping() (id int, err error) {
	id = this.idPool.AllocateId()
	defer this.idPool.DeallocateId(id)

	if err := this.OnMessageChunkToSend(PriorityOOB, id, this.newEmptyMessageHeader(id, messageTypePing)); err != nil {
		return 0, err
	}
	this.pingSendTimes[id] = time.Now()
	// TODO: Store outstanding ping for response ack
	// TODO: Record ping time?

	return id, nil
}

func (this *Protocol) pingAck(messageId int) error {
	return this.OnMessageChunkToSend(PriorityOOB, messageId, this.newEmptyMessageHeader(messageId, messageTypeEmptyResponse))
}

func (this *Protocol) Feed(incomingStreamData []byte) error {
	remainingData := incomingStreamData
	// fmt.Printf("### P %p: Feed %v bytes. Negotiation complete: %v\n", this, len(remainingData), this.negotiator.IsNegotiationComplete())
	if !this.negotiator.IsNegotiationComplete() {
		var err error
		if remainingData, err = this.feedNegotiator(remainingData); err != nil {
			return err
		}
		if len(remainingData) == 0 {
			return nil
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
