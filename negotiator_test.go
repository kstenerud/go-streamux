package streamux

import (
	"fmt"
	"testing"
)

func buildInitMsg(version, minLen, maxLen, recLen, minId, maxId, recId int, qiReq, qiAllowed bool) []byte {
	var qiReqValue, qiAllowedValue uint32
	if qiReq {
		qiReqValue = 1
	}
	if qiAllowed {
		qiAllowedValue = 1
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
	msg[1] = byte(fields & 0xff)
	msg[2] = byte((fields >> 8) & 0xff)
	msg[3] = byte((fields >> 16) & 0xff)
	msg[4] = byte((fields >> 24) & 0xff)

	return msg
}

func performNegotiationInit(
	usLengthMinBits int, usLengthMaxBits int, usLengthRecommendBits int,
	usIdMinBits int, usIdMaxBits int, usIdRecommendBits int,
	usRequestQuickInit bool, usAllowQuickInit bool) error {

	var negotiator negotiator_
	return negotiator.Initialize(usLengthMinBits, usLengthMaxBits,
		usLengthRecommendBits, usIdMinBits, usIdMaxBits, usIdRecommendBits,
		usRequestQuickInit, usAllowQuickInit)
}

func performNegotiation(
	usLengthMinBits int, usLengthMaxBits int, usLengthRecommendBits int,
	usIdMinBits int, usIdMaxBits int, usIdRecommendBits int,
	usRequestQuickInit bool, usAllowQuickInit bool,
	themVersion int,
	themLengthMinBits int, themLengthMaxBits int, themLengthRecommendBits int,
	themIdMinBits int, themIdMaxBits int, themIdRecommendBits int,
	themRequestQuickInit bool, themAllowQuickInit bool) error {

	var negotiator negotiator_
	if err := negotiator.Initialize(usLengthMinBits, usLengthMaxBits,
		usLengthRecommendBits, usIdMinBits, usIdMaxBits, usIdRecommendBits,
		usRequestQuickInit, usAllowQuickInit); err != nil {

		return err
	}

	message := buildInitMsg(themVersion, themLengthMinBits, themLengthMaxBits, themLengthRecommendBits,
		themIdMinBits, themIdMaxBits, themIdRecommendBits, themRequestQuickInit, themAllowQuickInit)

	messageAfter, err := negotiator.Feed(message)
	if len(messageAfter) != 0 {
		return fmt.Errorf("Expected all %v bytes to be used, but %v bytes remain", len(message), len(messageAfter))
	}

	return err
}

func assertNegotiationInitSuccess(t *testing.T, lengthMinBits int, lengthMaxBits int,
	lengthRecommendBits int, idMinBits int, idMaxBits int, idRecommendBits int,
	requestQuickInit bool, allowQuickInit bool) {

	if err := performNegotiationInit(lengthMinBits, lengthMaxBits,
		lengthRecommendBits, idMinBits, idMaxBits, idRecommendBits,
		requestQuickInit, allowQuickInit); err != nil {

		t.Error(err)
	}
}

func assertNegotiationInitFail(t *testing.T, lengthMinBits int, lengthMaxBits int,
	lengthRecommendBits int, idMinBits int, idMaxBits int, idRecommendBits int,
	requestQuickInit bool, allowQuickInit bool) {

	if err := performNegotiationInit(lengthMinBits, lengthMaxBits,
		lengthRecommendBits, idMinBits, idMaxBits, idRecommendBits,
		requestQuickInit, allowQuickInit); err == nil {

		t.Errorf("Negotiation init should have failed but didn't")
	}
}

func assertNegotiationSuccess(t *testing.T,
	usLengthMinBits int, usLengthMaxBits int, usLengthRecommendBits int,
	usIdMinBits int, usIdMaxBits int, usIdRecommendBits int,
	usRequestQuickInit bool, usAllowQuickInit bool,
	themVersion int,
	themLengthMinBits int, themLengthMaxBits int, themLengthRecommendBits int,
	themIdMinBits int, themIdMaxBits int, themIdRecommendBits int,
	themRequestQuickInit bool, themAllowQuickInit bool) {

	if err := performNegotiation(usLengthMinBits, usLengthMaxBits,
		usLengthRecommendBits, usIdMinBits, usIdMaxBits, usIdRecommendBits,
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
	usRequestQuickInit bool, usAllowQuickInit bool,
	themVersion int,
	themLengthMinBits int, themLengthMaxBits int, themLengthRecommendBits int,
	themIdMinBits int, themIdMaxBits int, themIdRecommendBits int,
	themRequestQuickInit bool, themAllowQuickInit bool) {

	if err := performNegotiation(usLengthMinBits, usLengthMaxBits,
		usLengthRecommendBits, usIdMinBits, usIdMaxBits, usIdRecommendBits,
		usRequestQuickInit, usAllowQuickInit, themVersion,
		themLengthMinBits, themLengthMaxBits, themLengthRecommendBits,
		themIdMinBits, themIdMaxBits, themIdRecommendBits,
		themRequestQuickInit, themAllowQuickInit); err == nil {

		t.Errorf("Negotiation should have failed but didn't")
	}
}

// =============================================================================

// General (init)

func TestInitValid(t *testing.T) {
	assertNegotiationInitSuccess(t, 1, 1, 1, 0, 0, 0, false, false)
}

// Length (init)

func TestMinLengthTooLow(t *testing.T) {
	assertNegotiationInitFail(t, 0, 1, 1, 0, 0, 0, false, false)
}

func TestMinLengthTooHigh(t *testing.T) {
	assertNegotiationInitFail(t, 16, 16, 1, 0, 0, 0, false, false)
}

func TestMaxLengthTooLow(t *testing.T) {
	assertNegotiationInitFail(t, 1, 0, 1, 0, 0, 0, false, false)
}

func TestMaxLengthTooHigh(t *testing.T) {
	assertNegotiationInitFail(t, 1, 31, 1, 0, 0, 0, false, false)
}

func TestRecLengthTooLow(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 0, 0, 0, 0, false, false)
}

func TestRecLengthTooHigh(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 32, 0, 0, 0, false, false)
}

// ID (init)

func TestMinIdTooLow(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, -1, 0, 0, false, false)
}

func TestMinIdTooHigh(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, 16, 0, 0, false, false)
}

func TestMaxIdTooLow(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, 0, -1, 0, false, false)
}

func TestMaxIdTooHigh(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, 0, 31, 0, false, false)
}

func TestRecIdTooLow(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, 0, 0, -1, false, false)
}

func TestRecIdTooHigh(t *testing.T) {
	assertNegotiationInitFail(t, 1, 1, 1, 0, 0, 32, false, false)
}

// General

func TestFullValid(t *testing.T) {
	assertNegotiationSuccess(t, 1, 1, 1, 0, 0, 0, false, false, 1, 1, 1, 1, 0, 0, 0, false, false)
}

// Length

// ID
