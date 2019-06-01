package test

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

func AssertDoesNotPanic(t *testing.T, f func()) {
	if err := testForPanic(f); err != nil {
		t.Error(err)
	}
}

func AssertDoesPanic(t *testing.T, f func()) {
	if err := testForPanic(f); err == nil {
		t.Errorf("Expected panic but none occurred")
	}
}

func AssertSlicesAreEquivalent(actual, expected []byte) error {
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

func NewTestBytes(length int) []byte {
	data := make([]byte, length)
	for i := 0; i < length; i++ {
		data[i] = byte(i & 0xff)
	}
	return data
}
