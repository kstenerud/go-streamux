package streamux

import (
	"fmt"
	"sync"
	"testing"
	"time"
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
	t                 *testing.T
	protocol          *Protocol
	sendChannel       chan []byte
	recvChannel       chan []byte
	wg                *sync.WaitGroup
	RequestsReceived  map[int][]byte
	RequestsEnded     map[int]bool
	ResponsesReceived map[int][]byte
	ResponsesEnded    map[int]bool
	RequestOrder      []int
	AbleToSend        bool
	NegotiationFailed bool
}

func newTestPeer(t *testing.T, lengthBits, idBits int, isServer bool, sendChannel chan []byte, recvChannel chan []byte, wg *sync.WaitGroup) *testPeer {
	this := new(testPeer)
	this.t = t
	this.RequestsReceived = make(map[int][]byte)
	this.RequestsEnded = make(map[int]bool)
	this.ResponsesReceived = make(map[int][]byte)
	this.ResponsesEnded = make(map[int]bool)
	this.RequestOrder = make([]int, 0, 100)
	this.sendChannel = sendChannel
	this.recvChannel = recvChannel
	this.wg = wg

	requestQuickInit := true
	allowQuickInit := false
	if isServer {
		requestQuickInit = false
		allowQuickInit = true
	}
	this.protocol = NewProtocol(1, 30, lengthBits, 0, 29, idBits, requestQuickInit, allowQuickInit, this, this)
	return this
}

func (this *testPeer) OnPingReceived() {

}

func (this *testPeer) OnPingAckReceived(latency time.Duration) {

}

func (this *testPeer) OnCancelReceived(messageId int) {

}

func (this *testPeer) OnCancelAckReceived(messageId int) {

}

func (this *testPeer) OnRequestChunkReceived(messageId int, isEnd bool, data []byte) error {
	// fmt.Printf("### TP %p: Received request id %v, %v bytes, end %v\n", this, messageId, len(data), isEnd)
	message, messageFound := this.RequestsReceived[messageId]
	endOfMessage, _ := this.RequestsEnded[messageId]

	if messageFound && endOfMessage {
		return fmt.Errorf("Request %v has already ended", messageId)
	}

	this.RequestsReceived[messageId] = append(message, data...)
	this.RequestsEnded[messageId] = isEnd
	if !messageFound {
		// fmt.Printf("### Adding id %v to request order\n", messageId)
		this.RequestOrder = append(this.RequestOrder, messageId)
	}

	return nil
}

func (this *testPeer) OnResponseChunkReceived(messageId int, isEnd bool, data []byte) error {
	// fmt.Printf("### TP %p: Receive response id %v, %v bytes, end %v\n", this, messageId, len(data), isEnd)
	message, messageFound := this.ResponsesReceived[messageId]
	endOfMessage, _ := this.ResponsesEnded[messageId]

	if messageFound && endOfMessage {
		return fmt.Errorf("Response %v has already ended", messageId)
	}

	this.ResponsesReceived[messageId] = append(message, data...)
	this.ResponsesEnded[messageId] = isEnd

	return nil
}

func (this *testPeer) OnNegotiationFailed() {
	// fmt.Printf("### %TP p: Negotiation failed\n", this)
	this.NegotiationFailed = true
}

func (this *testPeer) OnAbleToSend() {
	// fmt.Printf("### Able to send\n")
	this.AbleToSend = true
}

func (this *testPeer) OnMessageChunkToSend(priority int, data []byte) error {
	// fmt.Printf("### TP %p: Sending message chunk size %v\n", this, len(data))
	toSend := make([]byte, len(data))
	copy(toSend, data)
	this.sendChannel <- toSend
	return nil
}

func (this *testPeer) SendMessage(priority int, data []byte) (int, error) {
	return this.protocol.SendMessage(priority, data)
}

func (this *testPeer) SendResponse(priority int, id int, data []byte) error {
	return this.protocol.SendResponseMessage(priority, id, data)
}

func (this *testPeer) GetFirstRequest() []byte {
	return this.GetRequestAtIndex(0)
}

func (this *testPeer) GetRequestAtIndex(index int) []byte {
	return this.GetRequest(this.GetRequestId(index))
}

func (this *testPeer) GetRequestId(index int) int {
	if index < len(this.RequestOrder) {
		return this.RequestOrder[index]
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

func (this *testPeer) BeginInitialization() error {
	return this.protocol.BeginInitialization()
}

func (this *testPeer) BeginFeeding() {
	this.wg.Add(1)
	go func() {
		defer this.wg.Done()
		for data := range this.recvChannel {
			// fmt.Printf("### Reading chunk of %v bytes\n", len(data))
			if err := this.protocol.Feed(data); err != nil {
				this.t.Error(err)
				return
			}
		}
	}()
}

func (this *testPeer) Wait() {
	this.wg.Wait()
}

func (this *testPeer) Close() {
	close(this.sendChannel)
}

func newTestPeerPair(t *testing.T, lengthBits, idBits int) (client, server *testPeer, err error) {
	wg := new(sync.WaitGroup)
	clientSendsChannel := make(chan []byte)
	serverSendsChannel := make(chan []byte)
	client = newTestPeer(t, lengthBits, idBits, false, clientSendsChannel, serverSendsChannel, wg)
	server = newTestPeer(t, lengthBits, idBits, true, serverSendsChannel, clientSendsChannel, wg)

	client.BeginFeeding()
	server.BeginFeeding()

	if err := client.BeginInitialization(); err != nil {
		return nil, nil, err
	}

	if err := server.BeginInitialization(); err != nil {
		return nil, nil, err
	}

	return client, server, nil
}

func newTestData(length int) []byte {
	data := make([]byte, length)
	for i := 0; i < length; i++ {
		data[i] = byte(i & 0xff)
	}
	return data
}
