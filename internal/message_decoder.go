package internal

import (
	"fmt"

	"github.com/kstenerud/go-streamux/internal/buffer"
)

type MessageDecoder struct {
	header             MessageHeader
	remainingByteCount int
	receiver           InternalMessageReceiver
}

// API

func NewMessageDecoder(idBits int, lengthBits int, receiver InternalMessageReceiver) *MessageDecoder {
	this := new(MessageDecoder)
	this.Init(idBits, lengthBits, receiver)
	return this
}

func (this *MessageDecoder) Init(idBits int, lengthBits int, receiver InternalMessageReceiver) {
	this.header.Init(idBits, lengthBits)
	this.receiver = receiver
	this.reset()
}

func (this *MessageDecoder) Feed(incomingStreamData []byte) (remainingData []byte, err error) {
	// fmt.Printf("### D %p: feed id %v. Data length %v. Is header decoded: %v\n", this, this.header.Id, len(incomingStreamData), this.header.IsDecoded())
	remainingData = incomingStreamData

	if !this.header.IsDecoded() {
		remainingData, err = this.header.Feed(remainingData)
		if err != nil {
			return remainingData, err
		}
		if !this.header.IsDecoded() {
			if len(remainingData) != 0 {
				return remainingData, fmt.Errorf("Internal bug: MessageDecoder.Feed: %v bytes remain in incoming stream, but header still not decoded", len(remainingData))
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
		decodedData, remainingData = buffer.ConsumeBytes(this.remainingByteCount, remainingData)
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

func (this *MessageDecoder) reset() {
	this.header.ClearEncoded()
}

func (this *MessageDecoder) isMessageChunkComplete() bool {
	return this.remainingByteCount == 0
}

func (this *MessageDecoder) notifyMessageData(chunk []byte) error {
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
