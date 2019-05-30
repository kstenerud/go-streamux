package streamux

const (
	shiftResponseBit = 1
	shiftId          = 2
)

func boolToUint32(value bool) uint32 {
	if value {
		return 1
	}
	return 0
}

func calculateHeaderLength(lengthBits, idBits int) int {
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

type messageHeader_ struct {
	// Public data (read only)
	Id             int
	Length         int
	IsResponse     bool
	IsEndOfMessage bool
	MessageType    messageType
	Encoded        []byte
	HeaderLength   int
	MaxChunkLength int

	// Session constants
	maskId      uint32
	maskLength  uint32
	shiftLength uint

	// Internal
	terminationField uint32
	responseField    uint32
	idField          uint32
}

type messageType int

const (
	messageTypeNormal messageType = iota
	messageTypeCancel
	messageTypeCancelAck
	messageTypePing
	messageTypeEmptyResponse
)

func newMessageHeader(lengthBits int, idBits int) *messageHeader_ {
	this := new(messageHeader_)
	this.Init(lengthBits, idBits)
	return this
}

func (this *messageHeader_) Init(lengthBits int, idBits int) {
	this.HeaderLength = calculateHeaderLength(lengthBits, idBits)
	this.maskId = 1<<uint(idBits) - 1
	this.shiftLength = shiftId + uint(idBits)
	this.maskLength = 1<<uint(lengthBits) - 1
	this.MaxChunkLength = 1<<uint(lengthBits) - 1
	this.Encoded = make([]byte, this.HeaderLength)
}

func (this *messageHeader_) setMessageType() {
	if this.Length > 0 {
		this.MessageType = messageTypeNormal
	}
	if this.IsEndOfMessage {
		if this.IsResponse {
			this.MessageType = messageTypeEmptyResponse
			return
		}
		this.MessageType = messageTypePing
		return
	}
	if this.IsResponse {
		this.MessageType = messageTypeCancelAck
		return
	}
	this.MessageType = messageTypeCancel
}

func (this *messageHeader_) encodeHeader() {
	headerFields := this.terminationField |
		this.responseField |
		this.idField |
		uint32(this.Length)<<this.shiftLength

	for i := 0; i < this.HeaderLength; i++ {
		this.Encoded[i] = byte(headerFields)
		headerFields >>= 8
	}
	// fmt.Printf("### MH %p: lshift %v, ishift %v\n", this, this.shiftLength, shiftId)
	// fmt.Printf("### MH %p: Encode header len %v, id %v, resp %v, term %v\n", this, this.Length, this.idField>>shiftId, this.responseField>>shiftResponseBit, this.terminationField)
}

func (this *messageHeader_) setIdAndResponse(id int, isResponse bool) {
	this.Id = id
	this.idField = uint32(id) << shiftId
	this.IsResponse = isResponse
	this.responseField = boolToUint32(isResponse) << shiftResponseBit
}

func (this *messageHeader_) setLengthAndTermination(length int, isEndOfMessage bool) {
	this.Length = length
	this.IsEndOfMessage = isEndOfMessage
	this.terminationField = boolToUint32(isEndOfMessage)
}

func (this *messageHeader_) SetAll(id int, length int, isResponse bool, isEndOfMessage bool) {
	this.setIdAndResponse(id, isResponse)
	this.setLengthAndTermination(length, isEndOfMessage)
	this.encodeHeader()
	this.setMessageType()
}

func (this *messageHeader_) SetIdAndResponseNoEncode(id int, isResponse bool) {
	this.setIdAndResponse(id, isResponse)
	this.setMessageType()
}

func (this *messageHeader_) SetLengthAndTermination(length int, isEndOfMessage bool) {
	this.setLengthAndTermination(length, isEndOfMessage)
	this.encodeHeader()
	this.setMessageType()
}

func (this *messageHeader_) SetLength(length int) {
	this.Length = length
	this.encodeHeader()
}

func (this *messageHeader_) IsDecoded() bool {
	return len(this.Encoded) == this.HeaderLength
}

func (this *messageHeader_) ClearEncoded() {
	this.Encoded = this.Encoded[:0]
}

func (this *messageHeader_) Feed(incomingStreamData []byte) (remainingData []byte) {
	remainingData = incomingStreamData
	// fmt.Printf("### MH %p: Feed: headerLength %v, headerBuffer %v, incoming %v\n", this, this.HeaderLength, len(this.Encoded), len(remainingData))

	if this.IsDecoded() {
		this.ClearEncoded()
	}

	this.Encoded, remainingData = fillBuffer(this.HeaderLength, this.Encoded, remainingData)

	if this.IsDecoded() {
		var headerFields uint32
		for i := this.HeaderLength - 1; i >= 0; i-- {
			headerFields <<= 8
			headerFields |= uint32(this.Encoded[i])
		}
		this.IsEndOfMessage = (headerFields & 1) == 1
		this.IsResponse = ((headerFields >> shiftResponseBit) & 1) == 1
		this.Id = int((headerFields >> shiftId) & this.maskId)
		this.Length = int((headerFields >> this.shiftLength) & this.maskLength)
		this.setMessageType()
		// fmt.Printf("### MH %p: lshift %v, lmask %v, ishift %v, imask %v\n", this, this.shiftLength, this.maskLength, shiftId, this.maskId)
		// fmt.Printf("### MH %p: Decode header %08x: len %v, id %v, resp %v, term %v\n", this, headerFields, this.Length, this.Id, this.IsResponse, this.IsEndOfMessage)
	}

	return remainingData
}
