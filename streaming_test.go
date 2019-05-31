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
	if _, err := a.SendMessage(0, expected); err != nil {
		t.Error(err)
		return
	}

	a.Close()
	b.Close()

	a.Wait()

	actual := b.GetFirstRequest()
	if err := assertSlicesAreEquivalent(actual, expected); err != nil {
		t.Error(err)
	}
}

// =============================================================================

func TestStream1(t *testing.T) {
	for i := 20; i >= 2; i-- {
		assertStreamData(t, i, 10, 100000)
	}
}

func TestStream2(t *testing.T) {
	for i := 15; i >= 2; i-- {
		assertStreamData(t, i, 2, 4096)
	}
	for i := 15; i >= 2; i-- {
		assertStreamData(t, i, 2, 4095)
	}
	for i := 15; i >= 2; i-- {
		assertStreamData(t, i, 2, 4094)
	}
}

func TestStream3(t *testing.T) {
	for i := 10; i >= 1; i-- {
		assertStreamData(t, i, 0, 10)
	}
}
