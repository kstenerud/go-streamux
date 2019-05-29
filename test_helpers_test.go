package streamux

import (
	"fmt"
	"testing"
)

func testForPanic(f func()) (err interface{}) {
	defer func() {
		err = recover()
	}()

	f()

	return err
}

func assertDoesNotPanic(t *testing.T, f func()) {
	if err := testForPanic(f); err != nil {
		t.Error(err)
	}
}

func assertDoesPanic(t *testing.T, f func()) {
	if err := testForPanic(f); err == nil {
		t.Errorf("Expected panic but none occurred")
	}
}

func assertSlicesAreEquivalent(actual, expected []byte) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("Slices are of diffrent lengths (actual = %v, expected = %v)", len(actual), len(expected))
	}

	for i := 0; i < len(actual); i++ {
		if actual[i] != expected[i] {
			return fmt.Errorf("Slices differ at index %v", i)
		}
	}

	return nil
}

type testPeer struct {
	protocol          *Protocol
	peer              *testPeer
	RequestsReceived  map[int][]byte
	RequestsEnded     map[int]bool
	ResponsesReceived map[int][]byte
	ResponsesEnded    map[int]bool
	RequestOrder      []int
	AbleToSend        bool
}

func (this *testPeer) OnRequestChunkReceived(messageId int, isEnd bool, data []byte) error {
	message, messageFound := this.RequestsReceived[messageId]
	endOfMessage, _ := this.RequestsEnded[messageId]

	if messageFound && endOfMessage {
		return fmt.Errorf("Request %v has already ended", messageId)
	}

	this.RequestsReceived[messageId] = append(message, data...)
	this.RequestsEnded[messageId] = isEnd
	if !messageFound {
		this.RequestOrder = append(this.RequestOrder, messageId)
	}

	return nil
}

func (this *testPeer) OnResponseChunkReceived(messageId int, isEnd bool, data []byte) error {
	message, messageFound := this.ResponsesReceived[messageId]
	endOfMessage, _ := this.ResponsesEnded[messageId]

	if messageFound && endOfMessage {
		return fmt.Errorf("Response %v has already ended", messageId)
	}

	this.ResponsesReceived[messageId] = append(message, data...)
	this.ResponsesEnded[messageId] = isEnd

	return nil
}

func (this *testPeer) OnAbleToSend() {
	this.AbleToSend = true
}

func (this *testPeer) OnMessageChunkToSend(priority int, data []byte) error {
	return this.peer.protocol.Feed(data)
}

func (this *testPeer) SendMessage(data []byte) error {
	return this.protocol.SendMessage(0, data)
}

func (this *testPeer) GetFirstRequest() []byte {
	return this.GetRequestAtIndex(0)
}

func (this *testPeer) GetRequestAtIndex(index int) []byte {
	if index < len(this.RequestOrder) {
		return this.GetRequest(this.RequestOrder[index])
	}
	panic(fmt.Errorf("Request at index %v not found", index))
}

func (this *testPeer) GetRequest(id int) []byte {
	if value, ok := this.RequestsReceived[id]; ok {
		if !this.RequestsEnded[id] {
			panic(fmt.Errorf("Request ID %v was not terminated", id))
		}
		return value
	}
	panic(fmt.Errorf("Request ID %v not found", id))
}

func (this *testPeer) GetResponse(id int) []byte {
	if value, ok := this.ResponsesReceived[id]; ok {
		if !this.ResponsesEnded[id] {
			panic(fmt.Errorf("Response ID %v was not terminated", id))
		}
		return value
	}
	panic(fmt.Errorf("Response ID %v not found", id))
}

func newTestPeer(lengthBits, idBits int) *testPeer {
	this := new(testPeer)
	this.RequestsReceived = make(map[int][]byte)
	this.RequestsEnded = make(map[int]bool)
	this.ResponsesReceived = make(map[int][]byte)
	this.ResponsesEnded = make(map[int]bool)
	this.RequestOrder = make([]int, 100)
	this.protocol = NewProtocol(1, 30, lengthBits, 0, 29, idBits, false, false, this, this)
	return this
}

func (this *testPeer) linkTo(them *testPeer) error {
	this.peer = them
	them.peer = this
	if err := this.protocol.Start(); err != nil {
		return err
	}
	return them.protocol.Start()
}

func newTestPeerPair(t *testing.T, lengthBits, idBits int) (a, b *testPeer, err error) {
	a = newTestPeer(lengthBits, idBits)
	b = newTestPeer(lengthBits, idBits)

	return a, b, a.linkTo(b)
}

func newTestData(length int) []byte {
	data := make([]byte, length)
	for i := 0; i < length; i++ {
		data[i] = byte(i & 0xff)
	}
	return data
}
