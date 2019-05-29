package streamux

import (
	"testing"
)

func assertStreamData(t *testing.T, lengthBits int, idBits int, dataSize int) {
	a, b, err := newTestPeerPair(t, lengthBits, idBits)
	if err != nil {
		t.Error(err)
		return
	}

	expected := newTestData(dataSize)
	a.SendMessage(expected)
	actual := b.GetFirstRequest()
	if err := assertSlicesAreEquivalent(actual, expected); err != nil {
		t.Error(err)
	}
}

// =============================================================================

func TestStream(t *testing.T) {
	for i := 20; i >= 2; i-- {
		assertStreamData(t, i, 10, 100000)
	}
	for i := 20; i >= 2; i-- {
		assertStreamData(t, i, 2, 99999)
	}
	for i := 10; i >= 1; i-- {
		assertStreamData(t, i, 0, 10)
	}
}

func TestEmpty(t *testing.T) {
	for i := 30; i >= 1; i-- {
		assertStreamData(t, i, 0, 0)
	}
}
