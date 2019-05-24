package streamux

import (
	"fmt"
)

type negotiator_ struct {
	IsNegotiated bool
	HeaderLength int
	LengthBits   int
	IdBits       int

	requestQuickInit int
	allowQuickInit   int
	lengthMinBits    int
	lengthMaxBits    int
	idMinBits        int
	idMaxBits        int
	messageBuffer    []byte
}

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

const initializeMessageLength = 5
const recommendedWildcard = 31
const maxTotalBits = 30

func minInt(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func maxInt(a, b int) int {
	if b > a {
		return b
	}
	return a
}

func midpointInt(a, b int) int {
	min := a
	max := b
	if min > max {
		min = b
		max = a
	}
	diff := max - min
	result := diff/2 + min
	if diff&1 == 1 {
		result++
	}
	return result
}

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

func (this *negotiator_) Initialize(lengthMinBits int, lengthMaxBits int,
	lengthRecommendBits int, idMinBits int, idMaxBits int, idRecommendBits int,
	requestQuickInit bool, allowQuickInit bool) {
	this.lengthMinBits = lengthMinBits
	this.lengthMaxBits = lengthMaxBits
	this.LengthBits = lengthRecommendBits
	this.idMinBits = idMinBits
	this.idMaxBits = idMaxBits
	this.idMaxBits = idRecommendBits
	this.requestQuickInit = 0
	if requestQuickInit {
		this.requestQuickInit = 1
	}
	this.allowQuickInit = 0
	if allowQuickInit {
		this.allowQuickInit = 1
	}
	this.messageBuffer = make([]byte, initializeMessageLength)
}

func (this *negotiator_) negotiateInitializeMessage(data []byte) error {
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
	// TODO
	// themRequestQuickInit := int((message >> shiftQuickInitRequest) & maskRecommended)
	// themAllowQuickInit := int((message >> shiftQuickInitAllowed) & 1)

	idBits, err := negotiateBitCount("ID", this.idMinBits,
		this.idMaxBits, this.IdBits,
		themIdMinBits, themIdMaxBits, themIdBits)
	if err != nil {
		return err
	}

	lengthBits, err := negotiateBitCount("length", this.lengthMinBits,
		this.lengthMaxBits, this.LengthBits, themLengthMinBits,
		themLengthMaxBits, themLengthBits)
	if err != nil {
		return err
	}

	this.IdBits, this.LengthBits = capBitCounts(idBits, lengthBits)
	this.HeaderLength = calculateHeaderLength(this.IdBits, this.LengthBits)

	return nil
}

func (this *negotiator_) BuildInitializeMessage() []byte {
	requestPieces := this.IdBits |
		this.idMaxBits<<shiftIdBitsMax |
		this.idMinBits<<shiftIdBitsMin |
		this.LengthBits<<shiftLengthBitsRecommended |
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

func (this *negotiator_) Negotiate(incomingStreamData []byte) (updatedStreamData []byte, err error) {
	updatedStreamData = incomingStreamData
	if !this.IsNegotiated {
		if len(this.messageBuffer) < initializeMessageLength {
			this.messageBuffer, updatedStreamData = fillBuffer(initializeMessageLength, this.messageBuffer, updatedStreamData)
			if len(this.messageBuffer) < initializeMessageLength {
				return updatedStreamData, nil
			}
		}
		if err := this.negotiateInitializeMessage(this.messageBuffer); err != nil {
			return updatedStreamData, err
		}
		this.IsNegotiated = true
	}
	return updatedStreamData, nil
}
