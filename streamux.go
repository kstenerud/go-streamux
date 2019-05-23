package streamux

type ProtocolState int

const (
	ProtocolStateUninitialized = iota
	ProtocolStateSentInitializeMessage
	ProtocolStateInitialized
	ProtocolStateFailed
)

func calculateHeaderLength(idBits, lengthBits int) int {
	totalBits := idBits + lengthBits
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

type SendableMessage struct {
	protocol           *Protocol
	id                 uint32
	idBits             int
	lengthBits         int
	headerLength       int
	maxChunkLength     int
	responseBitShifted uint32
	lengthBitShift     uint
	data               []byte
	closed             bool
}

func (this *SendableMessage) Close() {
	if this.closed {
		return
	}
	this.protocol.freeId(this.id)
	this.closed = true
}

func (this *SendableMessage) dataLength() int {
	return len(this.data) - this.headerLength
}

func (this *SendableMessage) sendCurrentChunk(terminationBit uint32) {
	length := uint32(this.dataLength())
	headerFields := terminationBit |
		this.responseBitShifted |
		this.id<<idBitShift |
		length<<this.lengthBitShift

	for i := 0; i < this.headerLength; i++ {
		this.data[i] = byte(headerFields)
		headerFields >>= 8
	}

	this.protocol.sendMessage(this.data)
}

const (
	responseBitShift = 1
	idBitShift       = 2
)

func boolToUint32(value bool) uint32 {
	if value {
		return 1
	}
	return 0
}

func (this *SendableMessage) AddData(data []byte) {
	for len(data) > 0 {
		currentLength := this.dataLength()
		lengthToAdd := len(data)
		if currentLength+lengthToAdd > this.maxChunkLength {
			lengthToAdd = this.maxChunkLength - currentLength
		}
		this.data = append(this.data, data[:lengthToAdd]...)
		currentLength = this.dataLength()
		if currentLength == this.maxChunkLength {
			this.sendCurrentChunk(0)
			data = data[lengthToAdd:]
		}
	}
}

func (this *SendableMessage) Complete() {
	this.sendCurrentChunk(1)
	this.Close()
}

func newSendableMessage(protocol *Protocol, id int, idBits int, lengthBits int, isResponse bool) *SendableMessage {
	message := new(SendableMessage)
	message.protocol = protocol
	message.id = uint32(id)
	message.idBits = idBits
	message.lengthBits = lengthBits
	message.headerLength = calculateHeaderLength(idBits, lengthBits)
	message.maxChunkLength = 1<<uint(lengthBits) - 1
	message.data = make([]byte, message.headerLength, message.maxChunkLength)
	message.responseBitShifted = boolToUint32(isResponse) << responseBitShift
	message.lengthBitShift = uint(idBitShift + idBits)

	return message
}

type ProtocolCallbacks interface {
	OnMessageChunkReceived(messageId int, isEnd bool, data []byte)
	OnDataReadyToSend(data []byte)
}

type Protocol struct {
	expectingInitMessage bool
	requestQuickInit     int
	allowQuickInit       int
	lengthMinBits        int
	lengthMaxBits        int
	idMinBits            int
	idMaxBits            int
	lengthBits           int
	idBits               int
	state                ProtocolState
	callbacks            ProtocolCallbacks
}

const protocolVersion = 1

const (
	shiftQuickInitRequest      = 30
	shiftQuickInitAllowed      = 29
	shiftLengthBitsMin         = 24
	shiftLengthBitsMax         = 19
	shiftLengthBitsRecommended = 14
	shiftIdBitsMin             = 10
	shiftIdBitsMax             = 5
)

const (
	maskMin         = 0x0f
	maskMax         = 0x1f
	maskRecommended = 0x1f
)

func (this *Protocol) sendMessage(message []byte) {
	this.callbacks.OnDataReadyToSend(message)
}

func (this *Protocol) freeId(id uint32) {
	// TODO
}

func NewProtocol(lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	idMinBits int, idMaxBits int, idRecommendBits int, callbacks ProtocolCallbacks) *Protocol {
	this := new(Protocol)
	this.lengthMinBits = lengthMinBits
	this.lengthMaxBits = lengthMaxBits
	this.lengthBits = lengthRecommendBits
	this.idMinBits = idMinBits
	this.idMaxBits = idMaxBits
	this.idMaxBits = idRecommendBits
	this.callbacks = callbacks

	this.requestQuickInit = 0 // TODO
	this.allowQuickInit = 0   // TODO

	this.sendMessage(this.buildInitializeMessage())

	return this
}

func (this *Protocol) buildInitializeMessage() []byte {
	requestPieces := this.idBits |
		this.idMaxBits<<shiftIdBitsMax |
		this.idMinBits<<shiftIdBitsMin |
		this.lengthBits<<shiftLengthBitsRecommended |
		this.lengthMaxBits<<shiftLengthBitsMax |
		this.lengthMinBits<<shiftLengthBitsMin |
		this.requestQuickInit<<shiftQuickInitRequest |
		this.allowQuickInit<<shiftQuickInitAllowed

	request := []byte{
		protocolVersion,
		byte(requestPieces >> 24),
		byte((requestPieces >> 16) & 0xff),
		byte((requestPieces >> 8) & 0xff),
		byte(requestPieces & 0xff)}
	return request
}

func (this *Protocol) nextFreeID() uint32 {
	return 0
}

func (this *Protocol) BeginMessage() *SendableMessage {
	// TODO
	return nil
}

func (this *Protocol) BegiReplyMessage(responseToId int) *SendableMessage {
	// TODO
	return nil
}

func (this *Protocol) Feed(data []byte) {
	// TODO
}
