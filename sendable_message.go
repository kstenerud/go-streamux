package streamux

import (
	"fmt"

	// "github.com/kstenerud/go-streamux/common"
	"github.com/kstenerud/go-streamux/internal"
	"github.com/kstenerud/go-streamux/internal/buffer"
)

const maxInitialBufferCapacity = 1024 + 4

// A SendableMessage is part of the advanced streamux API, allowing data to be
// fed into a message incrementally.
type SendableMessage struct {
	Id int

	priority   int
	header     internal.MessageHeader
	chunkData  buffer.FeedableBuffer
	isEnded    bool
	chunksSent int

	messageSender internal.InternalMessageSender
}

// API

func newSendableMessage(messageSender internal.InternalMessageSender, priority int, id int,
	idBits int, lengthBits int, isResponse bool) *SendableMessage {

	this := new(SendableMessage)
	this.Init(messageSender, priority, id, idBits, lengthBits, isResponse)
	return this
}

func (this *SendableMessage) Init(messageSender internal.InternalMessageSender,
	priority int, id int, idBits int, lengthBits int, isResponse bool) {

	this.Id = id
	this.messageSender = messageSender
	this.priority = priority
	this.header.Init(idBits, lengthBits)
	this.header.SetIdAndResponseNoEncode(id, isResponse)

	initialBufferCapacity := this.header.HeaderLength + this.header.MaxChunkLength
	if initialBufferCapacity > maxInitialBufferCapacity {
		initialBufferCapacity = maxInitialBufferCapacity
	}
	this.chunkData.Init(this.header.HeaderLength,
		this.header.HeaderLength+this.header.MaxChunkLength, initialBufferCapacity)
}

// Feed more data into the message. Data is sent in chunks of the maximum chunk
// size. Any remaining data that doesn't fill a full chunk will be buffered
// until the next call to Feed(), Flush(), or End().
func (this *SendableMessage) Feed(bytesToSend []byte) error {
	if this.isEnded {
		return fmt.Errorf("Cannot add more data: message has ended")
	}

	for len(bytesToSend) > this.chunkData.GetFreeByteCount() {
		bytesToSend = this.chunkData.Feed(bytesToSend)
		if err := this.sendCurrentChunk(); err != nil {
			return err
		}
	}

	this.chunkData.Feed(bytesToSend)

	return nil
}

// Send the next chunk of data, even if the buffer isn't completely full.
// This will send a non-terminated chunk that is less than the maximum chunk size.
// It won't send an empty chunk if there's no buffered data.
func (this *SendableMessage) Flush() error {
	if this.getDataLength() > 0 {
		return this.sendCurrentChunk()
	}
	return nil
}

// End this message, flushing the last of the data with the `terminate` bit set.
// This function will send a chunk even if there's no buffered data, because the
// peer needs to receive a terminated chunk to know that the message is finished.
func (this *SendableMessage) End() error {
	if this.isEnded {
		return nil
	}

	this.isEnded = true
	this.header.SetLengthAndTermination(this.getDataLength(), this.isEnded)

	switch this.header.MessageType {
	case internal.MessageTypeRequestEmptyTermination:
		if this.chunksSent == 0 {
			return fmt.Errorf("A request message must contain at least 1 byte of payload")
		}
	case internal.MessageTypeCancel, internal.MessageTypeCancelAck:
		return fmt.Errorf("Internal bug: Message type %v should not be possible", this.header.MessageType)
	case internal.MessageTypeRequest, internal.MessageTypeResponse, internal.MessageTypeEmptyResponse:
		// These are allowed
	default:
		return fmt.Errorf("Internal bug: Unhandled message type: %v", this.header.MessageType)
	}

	return this.sendCurrentChunk()
}

// Internal

func (this *SendableMessage) getDataLength() int {
	return this.chunkData.GetUsedByteCountOverMinimum()
}

func (this *SendableMessage) sendCurrentChunk() (err error) {
	// fmt.Printf("### SM %p: Send chunk length %v, response %v, end %v\n", this, this.getDataLength(), this.header.IsResponse, this.isEnded)
	this.header.SetLengthAndTermination(this.getDataLength(), this.isEnded)
	this.chunkData.OverwriteHead(this.header.Encoded.Data)
	if this.header.IsResponse {
		err = this.messageSender.OnResponseChunkToSend(this.priority, this.Id, this.header.IsEndOfMessage, this.chunkData.Data)
	} else {
		err = this.messageSender.OnRequestChunkToSend(this.priority, this.Id, this.header.IsEndOfMessage, this.chunkData.Data)
	}
	this.chunkData.Minimize()
	this.chunksSent++
	return err
}
