package streamux

type codecState int

const (
	codecStateUninitialized = iota
	codecStateSentInitializeMessage
	codecStateInitialized
	codecStateFailed
)

type Codec struct {
	requestQuickInit bool
	allowQuickInit   bool
	lengthMinBits    int
	lengthMaxBits    int
	idMinBits        int
	idMaxBits        int
	lengthBits       int
	idBits           int
	state            codecState
}

const protocolVersion = 1

func NewCodec(lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	idMinBits int, idMaxBits int, idRecommendBits int) *Codec {
	this := new(Codec)
	this.lengthMinBits = lengthMinBits
	this.lengthMaxBits = lengthMaxBits
	this.lengthBits = lengthRecommendBits
	this.idMinBits = idMinBits
	this.idMaxBits = idMaxBits
	this.idMaxBits = idRecommendBits
	// TODO: Quick Init
	return this
}

func (this *Codec) buildInitializeMessage() []byte {
	var requestPieces uint32 = 0 // TODO
	request := []byte{
		protocolVersion,
		byte(requestPieces >> 24),
		byte((requestPieces >> 16) & 0xff),
		byte((requestPieces >> 8) & 0xff),
		byte(requestPieces & 0xff)}
	return request
}

func (this *Codec) NextMessage() []byte {
	// TODO: block or not?
	switch this.state {
	case codecStateUninitialized:
		this.state = codecStateSentInitializeMessage
		return this.buildInitializeMessage()
	default:
		// TODO
		return nil
	}
}

func (this *Codec) onInitiatorRequest(request []byte) {
	// TODO
}
