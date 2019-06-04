package internal

import (
	"fmt"

	"github.com/kstenerud/go-streamux/internal/buffer"
)

const (
	shiftResponseBit = 1
	shiftLength      = 2
)

// MessageHeader encodes and decodes a message header. Parameters to the setters
// are assumed to be valid (this class performs no validation).
type MessageHeader struct {
	// Public data (read only)
	Id             int
	Length         int
	IsResponse     bool
	IsEndOfMessage bool
	MessageType    MessageType
	HeaderLength   int
	MaxChunkLength int
	Encoded        buffer.FeedableBuffer

	// Session constants
	maskId     uint32
	maskLength uint32
	maskUnused uint32
	shiftId    uint

	// Internal
	terminationField uint32
	responseField    uint32
}

// API

func NewMessageHeader(idBits int, lengthBits int) *MessageHeader {
	this := new(MessageHeader)
	this.Init(idBits, lengthBits)
	return this
}

func (this *MessageHeader) Init(idBits int, lengthBits int) {
	this.HeaderLength = calculateHeaderLength(idBits, lengthBits)
	this.maskId = 1<<uint(idBits) - 1
	this.shiftId = shiftLength + uint(lengthBits)
	this.maskLength = 1<<uint(lengthBits) - 1
	this.MaxChunkLength = 1<<uint(lengthBits) - 1
	this.maskUnused = ^(1<<uint(idBits+lengthBits+2) - 1)
	this.Encoded.Init(0, this.HeaderLength, this.HeaderLength)
}

func (this *MessageHeader) SetAll(id int, length int, isResponse bool, isEndOfMessage bool) {
	this.setIdAndResponse(id, isResponse)
	this.setLengthAndTermination(length, isEndOfMessage)
	this.encodeHeader()
	this.updateMessageType()
}

func (this *MessageHeader) SetIdAndType(id int, messageType MessageType) {
	switch messageType {
	case MessageTypeCancel:
		this.IsEndOfMessage = false
		this.IsResponse = false
	case MessageTypeCancelAck:
		this.IsEndOfMessage = false
		this.IsResponse = true
	case MessageTypeRequestEmptyTermination:
		this.IsEndOfMessage = true
		this.IsResponse = false
	case MessageTypeEmptyResponse:
		this.IsEndOfMessage = true
		this.IsResponse = true
	case MessageTypeRequest:
		panic(fmt.Errorf("Cannot use this API to set message type request"))
	case MessageTypeResponse:
		panic(fmt.Errorf("Cannot use this API to set message type response"))
	}
	this.Id = id
	this.Length = 0
	this.MessageType = messageType
	this.encodeHeader()
}

func (this *MessageHeader) SetIdAndResponseNoEncode(id int, isResponse bool) {
	this.setIdAndResponse(id, isResponse)
	this.updateMessageType()
}

func (this *MessageHeader) SetLengthAndTermination(length int, isEndOfMessage bool) {
	this.setLengthAndTermination(length, isEndOfMessage)
	this.encodeHeader()
	this.updateMessageType()
}

func (this *MessageHeader) IsDecoded() bool {
	return this.Encoded.IsFull()
}

func (this *MessageHeader) ClearEncoded() {
	this.Encoded.Minimize()
}

func (this *MessageHeader) Feed(incomingStreamData []byte) (remainingData []byte, err error) {
	remainingData = incomingStreamData
	// fmt.Printf("### MH %p: Feed: headerLength %v, headerBuffer %v, incoming %v\n", this, this.HeaderLength, len(this.Encoded), len(remainingData))

	remainingData = this.Encoded.Feed(remainingData)

	if this.IsDecoded() {
		var headerFields uint32
		for i := this.HeaderLength - 1; i >= 0; i-- {
			headerFields <<= 8
			headerFields |= uint32(this.Encoded.Data[i])
		}
		if headerFields&this.maskUnused != 0 {
			return remainingData, fmt.Errorf("Unused header bits were nonzero (0x%x)", headerFields&this.maskUnused)
		}
		this.IsEndOfMessage = (headerFields & 1) == 1
		this.IsResponse = ((headerFields >> shiftResponseBit) & 1) == 1
		this.Id = int((headerFields >> this.shiftId) & this.maskId)
		this.Length = int((headerFields >> shiftLength) & this.maskLength)
		this.updateMessageType()

		// fmt.Printf("### MH %p: lshift %v, lmask %v, ishift %v, imask %v\n", this, this.shiftLength, this.maskLength, shiftId, this.maskId)
		// fmt.Printf("### MH %p: Decode header %08x: len %v, id %v, resp %v, term %v\n", this, headerFields, this.Length, this.Id, this.IsResponse, this.IsEndOfMessage)
	}

	return remainingData, err
}

// Internal

func boolToUint32(value bool) uint32 {
	if value {
		return 1
	}
	return 0
}

func calculateHeaderLength(idBits, lengthBits int) int {
	totalBits := lengthBits + idBits
	switch {
	case totalBits <= 6:
		return 1
	case totalBits <= 14:
		return 2
	case totalBits <= 22:
		return 3
	}
	return 4
}

func (this *MessageHeader) updateMessageType() {
	// fmt.Printf("### Length %v, response %v, term %v\n", this.Length, this.IsResponse, this.IsEndOfMessage)
	if this.Length > 0 {
		if this.IsResponse {
			this.MessageType = MessageTypeResponse
			return
		}
		this.MessageType = MessageTypeRequest
		return
	}
	if this.IsEndOfMessage {
		if this.IsResponse {
			this.MessageType = MessageTypeEmptyResponse
			return
		}
		this.MessageType = MessageTypeRequestEmptyTermination
		return
	}
	if this.IsResponse {
		this.MessageType = MessageTypeCancelAck
		return
	}
	this.MessageType = MessageTypeCancel
}

func (this *MessageHeader) encodeHeader() {
	headerFields := this.terminationField |
		this.responseField |
		uint32(this.Id)<<this.shiftId |
		uint32(this.Length)<<shiftLength

	this.Encoded.Maximize()
	for i := 0; i < this.HeaderLength; i++ {
		this.Encoded.Data[i] = byte(headerFields)
		headerFields >>= 8
	}

	// fmt.Printf("### MH %p: lshift %v, ishift %v\n", this, this.shiftLength, shiftId)
	// fmt.Printf("### MH %p: Encode header len %v, id %v, resp %v, term %v\n", this, this.Length, this.Id, this.responseField>>shiftResponseBit, this.terminationField)
}

func (this *MessageHeader) setIdAndResponse(id int, isResponse bool) {
	this.Id = id
	this.IsResponse = isResponse
	this.responseField = boolToUint32(isResponse) << shiftResponseBit
}

func (this *MessageHeader) setLengthAndTermination(length int, isEndOfMessage bool) {
	this.Length = length
	this.IsEndOfMessage = isEndOfMessage
	this.terminationField = boolToUint32(isEndOfMessage)
}
