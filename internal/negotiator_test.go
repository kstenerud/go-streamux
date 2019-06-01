package internal

import (
	"fmt"
	"testing"

	"github.com/kstenerud/go-streamux/test"
)

func buildInitMsg(version, minLen, maxLen, recLen, minId, maxId, recId int, qiReq, qiAllowed bool) []byte {

	var qiReqValue, qiAllowedValue uint32
	if qiReq {
		qiReqValue = 1
	}
	if qiAllowed {
		qiAllowedValue = 1
	}

	if version < 0 || version > 255 ||
		minLen < 0 || minLen > 15 || maxLen < 0 || maxLen > 31 || recLen < 0 || recLen > 31 ||
		minId < 0 || minId > 15 || maxId < 0 || maxId > 31 || recId < 0 || recId > 31 {
		panic("Values out of range")
	}

	var fields uint32 = qiReqValue<<29 |
		qiAllowedValue<<28 |
		uint32(minLen<<24) |
		uint32(maxLen<<19) |
		uint32(recLen<<14) |
		uint32(minId<<10) |
		uint32(maxId<<5) |
		uint32(recId)

	msg := make([]byte, 5)
	msg[0] = byte(version)
	msg[1] = byte((fields >> 24) & 0xff)
	msg[2] = byte((fields >> 16) & 0xff)
	msg[3] = byte((fields >> 8) & 0xff)
	msg[4] = byte(fields & 0xff)

	return msg
}

func assertNegotiationInitSuccess(t *testing.T,
	lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	idMinBits int, idMaxBits int, idRecommendBits int,
	requestQuickInit bool, allowQuickInit bool) {

	test.AssertDoesNotPanic(t, func() {
		NewNegotiator(1,
			lengthMinBits, lengthMaxBits, lengthRecommendBits,
			idMinBits, idMaxBits, idRecommendBits,
			requestQuickInit, allowQuickInit)
	})
}

func assertNegotiationInitFail(t *testing.T,
	lengthMinBits int, lengthMaxBits int, lengthRecommendBits int,
	idMinBits int, idMaxBits int, idRecommendBits int,
	requestQuickInit bool, allowQuickInit bool) {

	test.AssertDoesPanic(t, func() {
		NewNegotiator(1,
			lengthMinBits, lengthMaxBits, lengthRecommendBits,
			idMinBits, idMaxBits, idRecommendBits,
			requestQuickInit, allowQuickInit)
	})
}

func performNegotiation(t *testing.T,
	usLengthMinBits int, usLengthMaxBits int, usLengthRecommendBits int,
	usIdMinBits int, usIdMaxBits int, usIdRecommendBits int,
	usRequestQuickInit bool, usAllowQuickInit bool, themVersion int,
	themLengthMinBits int, themLengthMaxBits int, themLengthRecommendBits int,
	themIdMinBits int, themIdMaxBits int, themIdRecommendBits int,
	themRequestQuickInit bool, themAllowQuickInit bool) (lengthBits int, idBits int, err error) {

	var negotiator ProtocolNegotiator
	test.AssertDoesNotPanic(t, func() {
		negotiator.Init(1,
			usLengthMinBits, usLengthMaxBits, usLengthRecommendBits,
			usIdMinBits, usIdMaxBits, usIdRecommendBits,
			usRequestQuickInit, usAllowQuickInit)
	})

	message := buildInitMsg(themVersion,
		themLengthMinBits, themLengthMaxBits, themLengthRecommendBits,
		themIdMinBits, themIdMaxBits, themIdRecommendBits,
		themRequestQuickInit, themAllowQuickInit)

	messageAfter, err := negotiator.Feed(message)
	if len(messageAfter) != 0 {
		return 0, 0, fmt.Errorf("Expected all %v bytes to be used, but %v bytes remain", len(message), len(messageAfter))
	}

	return negotiator.LengthBits, negotiator.IdBits, err
}

func assertNegotiationSuccess(t *testing.T,
	usLengthMinBits int, usLengthMaxBits int, usLengthRecommendBits int,
	usIdMinBits int, usIdMaxBits int, usIdRecommendBits int,
	usRequestQuickInit bool, usAllowQuickInit bool,
	themVersion int,
	themLengthMinBits int, themLengthMaxBits int, themLengthRecommendBits int,
	themIdMinBits int, themIdMaxBits int, themIdRecommendBits int,
	themRequestQuickInit bool, themAllowQuickInit bool) {

	if _, _, err := performNegotiation(t,
		usLengthMinBits, usLengthMaxBits, usLengthRecommendBits,
		usIdMinBits, usIdMaxBits, usIdRecommendBits,
		usRequestQuickInit, usAllowQuickInit, themVersion,
		themLengthMinBits, themLengthMaxBits, themLengthRecommendBits,
		themIdMinBits, themIdMaxBits, themIdRecommendBits,
		themRequestQuickInit, themAllowQuickInit); err != nil {

		t.Error(err)
	}
}

func assertNegotiationFail(t *testing.T,
	usLengthMinBits int, usLengthMaxBits int, usLengthRecommendBits int,
	usIdMinBits int, usIdMaxBits int, usIdRecommendBits int,
	usRequestQuickInit bool, usAllowQuickInit bool, themVersion int,
	themLengthMinBits int, themLengthMaxBits int, themLengthRecommendBits int,
	themIdMinBits int, themIdMaxBits int, themIdRecommendBits int,
	themRequestQuickInit bool, themAllowQuickInit bool) {

	if _, _, err := performNegotiation(t,
		usLengthMinBits, usLengthMaxBits, usLengthRecommendBits,
		usIdMinBits, usIdMaxBits, usIdRecommendBits,
		usRequestQuickInit, usAllowQuickInit, themVersion,
		themLengthMinBits, themLengthMaxBits, themLengthRecommendBits,
		themIdMinBits, themIdMaxBits, themIdRecommendBits,
		themRequestQuickInit, themAllowQuickInit); err == nil {

		t.Errorf("Negotiation should have failed but didn't")
	}
}

func assertNegotiation(t *testing.T,
	usLengthMinBits int, usLengthMaxBits int, usLengthRecommendBits int,
	usIdMinBits int, usIdMaxBits int, usIdRecommendBits int,
	usRequestQuickInit bool, usAllowQuickInit bool, themVersion int,
	themLengthMinBits int, themLengthMaxBits int, themLengthRecommendBits int,
	themIdMinBits int, themIdMaxBits int, themIdRecommendBits int,
	themRequestQuickInit bool, themAllowQuickInit bool,
	expectLengthBits int, expectIdBits int) {

	lengthBits, idBits, err := performNegotiation(t,
		usLengthMinBits, usLengthMaxBits, usLengthRecommendBits,
		usIdMinBits, usIdMaxBits, usIdRecommendBits,
		usRequestQuickInit, usAllowQuickInit, themVersion,
		themLengthMinBits, themLengthMaxBits, themLengthRecommendBits,
		themIdMinBits, themIdMaxBits, themIdRecommendBits,
		themRequestQuickInit, themAllowQuickInit)

	if err != nil {
		t.Error(err)
		return
	}

	if lengthBits != expectLengthBits {
		t.Errorf("Expected Length bits (min %v, max %v, rec %v) (min %v, max %v, rec %v) to be %v, but got %v",
			usLengthMinBits, usLengthMaxBits, usLengthRecommendBits,
			themLengthMinBits, themLengthMaxBits, themLengthRecommendBits,
			expectLengthBits, lengthBits)
	}

	if idBits != expectIdBits {
		t.Errorf("Expected ID bits (min %v, max %v, rec %v) (min %v, max %v, rec %v) to be %v, but got %v",
			usIdMinBits, usIdMaxBits, usIdRecommendBits,
			themIdMinBits, themIdMaxBits, themIdRecommendBits,
			expectIdBits, idBits)
	}
}

// =============================================================================

// General (init)

func TestInitValid(t *testing.T) {
	assertNegotiationInitSuccess(t, 1, 1, 1, 0, 0, 0, false, false)
	assertNegotiationInitSuccess(t, 15, 30, 30, 15, 29, 29, false, false)
}

func TestInitWildcard(t *testing.T) {
	assertNegotiationInitSuccess(t, 1, 1, 1, 0, 0, 31, false, false)
	assertNegotiationInitSuccess(t, 1, 1, 31, 0, 0, 0, false, false)
	assertNegotiationInitSuccess(t, 1, 1, 31, 0, 0, 31, false, false)
}

// Length (init)

func TestInitMinLengthTooLow(t *testing.T) {
	assertNegotiationInitFail(t, 0, 1, 1, 0, 0, 0, false, false)
}

func TestInitMinLengthTooHigh(t *testing.T) {
	assertNegotiationInitFail(t, 16, 16, 1, 0, 0, 0, false, false)
}

func TestInitMaxLengthTooLow(t *testing.T) {
	assertNegotiationInitFail(t, 1, 0, 1, 0, 0, 0, false, false)
}

func TestInitMaxLengthTooHigh(t *testing.T) {
	assertNegotiationInitFail(t, 1, 31, 1, 0, 0, 0, false, false)
}

func TestInitRecLengthTooLow(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 0, 0, 0, 0, false, false)
}

func TestInitRecLengthTooHigh(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 32, 0, 0, 0, false, false)
}

func TestInitLengthMinGtMax(t *testing.T) {
	assertNegotiationInitFail(t, 2, 1, 2, 0, 0, 0, false, false)
}

func TestInitLengthRecGtMax(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 10, 0, 0, 0, false, false)
}

func TestInitLengthRecLtMin(t *testing.T) {
	assertNegotiationInitFail(t, 2, 2, 1, 0, 0, 0, false, false)
}

// ID (init)

func TestInitMinIdTooLow(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, -1, 0, 0, false, false)
}

func TestInitMinIdTooHigh(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, 16, 0, 0, false, false)
}

func TestInitMaxIdTooLow(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, 0, -1, 0, false, false)
}

func TestInitMaxIdTooHigh(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, 0, 30, 0, false, false)
}

func TestInitRecIdTooLow(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, 0, 0, -1, false, false)
}

func TestInitRecIdTooHigh(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, 0, 0, 32, false, false)
}

func TestInitIdMinGtMax(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, 1, 0, 1, false, false)
}

func TestInitIdRecGtMax(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, 0, 1, 2, false, false)
}

func TestInitIdRecLtMin(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, 1, 2, 0, false, false)
}

// General (full)

func TestFullValid(t *testing.T) {
	assertNegotiationSuccess(t, 1, 1, 1, 0, 0, 0, false, false, 1, 1, 1, 1, 0, 0, 0, false, false)
	assertNegotiationSuccess(t, 1, 30, 15, 0, 29, 15, false, false, 1, 1, 30, 15, 0, 29, 15, false, false)
}

func TestFullVersion(t *testing.T) {
	assertNegotiationFail(t, 1, 30, 15, 0, 29, 15, false, false, 0, 1, 30, 15, 0, 29, 15, false, false)
	assertNegotiationFail(t, 1, 30, 15, 0, 29, 15, false, false, 50, 1, 30, 15, 0, 29, 15, false, false)
}

func TestNegotiationFullWildcard(t *testing.T) {
	assertNegotiation(t, 6, 20, 10, 9, 16, 11, false, false, 1, 8, 18, 18, 6, 18, 8, false, false, 10, 9)
	assertNegotiation(t, 6, 20, 10, 9, 16, 11, false, false, 1, 8, 18, 18, 6, 18, 31, false, false, 10, 11)
	assertNegotiation(t, 6, 20, 10, 9, 16, 11, false, false, 1, 8, 18, 31, 6, 18, 8, false, false, 10, 9)
	assertNegotiation(t, 6, 20, 10, 9, 16, 11, false, false, 1, 8, 18, 31, 6, 18, 31, false, false, 10, 11)

	assertNegotiation(t, 6, 20, 10, 9, 16, 31, false, false, 1, 8, 18, 18, 6, 18, 8, false, false, 10, 9)
	assertNegotiation(t, 6, 20, 10, 9, 16, 31, false, false, 1, 8, 18, 18, 6, 18, 31, false, false, 10, 13)
	assertNegotiation(t, 6, 20, 10, 9, 16, 31, false, false, 1, 8, 18, 31, 6, 18, 8, false, false, 10, 9)
	assertNegotiation(t, 6, 20, 10, 9, 16, 31, false, false, 1, 8, 18, 31, 6, 18, 31, false, false, 10, 13)

	assertNegotiation(t, 6, 20, 31, 9, 16, 11, false, false, 1, 8, 18, 18, 6, 18, 8, false, false, 18, 9)
	assertNegotiation(t, 6, 20, 31, 9, 16, 11, false, false, 1, 8, 18, 18, 6, 18, 31, false, false, 18, 11)
	assertNegotiation(t, 6, 20, 31, 9, 16, 11, false, false, 1, 8, 18, 31, 6, 18, 8, false, false, 13, 9)
	assertNegotiation(t, 6, 20, 31, 9, 16, 11, false, false, 1, 8, 18, 31, 6, 18, 31, false, false, 13, 11)

	assertNegotiation(t, 6, 20, 31, 9, 16, 31, false, false, 1, 8, 18, 18, 6, 18, 8, false, false, 18, 9)
	assertNegotiation(t, 6, 20, 31, 9, 16, 31, false, false, 1, 8, 18, 18, 6, 18, 31, false, false, 17, 13)
	assertNegotiation(t, 6, 20, 31, 9, 16, 31, false, false, 1, 8, 18, 31, 6, 18, 8, false, false, 13, 9)
	assertNegotiation(t, 6, 20, 31, 9, 16, 31, false, false, 1, 8, 18, 31, 6, 18, 31, false, false, 13, 13)
}

// Length (full)

func TestFullMinLengthTooLow(t *testing.T) {
	assertNegotiationFail(t, 1, 30, 15, 0, 29, 15, false, false, 1, 0, 1, 1, 0, 0, 0, false, false)
}

func TestFullMaxLengthTooLow(t *testing.T) {
	assertNegotiationFail(t, 1, 30, 15, 0, 29, 15, false, false, 1, 1, 0, 1, 0, 0, 0, false, false)
}

func TestFullMaxLengthTooHigh(t *testing.T) {
	assertNegotiationFail(t, 1, 30, 15, 0, 29, 15, false, false, 1, 1, 31, 1, 0, 0, 0, false, false)
}

func TestFullRecLengthTooLow(t *testing.T) {
	assertNegotiationFail(t, 1, 30, 15, 0, 29, 15, false, false, 1, 1, 1, 0, 0, 0, 0, false, false)
}

func TestFullLengthMinGtMax(t *testing.T) {
	assertNegotiationFail(t, 1, 30, 15, 0, 29, 15, false, false, 1, 2, 1, 2, 0, 0, 0, false, false)
}

func TestFullLengthRecGtMax(t *testing.T) {
	assertNegotiationFail(t, 1, 30, 15, 0, 29, 15, false, false, 1, 1, 1, 10, 0, 0, 0, false, false)
}

func TestFullLengthRecLtMin(t *testing.T) {
	assertNegotiationFail(t, 1, 30, 15, 0, 29, 15, false, false, 1, 2, 2, 1, 0, 0, 0, false, false)
}

// ID (full)

func TestFullMaxIdTooHigh(t *testing.T) {
	assertNegotiationFail(t, 1, 30, 15, 0, 29, 15, false, false, 1, 1, 1, 1, 0, 30, 0, false, false)
}

func TestFullIdMinGtMax(t *testing.T) {
	assertNegotiationFail(t, 1, 30, 15, 0, 29, 15, false, false, 1, 1, 1, 1, 1, 0, 1, false, false)
}

func TestFullIdRecGtMax(t *testing.T) {
	assertNegotiationFail(t, 1, 30, 15, 0, 29, 15, false, false, 1, 1, 1, 1, 0, 1, 2, false, false)
}

func TestFullIdRecLtMin(t *testing.T) {
	assertNegotiationFail(t, 1, 30, 15, 0, 29, 15, false, false, 1, 1, 1, 1, 1, 2, 0, false, false)
}

// Quick Init

func TestNegotiationQuickInitSuccess(t *testing.T) {
	assertNegotiation(t, 6, 20, 15, 6, 16, 15, true, false, 1, 8, 15, 14, 6, 18, 10, false, true, 15, 15)
}

func TestNegotiationQuickInitBadParams(t *testing.T) {
	assertNegotiationInitFail(t, 6, 20, 15, 6, 16, 15, true, true)
	assertNegotiationFail(t, 6, 20, 15, 6, 16, 15, true, false, 1, 8, 15, 14, 6, 18, 10, true, true)
}

func TestNegotiationQuickInitNotAccepting(t *testing.T) {
	assertNegotiationFail(t, 6, 20, 15, 6, 16, 15, true, false, 1, 8, 15, 14, 6, 18, 10, false, false)
	assertNegotiationFail(t, 6, 20, 15, 6, 16, 15, false, false, 1, 8, 15, 14, 6, 18, 10, true, false)
}

func TestNegotiationQuickInitParamsOutOfRange(t *testing.T) {
	assertNegotiation(t, 6, 20, 10, 6, 19, 15, true, false, 1, 8, 15, 14, 7, 18, 10, false, true, 10, 15)
	assertNegotiationFail(t, 6, 20, 6, 6, 19, 15, true, false, 1, 8, 15, 14, 7, 18, 10, false, true)
	assertNegotiationFail(t, 6, 20, 17, 6, 19, 15, true, false, 1, 8, 15, 14, 7, 18, 10, false, true)
	assertNegotiationFail(t, 6, 20, 15, 6, 19, 6, true, false, 1, 8, 15, 14, 7, 18, 10, false, true)
	assertNegotiationFail(t, 6, 20, 10, 6, 19, 19, true, false, 1, 8, 15, 14, 7, 18, 10, false, true)
}

// Spec Examples

func TestNegotiationSpecCompatible(t *testing.T) {
	assertNegotiation(t, 6, 20, 14, 6, 12, 8, false, false, 1, 5, 15, 15, 6, 15, 7, false, false, 14, 7)
}

func TestNegotiationSpecIdMinGtMax(t *testing.T) {
	assertNegotiationFail(t, 5, 12, 12, 6, 8, 8, false, false, 1, 5, 15, 15, 10, 15, 10, false, false)
}

func TestNegotiationSpecWildcardGt30(t *testing.T) {
	assertNegotiation(t, 6, 20, 31, 6, 16, 14, false, false, 1, 15, 18, 31, 6, 18, 15, false, false, 16, 14)
}

func TestNegotiationSpecAllWildcard(t *testing.T) {
	assertNegotiation(t, 6, 20, 31, 6, 16, 31, false, false, 1, 8, 15, 31, 6, 18, 31, false, false, 12, 11)
}

func TestNegotiationSpecQuickInitSuccess(t *testing.T) {
	assertNegotiation(t, 10, 18, 14, 8, 15, 8, true, false, 1, 8, 15, 10, 6, 18, 10, false, true, 14, 8)
}

func TestNegotiationSpecQuickInitOutOfRange(t *testing.T) {
	assertNegotiationFail(t, 10, 18, 16, 8, 15, 8, true, false, 1, 8, 15, 10, 6, 18, 10, false, true)
}

func TestNegotiationSpecQuickInitAllowedButNotRequested(t *testing.T) {
	assertNegotiation(t, 10, 18, 14, 8, 15, 8, false, false, 1, 8, 15, 10, 6, 18, 10, false, true, 10, 8)
}
