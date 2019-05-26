package streamux

import (
	"testing"
)

type testStreamer struct {
	t                *testing.T
	protocol         *Protocol
	peer             *testStreamer
	requestReceived  []byte
	requestEnded     bool
	responseReceived []byte
	responseEnded    bool
}

func (this *testStreamer) OnRequestChunkReceived(messageId int, isEnd bool, data []byte) {
	this.requestReceived = append(this.requestReceived, data...)
	this.requestEnded = isEnd
}

func (this *testStreamer) OnResponseChunkReceived(messageId int, isEnd bool, data []byte) {
	this.responseReceived = append(this.responseReceived, data...)
	this.responseEnded = isEnd
}

func (this *testStreamer) OnMessageChunkToSend(priority int, data []byte) {
	if err := this.peer.protocol.Feed(data); err != nil {
		this.t.Error(err)
	}
}

func (this *testStreamer) Stream(data []byte) {
	this.protocol.SendMessage(0, data)
}

func newStreamerPair(t *testing.T, lengthBits, idBits int) (a, b *testStreamer) {
	a = new(testStreamer)
	b = new(testStreamer)

	a.t = t
	a.peer = b
	a.protocol = NewProtocol(1, 29, lengthBits, 1, 29, idBits, false, false, a, a)

	b.t = t
	b.peer = a
	b.protocol = NewProtocol(1, 29, lengthBits, 1, 29, idBits, false, false, b, b)

	return a, b
}

func newTestData(length int) []byte {
	data := make([]byte, length)
	for i := 0; i < length; i++ {
		data[i] = byte(i & 0xff)
	}
	return data
}

func assertStreamData(t *testing.T, lengthBits int, idBits int, dataSize int) {
	a, b := newStreamerPair(t, lengthBits, idBits)
	expected := newTestData(dataSize)
	a.Stream(expected)
	actual := b.requestReceived
	if err := assertSlicesAreEquivalent(actual, expected); err != nil {
		t.Error(err)
	}
}

func TestStream(t *testing.T) {
	assertStreamData(t, 10, 10, 10000)
}
