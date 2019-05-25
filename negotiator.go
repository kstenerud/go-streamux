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
	shiftQuickInitRequest      = 29
	shiftQuickInitAllowed      = 28
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

func validateMinMaxLimits(name string, value int, min int, max int) error {
	if value < min {
		return fmt.Errorf("%v has value %v, which is less than the minimum of %v", name, value, min)
	}
	if value > max {
		return fmt.Errorf("%v has value %v, which is greater than the minimum of %v", name, value, min)
	}
	return nil
}

func validateMinMaxField(name string, min int, max int) error {
	if min > max {
		return fmt.Errorf("%v min value (%v) is greater than max value (%v)", name, min, max)
	}
	return nil
}

func validateMinMaxRecommend(name string, min int, max int, recommend int) error {
	if recommend == recommendedWildcard {
		return nil
	}

	if recommend < min {
		return fmt.Errorf("recommended %v (%v) is less than max %v (%v)", name, recommend, name, max)
	}
	if recommend > max {
		return fmt.Errorf("recommended %v (%v) is greater than min %v (%v)", name, recommend, name, min)
	}
	return nil
}

func validateInitializeFields(lengthMinBits int, lengthMaxBits int,
	lengthRecommendBits int, idMinBits int, idMaxBits int, idRecommendBits int,
	requestQuickInit int, allowQuickInit int) error {

	if err := validateMinMaxLimits("min length", lengthMinBits, 1, 15); err != nil {
		return err
	}

	if err := validateMinMaxLimits("max length", lengthMaxBits, 1, 30); err != nil {
		return err
	}

	if err := validateMinMaxField("length", lengthMinBits, lengthMaxBits); err != nil {
		return err
	}

	if err := validateMinMaxLimits("recommended length", lengthRecommendBits, 1, 31); err != nil {
		return err
	}

	if err := validateMinMaxRecommend("length", lengthMinBits, lengthMaxBits, lengthRecommendBits); err != nil {
		return err
	}

	if err := validateMinMaxLimits("min ID", idMinBits, 0, 15); err != nil {
		return err
	}

	if err := validateMinMaxLimits("max ID", idMaxBits, 0, 29); err != nil {
		return err
	}

	if err := validateMinMaxField("ID", idMinBits, idMaxBits); err != nil {
		return err
	}

	if err := validateMinMaxLimits("recommended ID", idRecommendBits, 0, 31); err != nil {
		return err
	}

	if err := validateMinMaxRecommend("ID", idMinBits, idMaxBits, idRecommendBits); err != nil {
		return err
	}

	return nil
}

func (this *negotiator_) Initialize(lengthMinBits int, lengthMaxBits int,
	lengthRecommendBits int, idMinBits int, idMaxBits int, idRecommendBits int,
	requestQuickInit bool, allowQuickInit bool) error {

	this.lengthMinBits = lengthMinBits
	this.lengthMaxBits = lengthMaxBits
	this.LengthBits = lengthRecommendBits
	this.idMinBits = idMinBits
	this.idMaxBits = idMaxBits
	this.IdBits = idRecommendBits
	this.requestQuickInit = 0
	if requestQuickInit {
		this.requestQuickInit = 1
	}
	this.allowQuickInit = 0
	if allowQuickInit {
		this.allowQuickInit = 1
	}

	if err := validateInitializeFields(this.lengthMinBits, this.lengthMaxBits, this.LengthBits,
		this.idMinBits, this.idMaxBits, this.IdBits, this.requestQuickInit, this.allowQuickInit); err != nil {
		return err
	}

	this.messageBuffer = make([]byte, 0, initializeMessageLength)
	return nil
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
	themRequestQuickInit := int((message >> shiftQuickInitRequest) & 1)
	themAllowQuickInit := int((message >> shiftQuickInitAllowed) & 1)

	if err := validateInitializeFields(themLengthMinBits, themLengthMaxBits,
		themLengthBits, themIdMinBits, themIdMaxBits, themIdBits,
		themRequestQuickInit, themAllowQuickInit); err != nil {

		return err
	}

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

func (this *negotiator_) Feed(incomingStreamData []byte) (updatedStreamData []byte, err error) {
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
