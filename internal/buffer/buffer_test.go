package buffer

import (
	"testing"

	"github.com/kstenerud/go-streamux/test"
)

func TestMinimize(t *testing.T) {
	expected := 2
	buffer := New(expected, 5, 5)
	if len(buffer.Data) != expected {
		t.Errorf("Expected size %v but got %v", expected, len(buffer.Data))
	}

	buffer.Feed([]byte{0, 0})
	buffer.Minimize()
	if len(buffer.Data) != expected {
		t.Errorf("Expected size %v but got %v", expected, len(buffer.Data))
	}
}

func TestMaximize(t *testing.T) {
	expected := 4
	buffer := New(1, expected, expected)
	buffer.Maximize()
	if len(buffer.Data) != expected {
		t.Errorf("Expected size %v but got %v", expected, len(buffer.Data))
	}
}

func TestMaximizeWithResize(t *testing.T) {
	expected := 4
	buffer := New(1, expected, 1)
	buffer.Maximize()
	if len(buffer.Data) != expected {
		t.Errorf("Expected size %v but got %v", expected, len(buffer.Data))
	}
}

func TestFreeByteCount(t *testing.T) {
	expected := 3
	buffer := New(2, 5, 5)
	if buffer.GetFreeByteCount() != expected {
		t.Errorf("Expected count %v but got %v", expected, buffer.GetFreeByteCount())
	}

	buffer.Feed([]byte{0, 0})
	expected = 1
	if buffer.GetFreeByteCount() != expected {
		t.Errorf("Expected count %v but got %v", expected, buffer.GetFreeByteCount())
	}
}

func TestUsedByteCount(t *testing.T) {
	expected := 0
	buffer := New(2, 5, 5)
	if buffer.GetUsedByteCountOverMinimum() != expected {
		t.Errorf("Expected count %v but got %v", expected, buffer.GetUsedByteCountOverMinimum())
	}

	buffer.Feed([]byte{0, 0})
	expected = 2
	if buffer.GetUsedByteCountOverMinimum() != expected {
		t.Errorf("Expected count %v but got %v", expected, buffer.GetUsedByteCountOverMinimum())
	}

	buffer.Minimize()
	expected = 0
	if buffer.GetUsedByteCountOverMinimum() != expected {
		t.Errorf("Expected count %v but got %v", expected, buffer.GetUsedByteCountOverMinimum())
	}
}

func TestOverwriteHead(t *testing.T) {
	expected := []byte{1, 2, 3}
	buffer := New(2, 5, 5)
	buffer.ExpandTo(4)
	buffer.OverwriteHead(expected)
	actual := buffer.Data[:len(expected)]
	test.AssertSlicesAreEquivalent(t, actual, expected)
}
