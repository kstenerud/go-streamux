package streamux

import (
	"fmt"
)

type messageDecoder struct {
	header             messageHeader
	remainingByteCount int
	receiver           internalMessageReceiver
}

// API

func newMessageDecoder(lengthBits int, idBits int, receiver internalMessageReceiver) *messageDecoder {
	this := new(messageDecoder)
	this.Init(lengthBits, idBits, receiver)
	return this
}

func (this *messageDecoder) Init(lengthBits int, idBits int, receiver internalMessageReceiver) {
	this.header.Init(lengthBits, idBits)
	this.receiver = receiver
	this.reset()
}

func (this *messageDecoder) Feed(incomingStreamData []byte) (remainingData []byte, err error) {
	// fmt.Printf("### D %p: feed id %v. Data length %v. Is header decoded: %v\n", this, this.header.Id, len(incomingStreamData), this.header.IsDecoded())
	remainingData = incomingStreamData

	if !this.header.IsDecoded() {
		remainingData = this.header.Feed(remainingData)
		if !this.header.IsDecoded() {
			if len(remainingData) != 0 {
				return remainingData, fmt.Errorf("INTERNAL BUG: %v bytes in incoming stream, but header still not decoded", len(remainingData))
			}
			return remainingData, nil
		} else if this.header.Length == 0 {
			err = this.receiver.OnZeroLengthMessageReceived(this.header.Id, this.header.MessageType)
			this.reset()
			return remainingData, err
		}

		this.remainingByteCount = this.header.Length
	}

	// fmt.Printf("#### D %p: Remaining data %v, chunk complete %v\n", this, len(remainingData), this.isMessageChunkComplete())
	for len(remainingData) > 0 && !this.isMessageChunkComplete() {
		var decodedData []byte
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
