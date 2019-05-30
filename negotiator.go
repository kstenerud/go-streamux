package streamux

import (
	"fmt"
)

type negotiatorState_ int

const (
	negotiatorStateNotNegotiated = iota
	negotiatorStateFailed
	negotiatorStateQuickNegotiated
	negotiatorStateFullyNegotiated
)

type negotiator_ struct {
	LengthBits int
	IdBits     int

	requestQuickInit int
	allowQuickInit   int
	lengthMinBits    int
	lengthMaxBits    int
	idMinBits        int
	idMaxBits        int
	messageBuffer    []byte
	state            negotiatorState_
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

func negotiateBitCount(name string, usMin int, usMax int, usRecommended int, themMin int, themMax int, themRecommended int) (count int, err error) {
	min := maxInt(usMin, themMin)
	max := minInt(usMax, themMax)
	if max < min {
		return -1, fmt.Errorf("Negotiation failed: max %v (%v) is less than min %v (%v)", name, max, name, min)
	}

	recommended := minInt(usRecommended, themRecommended)
	if recommended == recommendedWildcard {
		recommended = midpointInt(min, max)
	}

	negotiated := minInt(maxInt(recommended, min), max)

	return negotiated, nil
}

func capBitCounts(lengthBits, idBits int) (lengthBitsCapped, idBitsCapped int) {
	idBitsCapped = idBits
	lengthBitsCapped = lengthBits
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
	return lengthBitsCapped, idBitsCapped
}

func validateMinMaxLimits(name string, value int, min int, max int) error {
	if value < min {
		return fmt.Errorf("Negotiation failed: %v (%v) is less than min %v (%v)", name, value, name, min)
	}
	if value > max {
		return fmt.Errorf("Negotiation failed: %v (%v) is greater than max %v (%v)", name, value, name, max)
	}
	return nil
}

func validateMinMaxField(name string, min int, max int) error {
	if min > max {
		return fmt.Errorf("Negotiation failed: min %v (%v) is greater than max %v (%v)", name, min, name, max)
	}
	return nil
}

func validateMinMaxRecommend(name string, min int, max int, recommend int) error {
	if recommend == recommendedWildcard {
		return nil
	}

	if recommend < min {
		return fmt.Errorf("Negotiation failed: recommended %v (%v) is less than min %v (%v)", name, recommend, name, min)
	}
	if recommend > max {
		return fmt.Errorf("Negotiation failed: recommended %v (%v) is greater than max %v (%v)", name, recommend, name, max)
	}
	return nil
}

func validateInitializeFields(lengthMinBits int, lengthMaxBits int,
	lengthRecommendBits int, idMinBits int, idMaxBits int, idRecommendBits int,
	requestQuickInit int, allowQuickInit int) error {

	if requestQuickInit != 0 {
		if allowQuickInit != 0 {
			return fmt.Errorf("Cannot enable both quick init \"request\" and \"allow\" at the same time")
		}
		if lengthRecommendBits == recommendedWildcard {
			return fmt.Errorf("Cannot set recommended length bits to wildard (%v) when requesting quick init", lengthRecommendBits)
		}
		if idRecommendBits == recommendedWildcard {
			return fmt.Errorf("Cannot set recommended ID bits to wildard (%v) when requesting quick init", idRecommendBits)
		}
	}

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

func (this *negotiator_) markNegotiationFailure() {
	this.state = negotiatorStateFailed
}

func (this *negotiator_) markNegotiationSuccess() {
	if this.state != negotiatorStateFailed {
		this.state = negotiatorStateFullyNegotiated
	}
}

func newNegotiator(lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	idMinBits int, idMaxBits int, idRecommendBits int,
	requestQuickInit bool, allowQuickInit bool) *negotiator_ {

	this := new(negotiator_)
	this.Init(lengthMinBits, lengthMaxBits, lengthRecommendBits,
		idMinBits, idMaxBits, idRecommendBits,
		requestQuickInit, allowQuickInit)

	return this
}

func (this *negotiator_) Init(lengthMinBits int, lengthMaxBits int,
	lengthRecommendBits int, idMinBits int, idMaxBits int, idRecommendBits int,
	requestQuickInit bool, allowQuickInit bool) {

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
		this.markNegotiationFailure()
		panic(err)
	}

	this.messageBuffer = make([]byte, 0, initializeMessageLength)

	if requestQuickInit {
		this.state = negotiatorStateQuickNegotiated
	} else {
		this.state = negotiatorStateNotNegotiated
	}
}

func (this *negotiator_) negotiateInitializeMessage() error {
	version := int(this.messageBuffer[0])
	if version != ProtocolVersion {
		return fmt.Errorf("Negotiation failed: Expected protocol version %v, but got %v", ProtocolVersion, version)
	}
	message :=
		uint(this.messageBuffer[1])<<24 |
			uint(this.messageBuffer[2])<<16 |
			uint(this.messageBuffer[3])<<8 |
			uint(this.messageBuffer[4])

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

	if this.requestQuickInit != 0 {
		if themAllowQuickInit == 0 {
			return fmt.Errorf("We requested quick init but peer doesn't allow it")
		}

		// Make sure our recommended values work with their limits
		if err := validateInitializeFields(themLengthMinBits, themLengthMaxBits,
			this.LengthBits, themIdMinBits, themIdMaxBits, this.IdBits,
			themRequestQuickInit, themAllowQuickInit); err != nil {

			return err
		}

		// Note: Header length, length bits, and id bits are already calculated.

	} else if themRequestQuickInit != 0 {
		if this.allowQuickInit == 0 {
			return fmt.Errorf("Peer requested quick init but we don't allow it")
		}

		// Make sure their recommended values work with our limits
		if err := validateInitializeFields(this.lengthMinBits, this.lengthMaxBits,
			themLengthBits, this.idMinBits, this.idMaxBits, themIdBits,
			this.requestQuickInit, this.allowQuickInit); err != nil {

			return err
		}
		this.LengthBits = themLengthBits
		this.IdBits = themIdBits
	} else {
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

		this.LengthBits, this.IdBits = capBitCounts(lengthBits, idBits)

	}

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
		ProtocolVersion,
		byte(requestPieces >> 24),
		byte((requestPieces >> 16) & 0xff),
		byte((requestPieces >> 8) & 0xff),
		byte(requestPieces & 0xff)}
	return request
}

func (this *negotiator_) Feed(incomingStreamData []byte) ([]byte, error) {
	if !this.IsNegotiationComplete() {
		if len(this.messageBuffer) < initializeMessageLength {
			this.messageBuffer, incomingStreamData = fillBuffer(initializeMessageLength, this.messageBuffer, incomingStreamData)
			if len(this.messageBuffer) < initializeMessageLength {
				return incomingStreamData, nil
			}
		}
		if err := this.negotiateInitializeMessage(); err != nil {
			this.markNegotiationFailure()
			return incomingStreamData, err
		}
		this.markNegotiationSuccess()
	}
	return incomingStreamData, nil
}

func (this *negotiator_) CanSendMessages() bool {
	return this.state == negotiatorStateQuickNegotiated || this.state == negotiatorStateFullyNegotiated
}

func (this *negotiator_) CanReceiveMessages() bool {
	return this.state == negotiatorStateFullyNegotiated
}

func (this *negotiator_) IsNegotiationComplete() bool {
	return this.state == negotiatorStateFullyNegotiated || this.state == negotiatorStateFailed
}

func (this *negotiator_) ExplainFailure() string {
	if this.state == negotiatorStateFailed {
		return "Negotiation failed"
	}
	if this.state == negotiatorStateNotNegotiated {
		return "Negotiation not complete"
	}
	return "Unknown (this is a bug)"
}
