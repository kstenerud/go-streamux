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

	isHeaderDecoded bool
	bytesRemaining  int
	callbacks       MessageReceiveCallbacks
}

func (this *messageDecoder_) reset() {
	this.headerBuffer = this.headerBuffer[:0]
	this.isHeaderDecoded = false
}

func (this *messageDecoder_) isMessageChunkComplete() bool {
	return this.bytesRemaining == 0
}

func (this *messageDecoder_) decodeHeader(incomingStreamData []byte) []byte {
	// fmt.Printf("### D %p: Decode header: headerLength %v, headerBuffer %v, incoming %v\n", this, this.headerLength, len(this.headerBuffer), len(incomingStreamData))

	this.headerBuffer, incomingStreamData = fillBuffer(this.headerLength, this.headerBuffer, incomingStreamData)

	if len(this.headerBuffer) == this.headerLength {
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
		this.isHeaderDecoded = true
		// fmt.Printf("### Decode header: Length %v, shift %v\n", this.length, this.shiftLength)
		// fmt.Printf("### D %p: terminated: %v\n", this, this.isEndOfMessage)
	}

	return incomingStreamData
}

func newMessageDecoder(headerLength int, lengthBits int, idBits int, callbacks MessageReceiveCallbacks) *messageDecoder_ {
	this := new(messageDecoder_)
	this.Init(headerLength, lengthBits, idBits, callbacks)
	return this
}

func (this *messageDecoder_) Init(headerLength int, lengthBits int, idBits int, callbacks MessageReceiveCallbacks) {
	if headerLength == 0 {
		panic(fmt.Errorf("Internal bug: Should not have header length 0"))
	}
	this.headerLength = headerLength
	this.maskId = (1 << uint32(idBits)) - 1
	this.shiftLength = shiftId + uint(idBits)
	this.maskLength = (1 << uint32(lengthBits)) - 1
	this.headerBuffer = make([]byte, 0, headerLength)
	this.callbacks = callbacks
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
	// fmt.Printf("### D %p: feed id %v. Data length %v. Is header decoded: %v\n", this, this.id, len(incomingStreamData), this.isHeaderDecoded)
	if !this.isHeaderDecoded {
		incomingStreamData = this.decodeHeader(incomingStreamData)
		if !this.isHeaderDecoded {
			if len(incomingStreamData) != 0 {
				return fmt.Errorf("INTERNAL BUG: %v bytes in incoming stream, but header still not decoded", len(incomingStreamData))
			}
			return nil
		}
	}

	var decodedData []byte

	if this.isMessageChunkComplete() {
		// fmt.Printf("#### D %p: Message chunk complete. Length %v\n", this, len(decodedData))
		return this.notifyMessageData(decodedData)
	}

	for len(incomingStreamData) > 0 && !this.isMessageChunkComplete() {
		decodedData, incomingStreamData = useBytes(this.bytesRemaining, incomingStreamData)
		this.bytesRemaining -= len(decodedData)
		// fmt.Printf("#### D %p: Message data. Length %v\n", this, len(decodedData))
		if err := this.notifyMessageData(decodedData); err != nil {
			fmt.Printf("ERROR %v\n", err)
			return err
		}
	}
	return nil
}
