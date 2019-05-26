package streamux

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

func (this *messageDecoder_) decodeHeader(incomingStreamData []byte) []byte {
	if len(this.headerBuffer) < this.headerLength {
		this.headerBuffer, incomingStreamData = fillBuffer(this.headerLength, this.headerBuffer, incomingStreamData)
	}

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
	}

	return incomingStreamData
}

func (this *messageDecoder_) Initialize(headerLength int, lengthBits int, idBits int) {
	this.headerLength = headerLength
	this.maskId = (1 << uint32(idBits)) - 1
	this.shiftLength = shiftId + uint(idBits)
	this.maskLength = (1 << uint32(lengthBits)) - 1
	this.headerBuffer = make([]byte, 0, headerLength)
}

func (this *messageDecoder_) Feed(incomingStreamData []byte) error {
	incomingStreamData = this.decodeHeader(incomingStreamData)

	var decodedData []byte
	var isChunkComplete bool
	for len(incomingStreamData) > 0 && !isChunkComplete {
		decodedData, incomingStreamData = useBytes(this.bytesRemaining, incomingStreamData)
		this.bytesRemaining -= len(decodedData)
		isChunkComplete = this.bytesRemaining == 0
		if this.isResponse {
			this.callbacks.OnResponseChunkReceived(this.id, this.isEndOfMessage, decodedData)
		} else {
			this.callbacks.OnRequestChunkReceived(this.id, this.isEndOfMessage, decodedData)
		}
		if isChunkComplete {
			this.headerBuffer = this.headerBuffer[:0]
		}
	}
	return nil
}
