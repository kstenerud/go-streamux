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
