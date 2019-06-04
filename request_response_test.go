package streamux

import (
	// "fmt"
	"testing"
	"time"

	"github.com/kstenerud/go-streamux/test"
)

// TODO: Multiplexing

func TestRequestResponse(t *testing.T) {
	lengthBits := 10
	idBits := 4

	a, b, err := newTestPeerPair(t, idBits, lengthBits)
	if err != nil {
		t.Error(err)
		return
	}

	expectedRequest := test.NewTestBytes(10)
	if _, err := a.SendMessage(0, expectedRequest); err != nil {
		t.Error(err)
		return
	}

	time.Sleep(time.Millisecond * 5)

	id := b.GetRequestId(0)

	expectedResponse := test.NewTestBytes(15)
	if err := b.SendResponse(0, id, expectedResponse); err != nil {
		t.Error(err)
		return
	}

	a.Close()
	b.Close()

	a.Wait()

	actualRequest := b.GetRequest(id)
	test.AssertSlicesAreEquivalent(t, actualRequest, expectedRequest)
	actualResponse := a.GetResponse(id)
	test.AssertSlicesAreEquivalent(t, actualResponse, expectedResponse)
}

func TestCancel(t *testing.T) {
	lengthBits := 10
	idBits := 4

	a, b, err := newTestPeerPair(t, idBits, lengthBits)
	if err != nil {
		t.Error(err)
		return
	}

	expectedRequest := test.NewTestBytes(10)
	messageId, err := a.SendMessage(0, expectedRequest)
	if err != nil {
		t.Error(err)
		return
	}

	if err := a.SendCancel(messageId); err != nil {
		t.Error(err)
		return
	}

	time.Sleep(time.Millisecond * 5)

	cancelAckId := a.CancelAcksReceived[0]

	a.Close()
	b.Close()

	a.Wait()

	if messageId != cancelAckId {
		t.Errorf("Canceled ID %v != cancel ack ID %v", messageId, cancelAckId)
	}
}

func TestPing(t *testing.T) {
	lengthBits := 10
	idBits := 4

	a, b, err := newTestPeerPair(t, idBits, lengthBits)
	if err != nil {
		t.Error(err)
		return
	}

	pingSendId, err := a.SendPing()
	if err != nil {
		t.Error(err)
		return
	}

	time.Sleep(time.Millisecond * 5)

	pingAckId := a.PingAcksReceived[0]

	a.Close()
	b.Close()

	a.Wait()

	if pingSendId != pingAckId {
		t.Errorf("Ping send ID %v != ping ack ID %v", pingSendId, pingAckId)
	}
}
