package streamux

import (
	"fmt"
)

const initializeMessageLength = 5
const recommendedWildcard = 31
const maxTotalBits = 30

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

type ProtocolCallbacks interface {
	OnMessageChunkReceived(messageId int, isEnd bool, data []byte)
	OnDataReadyToSend(data []byte)
}

type negotiationStruct struct {
	requestQuickInit int
	allowQuickInit   int
	lengthMinBits    int
	lengthMaxBits    int
	idMinBits        int
	idMaxBits        int
	lengthBits       int
	idBits           int
}

type Protocol struct {
	headerLength int
	negotiation  negotiationStruct
	idPool       *IdPool
	callbacks    ProtocolCallbacks
	isInitilized bool
}

func (this *Protocol) sendMessageChunk(message []byte) {
	this.callbacks.OnDataReadyToSend(message)
}

func (this *Protocol) disposeId(id int) {
	this.idPool.CheckinId(id)
}

func (this *Protocol) buildInitializeMessage() []byte {
	requestPieces := this.negotiation.idBits |
		this.negotiation.idMaxBits<<shiftIdBitsMax |
		this.negotiation.idMinBits<<shiftIdBitsMin |
		this.negotiation.lengthBits<<shiftLengthBitsRecommended |
		this.negotiation.lengthMaxBits<<shiftLengthBitsMax |
		this.negotiation.lengthMinBits<<shiftLengthBitsMin |
		this.negotiation.requestQuickInit<<shiftQuickInitRequest |
		this.negotiation.allowQuickInit<<shiftQuickInitAllowed

	request := []byte{
		protocolVersion,
		byte(requestPieces >> 24),
		byte((requestPieces >> 16) & 0xff),
		byte((requestPieces >> 8) & 0xff),
		byte(requestPieces & 0xff)}
	return request
}

func negotiateBitCount(name string, usMin int, usMax int, usRecommended int, themMin int, themMax int, themRecommended int) (count int, err error) {
	min := maxInt(usMin, themMin)
	max := minInt(usMax, themMax)
	if max < min {
		return -1, fmt.Errorf("Negotiation failed: %v max (%v) is less than min (%v)", name, max, min)
	}

	recommended := minInt(usRecommended, themRecommended)
	if recommended == recommendedWildcard {
		recommended = midpointInt(min, max)
	}

	negotiated := minInt(maxInt(recommended, min), max)

	return negotiated, nil
}

func capBitCounts(idBits, lengthBits int) (idBitsCapped, lengthBitsCapped int) {
	if lengthBits+idBits > maxTotalBits {
		midpoint := maxTotalBits / 2
		if lengthBits >= midpoint {
			if idBits >= midpoint {
				idBitsCapped = midpoint
				lengthBitsCapped = midpoint
			} else {
				lengthBitsCapped = maxTotalBits - idBits
			}
		} else {
			idBitsCapped = maxTotalBits - lengthBits
		}
	}
	return idBitsCapped, lengthBitsCapped
}

func (this *Protocol) negotiateInitializeHeader(data []byte) error {
	if len(data) < initializeMessageLength {
		return fmt.Errorf("Expected %v bytes for initialize message, but got %v", initializeMessageLength, len(data))
	}
	version := int(data[0])
	if version != protocolVersion {
		return fmt.Errorf("Negotiation failed: Expected protocol version %v, but got %v", protocolVersion, version)
	}
	message := uint(data[1]) |
		uint(data[2])<<8 |
		uint(data[3])<<16 |
		uint(data[4])<<24

	themIdBits := int(message & maskRecommended)
	themIdMaxBits := int((message >> shiftIdBitsMax) & maskMax)
	themIdMinBits := int((message >> shiftIdBitsMin) & maskMin)
	themLengthBits := int((message >> shiftLengthBitsRecommended) & maskRecommended)
	themLengthMaxBits := int((message >> shiftLengthBitsMax) & maskMax)
	themLengthMinBits := int((message >> shiftLengthBitsMin) & maskMin)
	// themRequestQuickInit := int((message >> shiftQuickInitRequest) & maskRecommended)
	// themAllowQuickInit := int((message >> shiftQuickInitAllowed) & 1)

	idBits, err := negotiateBitCount("ID", this.negotiation.idMinBits,
		this.negotiation.idMaxBits, this.negotiation.idBits,
		themIdMinBits, themIdMaxBits, themIdBits)
	if err != nil {
		return err
	}

	lengthBits, err := negotiateBitCount("length", this.negotiation.lengthMinBits,
		this.negotiation.lengthMaxBits, this.negotiation.lengthBits, themLengthMinBits,
		themLengthMaxBits, themLengthBits)
	if err != nil {
		return err
	}

	this.negotiation.idBits, this.negotiation.lengthBits = capBitCounts(idBits, lengthBits)

	this.headerLength = calculateHeaderLength(this.negotiation.idBits, this.negotiation.lengthBits)

	return nil
}

func (this *Protocol) getHeaderLength() int {
	return this.headerLength
}

func (this *Protocol) nextFreeID() int {
	return this.idPool.CheckoutNextId()
}

func NewProtocol(lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	idMinBits int, idMaxBits int, idRecommendBits int, callbacks ProtocolCallbacks) *Protocol {
	this := new(Protocol)
	this.negotiation.lengthMinBits = lengthMinBits
	this.negotiation.lengthMaxBits = lengthMaxBits
	this.negotiation.lengthBits = lengthRecommendBits
	this.negotiation.idMinBits = idMinBits
	this.negotiation.idMaxBits = idMaxBits
	this.negotiation.idMaxBits = idRecommendBits
	this.negotiation.requestQuickInit = 0 // TODO
	this.negotiation.allowQuickInit = 0   // TODO
	this.callbacks = callbacks

	this.sendMessageChunk(this.buildInitializeMessage())

	return this
}

func (this *Protocol) BeginMessage() *SendableMessage {
	isResponse := false
	return newSendableMessage(this, this.nextFreeID(), this.negotiation.idBits,
		this.negotiation.lengthBits, isResponse)
}

func (this *Protocol) BeginResponseMessage(responseToId int) *SendableMessage {
	isResponse := true
	return newSendableMessage(this, this.nextFreeID(), this.negotiation.idBits,
		this.negotiation.lengthBits, isResponse)
}

func (this *Protocol) Feed(data []byte) error {
	if !this.isInitilized {
		if err := this.negotiateInitializeHeader((data)); err != nil {
			return err
		}
		this.isInitilized = true
	}

	// TODO

	return nil
}
