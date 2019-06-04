package streamux

import (
	// "encoding/hex"
	"fmt"
	"math"
	"time"

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
	requestStateMachine            internal.RequestStateMachine
	sender                         MessageSender
	receiver                       MessageReceiver
	activeIncomingRequests         map[int]bool
	activeOutgoingPings            map[int]time.Time
}

// API

func NewProtocol(idMinBits int, idMaxBits int, idRecommendBits int,
	lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	requestQuickInit bool, allowQuickInit bool, sender MessageSender,
	receiver MessageReceiver) *Protocol {

	this := new(Protocol)
	this.Init(idMinBits, idMaxBits, idRecommendBits,
		lengthMinBits, lengthMaxBits, lengthRecommendBits,
		requestQuickInit, allowQuickInit,
		sender, receiver)

	return this
}

func (this *Protocol) Init(idMinBits int, idMaxBits int, idRecommendBits int,
	lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	requestQuickInit bool, allowQuickInit bool, sender MessageSender,
	receiver MessageReceiver) {

	this.negotiator.Init(ProtocolVersion,
		idMinBits, idMaxBits, idRecommendBits,
		lengthMinBits, lengthMaxBits, lengthRecommendBits,
		requestQuickInit, allowQuickInit)
	this.sender = sender
	this.receiver = receiver
	this.activeIncomingRequests = make(map[int]bool)
	this.activeOutgoingPings = make(map[int]time.Time)
}

func (this *Protocol) SendInitialization() error {
	if !this.hasBegunInitialization {
		this.hasBegunInitialization = true
		if this.negotiator.CanSendMessages() {
			this.finishEarlyInitialization()
		}
		// fmt.Printf("### P %p: Sending init message\n", this)
		return this.sendRawMessage(PriorityOOB, -1, this.negotiator.BuildInitializeMessage())
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

	err = this.requestStateMachine.TryBeginRequest(func(id int) {
		isResponse := false
		message = newSendableMessage(this, priority, id,
			this.negotiator.IdBits, this.negotiator.LengthBits, isResponse)
		// fmt.Printf("### P %p: Begin request. New SM %p\n", this, message)
	})
	return message, err
}

// Advanced API. The SendableMessage returned by this method can be used to incrementally
// add data to the message being sent. Data will be queued and sent as it fills the maximum chunk length.
func (this *Protocol) BeginResponse(priority int, responseToId int) (*SendableMessage, error) {
	if !this.negotiator.CanSendMessages() {
		return nil, fmt.Errorf("Can't send messages: %v", this.negotiator.ExplainFailure())
	}

	// fmt.Printf("### P %p: Begin response id %v\n", this, responseToId)
	isResponse := true
	return newSendableMessage(this, priority, responseToId,
		this.negotiator.IdBits, this.negotiator.LengthBits, isResponse), nil
}

// Cancel a message/operation. If the operation is still active on the other peer,
// it will be canceled and all remaining queued message chunks of that id removed.
// You will always receive a cancel ack notification, even if no such operation exists.
func (this *Protocol) Cancel(messageId int) (err error) {
	outerErr := this.requestStateMachine.TryCancelRequest(messageId, func(id int) {
		// fmt.Printf("### P %p: Send cancel %v\n", this, messageId)
		err = this.sendRawMessage(PriorityOOB, id, this.newEmptyMessageHeader(id, internal.MessageTypeCancel))
	})
	if outerErr != nil {
		err = outerErr
	}
	return err
}

// Send a ping to the other peer. You will eventually receive a ping ack notification
// with a report on how long it took to receive a reply from the time that the
// ping was queued for sending.
func (this *Protocol) Ping() (id int, err error) {
	outerErr := this.requestStateMachine.TryPing(func(newId int) {
		id = newId
		// fmt.Printf("### P %p: Send ping %v\n", this, id)
		if err = this.sendRawMessage(PriorityOOB, id, this.newEmptyMessageHeader(id, internal.MessageTypeRequestEmptyTermination)); err == nil {
			this.activeOutgoingPings[id] = time.Now()
		}
	})
	if outerErr != nil {
		err = outerErr
	}
	return id, err
}

// Feed data from the other peer into this protocol. This method will always
// either consume all bytes, or return an error.
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

// Internal callback
func (this *Protocol) OnRequestChunkToSend(priority int, messageId int, isEnd bool, data []byte) (err error) {
	// fmt.Printf("### P %p: Send request chunk id %v, data %v, term %v\n", this, messageId, len(data), isEnd)
	outerErr := this.requestStateMachine.TrySendRequestChunk(messageId, isEnd, func(id int, isTerminated bool) {
		err = this.sendRawMessage(priority, id, data)
	})
	if outerErr != nil {
		err = outerErr
	}
	return err
}

// Internal callback
func (this *Protocol) OnResponseChunkReceived(messageId int, isEnd bool, data []byte) (err error) {
	// fmt.Printf("### P %p: Receive response chunk id %v, term %v, data %v\n", this, messageId, isEnd, len(data))
	outerErr := this.requestStateMachine.TryReceiveResponseChunk(messageId, isEnd, func(id int, isTerminated bool) {
		err = this.receiver.OnResponseChunkReceived(id, isTerminated, data)
		// fmt.Printf("### P %p: Try receive did call with err %v\n", this, err)
	})
	if outerErr != nil {
		err = outerErr
	}
	return err
}

// Internal callback
func (this *Protocol) OnRequestChunkReceived(messageId int, isEnd bool, data []byte) error {
	// fmt.Printf("### P %p: Receive request chunk id %v, term %v, data %v\n", this, messageId, isEnd, len(data))
	if isEnd {
		delete(this.activeIncomingRequests, messageId)
	} else {
		this.activeIncomingRequests[messageId] = true
	}
	return this.receiver.OnRequestChunkReceived(messageId, isEnd, data)
}

// Internal callback
func (this *Protocol) OnResponseChunkToSend(priority int, messageId int, isEnd bool, data []byte) (err error) {
	// fmt.Printf("### P %p: Send response chunk id %v, data %v, term %v\n", this, messageId, len(data), isEnd)
	return this.sendRawMessage(priority, messageId, data)
}

// Internal callback
func (this *Protocol) OnZeroLengthMessageReceived(messageId int, messageType internal.MessageType) (err error) {
	// fmt.Printf("### P %p: Zero length message id %v, type %v\n", this, messageId, messageType)
	switch messageType {
	default:
		err = fmt.Errorf("Internal bug: Protocol.OnZeroLengthMessageReceived: Unexpected message type %v", messageType)
	case internal.MessageTypeCancel:
		if err = this.receiver.OnCancelReceived(messageId); err == nil {
			err = this.cancelAck(messageId)
		}
	case internal.MessageTypeCancelAck:
		outerErr := this.requestStateMachine.TryReceiveCancelAck(messageId, func(id int) {
			err = this.receiver.OnCancelAckReceived(messageId)
		})
		if outerErr != nil {
			err = outerErr
		}
	case internal.MessageTypeEmptyResponse:
		if startTime, exists := this.activeOutgoingPings[messageId]; exists {
			err = this.receiver.OnPingAckReceived(messageId, time.Now().Sub(startTime))
		} else {
			err = this.receiver.OnEmptyResponseReceived(messageId)
		}
	case internal.MessageTypeRequestEmptyTermination:
		if _, isActive := this.activeIncomingRequests[messageId]; isActive {
			isTerminated := true
			err = this.OnRequestChunkReceived(messageId, isTerminated, []byte{})
		} else {
			delete(this.activeOutgoingPings, messageId)
			if err = this.pingAck(messageId); err == nil {
				err = this.receiver.OnPingReceived(messageId)
			}
		}
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
			return nil, fmt.Errorf("Internal bug: Protocol.feedNegotiator: %v bytes in incoming stream, but negotiation still not complete", len(incomingStreamData))
		}
		return remainingData, nil
	}

	this.finishEarlyInitialization()

	return remainingData, nil
}

func (this *Protocol) finishEarlyInitialization() {
	if !this.hasFinishedEarlyInitialization {
		this.hasFinishedEarlyInitialization = true
		this.decoder.Init(this.negotiator.IdBits, this.negotiator.LengthBits, this)
		this.requestStateMachine.Init(internal.NewIdPool(this.negotiator.IdBits))
		this.sender.OnAbleToSend()
	}
}

func (this *Protocol) sendRawMessage(priority int, messageId int, data []byte) error {
	// fmt.Printf("### P %p: Send raw message id %v, data %v\n", this, messageId, len(data))
	return this.sender.OnMessageChunkToSend(priority, messageId, data)
}

func (this *Protocol) newEmptyMessageHeader(id int, messageType internal.MessageType) []byte {
	var header internal.MessageHeader
	header.Init(this.negotiator.IdBits, this.negotiator.LengthBits)
	header.SetIdAndType(id, messageType)
	// fmt.Printf("### P %p: Sending empty message [%v]\n", this, hex.EncodeToString(header.Encoded.Data))
	return header.Encoded.Data
}

func (this *Protocol) cancelAck(id int) error {
	// fmt.Printf("### P %p: Send cancel ack id %v\n", this, id)
	return this.sendRawMessage(PriorityOOB, id, this.newEmptyMessageHeader(id, internal.MessageTypeCancelAck))
}

func (this *Protocol) pingAck(id int) error {
	// fmt.Printf("### P %p: Send ping ack id %v\n", this, id)
	return this.sendRawMessage(PriorityOOB, id, this.newEmptyMessageHeader(id, internal.MessageTypeEmptyResponse))
}
