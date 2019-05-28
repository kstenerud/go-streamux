package streamux

import (
	"fmt"
)

type messageDecoder_ struct {
	// Session constants
	headerLength int
	maskId       uint32
	maskLength   uint32
	shiftLength  uint

	// Header data
	headerBuffer   []byte
	isResponse     bool
	isEndOfMessage bool
	id             int
	length         int

	bytesRemaining int
	callbacks      MessageReceiveCallbacks
}

func (this *messageDecoder_) reset() {
	this.headerBuffer = this.headerBuffer[:0]
}

func (this *messageDecoder_) isHeaderDecoded() bool {
	return len(this.headerBuffer) == this.headerLength
}

func (this *messageDecoder_) isMessageChunkComplete() bool {
	return this.bytesRemaining == 0
}

func (this *messageDecoder_) decodeHeader(incomingStreamData []byte) []byte {
	this.headerBuffer, incomingStreamData = fillBuffer(this.headerLength, this.headerBuffer, incomingStreamData)

	if this.isHeaderDecoded() {
		var header uint32
		for i := this.headerLength - 1; i >= 0; i-- {
			header <<= 8
			header |= uint32(this.headerBuffer[i])
		}
		this.isEndOfMessage = (header & 1) == 1
		this.isResponse = ((header >> shiftResponseBit) & 1) == 1
		this.id = int((header >> shiftId) & this.maskId)
		this.length = int((header >> this.shiftLength) & this.maskLength)
		this.bytesRemaining = this.length
	}

	return incomingStreamData
}

func (this *messageDecoder_) Init(headerLength int, lengthBits int, idBits int) {
	this.headerLength = headerLength
	this.maskId = (1 << uint32(idBits)) - 1
	this.shiftLength = shiftId + uint(idBits)
	this.maskLength = (1 << uint32(lengthBits)) - 1
	this.headerBuffer = make([]byte, 0, headerLength)
	this.reset()
}

func (this *messageDecoder_) notifyMessageData(chunk []byte) error {
	if this.isResponse {
		if err := this.callbacks.OnResponseChunkReceived(this.id, this.isEndOfMessage, chunk); err != nil {
			return err
		}
	} else {
		if err := this.callbacks.OnRequestChunkReceived(this.id, this.isEndOfMessage, chunk); err != nil {
			return err
		}
	}
	if this.isMessageChunkComplete() {
		this.reset()
	}
	return nil
}

func (this *messageDecoder_) Feed(incomingStreamData []byte) error {
	if !this.isHeaderDecoded() {
		incomingStreamData = this.decodeHeader(incomingStreamData)
		if !this.isHeaderDecoded() {
			if len(incomingStreamData) != 0 {
				return fmt.Errorf("INTERNAL BUG: %v bytes in incoming stream, but header still not decoded", len(incomingStreamData))
			}
			return nil
		}
	}

	var decodedData []byte

	if this.isMessageChunkComplete() {
		return this.notifyMessageData(decodedData)
	}

	for len(incomingStreamData) > 0 && !this.isMessageChunkComplete() {
		decodedData, incomingStreamData = useBytes(this.bytesRemaining, incomingStreamData)
		this.bytesRemaining -= len(decodedData)
		if err := this.notifyMessageData(decodedData); err != nil {
			return err
		}
	}
	return nil
}
