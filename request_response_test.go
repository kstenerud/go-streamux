package streamux

import (
	"testing"
	"time"

	"github.com/kstenerud/go-streamux/test"
)

// TODO: Response to nonexistent request
// TODO: Cancel
// TODO: Cancel nonexistent request
// TODO: Cancel ack nonexistent cancel
// TODO: Ping
// TODO: Multiplexing
// TODO: Thread safety

func TestRequestResponse(t *testing.T) {
	lengthBits := 10
	idBits := 4

	a, b, err := newTestPeerPair(t, lengthBits, idBits)
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
	if err := test.AssertSlicesAreEquivalent(actualRequest, expectedRequest); err != nil {
		t.Error(err)
	}
	actualResponse := a.GetResponse(id)
	if err := test.AssertSlicesAreEquivalent(actualResponse, expectedResponse); err != nil {
		t.Error(err)
	}
}
