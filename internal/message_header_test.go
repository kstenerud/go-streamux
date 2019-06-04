package internal

import (
	"testing"

	"github.com/kstenerud/go-streamux/test"
)

func assertEncoded(t *testing.T, idBits int, lengthBits int, id int, length int, isResponse bool, isEnd bool, expected []byte) {
	header := NewMessageHeader(idBits, lengthBits)
	header.SetAll(id, length, isResponse, isEnd)
	test.AssertSlicesAreEquivalent(header.Encoded.Data, expected)

	header = NewMessageHeader(idBits, lengthBits)
	remainingBytes := header.Feed(expected)
	if len(remainingBytes) > 0 {
		t.Errorf("[%v %v %v %v %v %v] Remaining length should have been 0 but was %v",
			idBits, lengthBits, id, length, isResponse, isEnd, len(remainingBytes))
	}
	if !header.IsDecoded() {
		t.Errorf("[%v %v %v %v %v %v] Header should be decoded but isn't",
			idBits, lengthBits, id, length, isResponse, isEnd)
	}
	if header.Id != id {
		t.Errorf("[%v %v %v %v %v %v] Expected ID of %v but got %v",
			idBits, lengthBits, id, length, isResponse, isEnd, header.Id, id)
	}
	if header.Length != length {
		t.Errorf("[%v %v %v %v %v %v] Expected length of %v but got %v",
			idBits, lengthBits, id, length, isResponse, isEnd, header.Length, length)
	}
	if header.IsResponse != isResponse {
		t.Errorf("[%v %v %v %v %v %v] Expected response bit of %v but got %v",
			idBits, lengthBits, id, length, isResponse, isEnd, header.IsResponse, isResponse)
	}
	if header.IsEndOfMessage != isEnd {
		t.Errorf("[%v %v %v %v %v %v] Expected termination bit of %v but got %v",
			idBits, lengthBits, id, length, isResponse, isEnd, header.IsEndOfMessage, isEnd)
	}
}

// func TestStuff(t *testing.T) {
// 	assertEncoded(t, 1, 1, 0, 0, false, false, []byte{0x00})
// }
