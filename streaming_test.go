package streamux

import (
	"testing"

	"github.com/kstenerud/go-streamux/test"
)

func assertStreamData(t *testing.T, idBits int, lengthBits int, dataSize int) {
	a, b, err := newTestPeerPair(t, idBits, lengthBits)
	if err != nil {
		t.Error(err)
		return
	}

	expected := test.NewTestBytes(dataSize)
	if _, err := a.SendMessage(0, expected); err != nil {
		t.Error(err)
		return
	}

	a.Close()
	b.Close()

	a.Wait()

	actual := b.GetFirstRequest()
	test.AssertSlicesAreEquivalent(t, actual, expected)
}

// =============================================================================

func TestStream1(t *testing.T) {
	for i := 20; i >= 2; i-- {
		assertStreamData(t, 10, i, 100000)
	}
}

func TestStream2(t *testing.T) {
	for i := 15; i >= 2; i-- {
		assertStreamData(t, 2, i, 4096)
	}
	for i := 15; i >= 2; i-- {
		assertStreamData(t, 2, i, 4095)
	}
	for i := 15; i >= 2; i-- {
		assertStreamData(t, 2, i, 4094)
	}
}

func TestStream3(t *testing.T) {
	for i := 10; i >= 1; i-- {
		assertStreamData(t, 0, i, 10)
	}
}
