package streamux

// TODO:
// - OOB messages
// - ???

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
	OnPingReceived()
	OnPingAckReceived(latency time.Duration)
	OnCancelReceived(messageId int)
	OnCancelAckReceived(messageId int)
}

type MessageSendCallbacks interface {
	OnNegotiationFailed()
	OnAbleToSend()
	OnMessageChunkToSend(priority int, data []byte) error
}

type Protocol struct {
	hasBegunInitialization         bool
	hasFinishedEarlyInitialization bool
	negotiator                     *negotiator_
	decoder                        *messageDecoder_
	idPool                         *idPool
	callbacks                      MessageSendCallbacks
	decoderCallbacks               MessageReceiveCallbacks
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
		this.decoder = newMessageDecoder(this.negotiator.HeaderLength, this.negotiator.LengthBits, this.negotiator.IdBits, this.decoderCallbacks)
		this.idPool = newIdPool(this.negotiator.IdBits)
		this.callbacks.OnAbleToSend()
	}
}

func (this *Protocol) feedDecoder(incomingStreamData []byte) error {
	// fmt.Printf("### P %p: Feeding %v bytes to decoder\n", this, len(incomingStreamData))
	return this.decoder.Feed(incomingStreamData)
}

func (this *Protocol) sendMessageChunk(priority int, chunk []byte) error {
	return this.callbacks.OnMessageChunkToSend(priority, chunk)
}

func (this *Protocol) BeginInitialization() error {
	if !this.hasBegunInitialization {
		this.hasBegunInitialization = true
		if this.negotiator.CanSendMessages() {
			this.finishEarlyInitialization()
		}
		// fmt.Printf("### P %p: Sending init message\n", this)
		return this.sendMessageChunk(PriorityOOB, this.negotiator.BuildInitializeMessage())
	}
	return nil
}

func (this *Protocol) SendMessage(priority int, contents []byte) (id int, err error) {
	message, err := this.BeginMessage(priority)
	if err != nil {
		return 0, err
	}
	defer message.Close()
	isEndOfData := true
	err = message.AddData(contents, isEndOfData)
	return message.Id, err
}

func (this *Protocol) SendResponseMessage(priority int, responseToId int, contents []byte) error {
	message, err := this.BeginResponseMessage(priority, responseToId)
	if err != nil {
		return err
	}
	defer message.Close()
	isEndOfData := true
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
	return newSendableMessage(nil, this.callbacks, priority, responseToId,
		this.negotiator.HeaderLength, this.negotiator.LengthBits,
		this.negotiator.IdBits, isResponse), nil
}

func (this *Protocol) emptyMessageHeader(id int, responseBit int, terminationBit int) []byte {
	headerFields := uint32(terminationBit) |
		uint32(responseBit)<<shiftResponseBit |
		uint32(id)<<shiftId

	header := make([]byte, this.negotiator.HeaderLength)
	for i := 0; i < len(header); i++ {
		header[i] = byte(headerFields)
		headerFields >>= 8
	}
	return header
}

func (this *Protocol) Cancel(messageId int) error {
	return this.callbacks.OnMessageChunkToSend(PriorityOOB, this.emptyMessageHeader(messageId, 0, 0))
}

func (this *Protocol) cancelAck(messageId int) error {
	return this.callbacks.OnMessageChunkToSend(PriorityOOB, this.emptyMessageHeader(messageId, 1, 0))
}

func (this *Protocol) Ping() error {
	id := this.idPool.AllocateId()
	defer this.idPool.DeallocateId(id)

	return this.callbacks.OnMessageChunkToSend(PriorityOOB, this.emptyMessageHeader(id, 0, 1))
	// TODO: Store outstanding ping for response ack
	// TODO: Record ping time?
}

func (this *Protocol) pingAck(messageId int) error {
	return this.callbacks.OnMessageChunkToSend(PriorityOOB, this.emptyMessageHeader(messageId, 1, 1))
}

func (this *Protocol) Feed(incomingStreamData []byte) error {
	// fmt.Printf("### P %p: Feed %v bytes. Negotiation complete: %v\n", this, len(incomingStreamData), this.negotiator.IsNegotiationComplete())
	if !this.negotiator.IsNegotiationComplete() {
		var err error
		if incomingStreamData, err = this.feedNegotiator(incomingStreamData); err != nil {
			return err
		}
		if len(incomingStreamData) == 0 {
			return nil
		}
	}

	if this.negotiator.CanReceiveMessages() {
		return this.feedDecoder(incomingStreamData)
	} else if len(incomingStreamData) > 0 {
		return fmt.Errorf("Can't receive messages: %v", this.negotiator.ExplainFailure())
	}

	return nil
}
