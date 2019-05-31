package streamux

import (
	"fmt"
)

type messageDecoder struct {
	header             messageHeader
	remainingByteCount int
	receiver           MessageReceiver
}

// API

func newMessageDecoder(lengthBits int, idBits int, receiver MessageReceiver) *messageDecoder {
	this := new(messageDecoder)
	this.Init(lengthBits, idBits, receiver)
	return this
}

func (this *messageDecoder) Init(lengthBits int, idBits int, receiver MessageReceiver) {
	this.header.Init(lengthBits, idBits)
	this.receiver = receiver
	this.reset()
}

func (this *messageDecoder) Feed(incomingStreamData []byte) (remainingData []byte, err error) {
	// fmt.Printf("### D %p: feed id %v. Data length %v. Is header decoded: %v\n", this, this.id, len(incomingStreamData), this.isHeaderDecoded)
	remainingData = incomingStreamData

	if !this.header.IsDecoded() {
		remainingData = this.header.Feed(remainingData)
		if !this.header.IsDecoded() {
			if len(remainingData) != 0 {
				return remainingData, fmt.Errorf("INTERNAL BUG: %v bytes in incoming stream, but header still not decoded", len(remainingData))
			}
			return remainingData, nil
		}

		this.remainingByteCount = this.header.Length
		switch this.header.MessageType {
		case messageTypeCancel:
			// fmt.Printf("### Cancel\n")
			this.receiver.OnCancelReceived(this.header.Id)
			return remainingData, nil
		case messageTypeCancelAck:
			// fmt.Printf("### Cancel Ack\n")
			this.receiver.OnCancelAckReceived(this.header.Id)
			return remainingData, nil
		case messageTypeRequestEmptyTermination:
			// fmt.Printf("### Ping\n")
			this.receiver.OnPingReceived(this.header.Id)
			return remainingData, nil
		case messageTypeEmptyResponse:
			this.receiver.OnEmptyResponseReceived(this.header.Id)
			return remainingData, nil
		default:
		}
	}

	var decodedData []byte
	for len(remainingData) > 0 && !this.isMessageChunkComplete() {
		decodedData, remainingData = consumeBytes(this.remainingByteCount, remainingData)
		this.remainingByteCount -= len(decodedData)
		// fmt.Printf("#### D %p: Message data. Length %v\n", this, len(decodedData))
		if err := this.notifyMessageData(decodedData); err != nil {
			fmt.Printf("ERROR %v\n", err)
			return remainingData, err
		}
	}
	return remainingData, nil
}

// Internal

func (this *messageDecoder) reset() {
	this.header.ClearEncoded()
}

func (this *messageDecoder) isMessageChunkComplete() bool {
	return this.remainingByteCount == 0
}

func (this *messageDecoder) notifyMessageData(chunk []byte) error {
	if this.header.IsResponse {
		if err := this.receiver.OnResponseChunkReceived(this.header.Id, this.header.IsEndOfMessage, chunk); err != nil {
			return err
		}
	} else {
		if err := this.receiver.OnRequestChunkReceived(this.header.Id, this.header.IsEndOfMessage, chunk); err != nil {
			return err
		}
	}
	if this.isMessageChunkComplete() {
		this.reset()
	}
	return nil
}
