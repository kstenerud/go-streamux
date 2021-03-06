package internal

import (
	"testing"

	"github.com/kstenerud/go-streamux/test"
)

func assertHeaderEncoded(t *testing.T, idBits int, lengthBits int, id int, length int, isResponse bool, isEnd bool, expected []byte) {
	header := NewMessageHeader(idBits, lengthBits)
	header.SetAll(id, length, isResponse, isEnd)
	test.AssertSlicesAreEquivalent(t, header.Encoded.Data, expected)

	header = NewMessageHeader(idBits, lengthBits)
	remainingBytes, err := header.Feed(expected)
	if err != nil {
		t.Error(err)
		return
	}
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

func assertMessageTypeEncoded(t *testing.T, idBits int, lengthBits int, id int, messageType MessageType, expected []byte) {
	header := NewMessageHeader(idBits, lengthBits)
	header.SetIdAndType(id, messageType)
	test.AssertSlicesAreEquivalent(t, header.Encoded.Data, expected)
}

func assertHeaderFails(t *testing.T, idBits int, lengthBits int, encodedHeader []byte) {
	header := NewMessageHeader(idBits, lengthBits)
	_, err := header.Feed(encodedHeader)
	if err == nil {
		t.Errorf("Should have generated an error")
	}
}

// =============================================================================

func TestHeaderCancel(t *testing.T) {
	assertMessageTypeEncoded(t, 0, 1, 0, MessageTypeCancel, []byte{0x00})
	assertMessageTypeEncoded(t, 2, 3, 1, MessageTypeCancel, []byte{0x20})
	assertMessageTypeEncoded(t, 10, 3, 2, MessageTypeCancel, []byte{0x40, 0x00})
	assertMessageTypeEncoded(t, 10, 6, 20, MessageTypeCancel, []byte{0x00, 0x14, 0x00})
	assertMessageTypeEncoded(t, 10, 15, 500, MessageTypeCancel, []byte{0x00, 0x00, 0xe8, 0x03})
}

func TestHeaderCancelAck(t *testing.T) {
	assertMessageTypeEncoded(t, 0, 1, 0, MessageTypeCancelAck, []byte{0x02})
	assertMessageTypeEncoded(t, 2, 3, 1, MessageTypeCancelAck, []byte{0x22})
	assertMessageTypeEncoded(t, 10, 3, 2, MessageTypeCancelAck, []byte{0x42, 0x00})
	assertMessageTypeEncoded(t, 10, 6, 20, MessageTypeCancelAck, []byte{0x02, 0x14, 0x00})
	assertMessageTypeEncoded(t, 10, 15, 500, MessageTypeCancelAck, []byte{0x02, 0x00, 0xe8, 0x03})
}

func TestHeaderPing(t *testing.T) {
	assertMessageTypeEncoded(t, 0, 1, 0, MessageTypeRequestEmptyTermination, []byte{0x01})
	assertMessageTypeEncoded(t, 2, 3, 1, MessageTypeRequestEmptyTermination, []byte{0x21})
	assertMessageTypeEncoded(t, 10, 3, 2, MessageTypeRequestEmptyTermination, []byte{0x41, 0x00})
	assertMessageTypeEncoded(t, 10, 6, 20, MessageTypeRequestEmptyTermination, []byte{0x01, 0x14, 0x00})
	assertMessageTypeEncoded(t, 10, 15, 500, MessageTypeRequestEmptyTermination, []byte{0x01, 0x00, 0xe8, 0x03})
}

func TestHeaderPingAck(t *testing.T) {
	assertMessageTypeEncoded(t, 0, 1, 0, MessageTypeEmptyResponse, []byte{0x03})
	assertMessageTypeEncoded(t, 2, 3, 1, MessageTypeEmptyResponse, []byte{0x23})
	assertMessageTypeEncoded(t, 10, 3, 2, MessageTypeEmptyResponse, []byte{0x43, 0x00})
	assertMessageTypeEncoded(t, 10, 6, 20, MessageTypeEmptyResponse, []byte{0x03, 0x14, 0x00})
	assertMessageTypeEncoded(t, 10, 15, 500, MessageTypeEmptyResponse, []byte{0x03, 0x00, 0xe8, 0x03})
}

func TestHeader0Bit1Bit(t *testing.T) {
	assertHeaderEncoded(t, 0, 1, 0, 0, false, false, []byte{0x00})

	assertHeaderEncoded(t, 0, 1, 0, 0, false, true, []byte{0x01})
	assertHeaderEncoded(t, 0, 1, 0, 0, true, false, []byte{0x02})
	assertHeaderEncoded(t, 0, 1, 0, 0, true, true, []byte{0x03})

	assertHeaderEncoded(t, 0, 1, 0, 1, false, false, []byte{0x04})
}

func TestHeader1Bit1Bit(t *testing.T) {
	assertHeaderEncoded(t, 1, 1, 0, 0, false, false, []byte{0x00})

	assertHeaderEncoded(t, 1, 1, 0, 0, false, true, []byte{0x01})
	assertHeaderEncoded(t, 1, 1, 0, 0, true, false, []byte{0x02})
	assertHeaderEncoded(t, 1, 1, 0, 0, true, true, []byte{0x03})

	assertHeaderEncoded(t, 1, 1, 0, 1, false, false, []byte{0x04})
	assertHeaderEncoded(t, 1, 1, 1, 0, false, false, []byte{0x08})
	assertHeaderEncoded(t, 1, 1, 1, 1, false, false, []byte{0x0c})
}

func TestHeader2Bit4Bit(t *testing.T) {
	assertHeaderEncoded(t, 2, 4, 0, 0, false, false, []byte{0x00})
	assertHeaderEncoded(t, 2, 4, 0, 1, false, false, []byte{0x04})
	assertHeaderEncoded(t, 2, 4, 3, 5, false, false, []byte{0xd4})
	assertHeaderEncoded(t, 2, 4, 3, 5, false, true, []byte{0xd5})
	assertHeaderEncoded(t, 2, 4, 3, 5, true, false, []byte{0xd6})
	assertHeaderEncoded(t, 2, 4, 3, 5, true, true, []byte{0xd7})
}

func TestHeader5Bit7Bit(t *testing.T) {
	assertHeaderEncoded(t, 5, 7, 0, 0, false, false, []byte{0x00, 0x00})
	assertHeaderEncoded(t, 5, 7, 0, 1, false, false, []byte{0x04, 0x00})
	assertHeaderEncoded(t, 5, 7, 1, 0, false, false, []byte{0x00, 0x02})
	assertHeaderEncoded(t, 5, 7, 3, 5, false, false, []byte{0x14, 0x06})
	assertHeaderEncoded(t, 5, 7, 3, 5, false, true, []byte{0x15, 0x06})
	assertHeaderEncoded(t, 5, 7, 3, 5, true, false, []byte{0x16, 0x06})
	assertHeaderEncoded(t, 5, 7, 3, 5, true, true, []byte{0x17, 0x06})
}

func TestHeader8Bit12Bit(t *testing.T) {
	assertHeaderEncoded(t, 8, 12, 0, 0, false, false, []byte{0x00, 0x00, 0x00})
	assertHeaderEncoded(t, 8, 12, 0, 1, false, false, []byte{0x04, 0x00, 0x00})
	assertHeaderEncoded(t, 8, 12, 1, 0, false, false, []byte{0x00, 0x40, 0x00})
	assertHeaderEncoded(t, 8, 12, 60, 4000, false, false, []byte{0x80, 0x3e, 0x0f})
	assertHeaderEncoded(t, 8, 12, 60, 4000, false, true, []byte{0x81, 0x3e, 0x0f})
	assertHeaderEncoded(t, 8, 12, 60, 4000, true, false, []byte{0x82, 0x3e, 0x0f})
	assertHeaderEncoded(t, 8, 12, 60, 4000, true, true, []byte{0x83, 0x3e, 0x0f})
}

func TestHeader13Bit14Bit(t *testing.T) {
	assertHeaderEncoded(t, 13, 14, 0, 0, false, false, []byte{0x00, 0x00, 0x00, 0x00})
	assertHeaderEncoded(t, 13, 14, 0, 1, false, false, []byte{0x04, 0x00, 0x00, 0x00})
	assertHeaderEncoded(t, 13, 14, 1, 0, false, false, []byte{0x00, 0x00, 0x01, 0x00})
	assertHeaderEncoded(t, 13, 14, 400, 10000, false, false, []byte{0x40, 0x9c, 0x90, 0x01})
	assertHeaderEncoded(t, 13, 14, 400, 10000, false, true, []byte{0x41, 0x9c, 0x90, 0x01})
	assertHeaderEncoded(t, 13, 14, 400, 10000, true, false, []byte{0x42, 0x9c, 0x90, 0x01})
	assertHeaderEncoded(t, 13, 14, 400, 10000, true, true, []byte{0x43, 0x9c, 0x90, 0x01})
}

func TestHeaderUnusedBitsSet(t *testing.T) {
	assertHeaderFails(t, 2, 2, []byte{0x40})
	assertHeaderFails(t, 10, 2, []byte{0x40, 0x80})
	assertHeaderFails(t, 10, 10, []byte{0x40, 0x00, 0xff})
	assertHeaderFails(t, 12, 13, []byte{0x40, 0x00, 0x00, 0xff})
}
