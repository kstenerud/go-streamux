package streamux

type messageDecoder_ struct {
	headerLength   int
	idBits         int
	lengthBits     int
	headerBuffer   []byte
	isResponse     bool
	isEndOfMessage bool
	id             int
	length         int
	bytesRemaining int
	callbacks      MessageReceiveCallbacks
}

func (this *messageDecoder_) decodeHeader() {
	if len(this.headerBuffer) == this.headerLength {
		var header uint32
		for i := 0; i < this.headerLength; i++ {
			header <<= 8
			header |= uint32(this.headerBuffer[i])
		}
		// TODO
	}
}

func (this *messageDecoder_) Initialize(headerLength int, idBits int, lengthBits int) {
	this.headerLength = headerLength
	this.idBits = idBits
	this.lengthBits = lengthBits
	this.headerBuffer = make([]byte, headerLength)
}

func (this *messageDecoder_) Feed(incomingStreamData []byte) error {
	var decodedData []byte
	var isChunkComplete bool
	for len(incomingStreamData) > 0 && !isChunkComplete {
		if len(this.headerBuffer) < this.headerLength {
			this.headerBuffer, incomingStreamData = fillBuffer(this.headerLength, this.headerBuffer, incomingStreamData)
			this.decodeHeader()
			continue
		}

		decodedData, incomingStreamData = useBytes(this.bytesRemaining, incomingStreamData)
		this.bytesRemaining -= len(decodedData)
		isChunkComplete = this.bytesRemaining == 0
		if this.isResponse {
			this.callbacks.OnResponseChunkReceived(this.id, this.isEndOfMessage, decodedData)
		} else {
			this.callbacks.OnRequestChunkReceived(this.id, this.isEndOfMessage, decodedData)
		}
	}
	return nil
}
