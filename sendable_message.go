package streamux

import (
	"fmt"
)

const maxInitialBufferCapacity = 1028

type SendableMessage struct {
	Id int

	priority int
	header   messageHeader_
	data     []byte
	isClosed bool

	idPool    *idPool
	callbacks MessageSendCallbacks
}

func (this *SendableMessage) dataLength() int {
	return len(this.data) - this.header.HeaderLength
}

func (this *SendableMessage) freeChunkSpace() int {
	return this.header.MaxChunkLength - this.dataLength()
}

func (this *SendableMessage) sendCurrentChunk(termination bool) error {
	this.header.SetLengthAndTermination(this.dataLength(), termination)
	copy(this.data, this.header.Encoded)
	err := this.callbacks.OnMessageChunkToSend(this.priority, this.data)
	this.data = this.data[0:this.header.HeaderLength]
	return err
}

func newSendableMessage(idPool *idPool, callbacks MessageSendCallbacks,
	priority int, id int, lengthBits int, idBits int, isResponse bool) *SendableMessage {

	this := new(SendableMessage)
	this.Init(idPool, callbacks, priority, id, lengthBits, idBits, isResponse)
	return this
}

func (this *SendableMessage) Init(idPool *idPool, callbacks MessageSendCallbacks,
	priority int, id int, lengthBits int, idBits int, isResponse bool) {

	this.Id = id
	this.idPool = idPool
	this.callbacks = callbacks
	this.priority = priority
	this.header.Init(lengthBits, idBits)
	this.header.SetIdAndResponseNoEncode(id, isResponse)

	initialBufferCapacity := this.header.MaxChunkLength
	if initialBufferCapacity > maxInitialBufferCapacity {
		initialBufferCapacity = maxInitialBufferCapacity
	}
	this.data = make([]byte, this.header.HeaderLength, initialBufferCapacity)
}

func (this *SendableMessage) AddData(bytesToSend []byte, isEndOfData bool) error {
	if this.isClosed {
		return fmt.Errorf("Message has been closed")
	}

	// TODO: Don't allow end of data if no bytes have been added since the beginning
	// except for empty response...

	for len(bytesToSend) > this.freeChunkSpace() {
		// fmt.Printf("### calc: max chunk length %v, data %v, header length %v\n", this.maxChunkLength, len(this.data), this.headerLength)
		appendByteCount := this.freeChunkSpace()
		bytesToAppend := bytesToSend[:appendByteCount]
		// fmt.Printf("### add data: free space %v, bytes to append %v, bytes to send %v, data %v\n",
		// 	this.freeChunkSpace(), len(bytesToAppend), len(bytesToSend), len(this.data))

		bytesToSend = bytesToSend[appendByteCount:]
		this.data = append(this.data, bytesToAppend...)
		// fmt.Printf("### bytes to send now %v, data now %v\n", len(bytesToSend), len(this.data))
		termination := false
		if err := this.sendCurrentChunk(termination); err != nil {
			// fmt.Printf("### ERROR %v\n", err)
			return err
		}
	}

	this.data = append(this.data, bytesToSend...)
	if err := this.sendCurrentChunk(isEndOfData); err != nil {
		return err
	}

	if isEndOfData {
		this.Close()
	}

	return nil
}

func (this *SendableMessage) Close() {
	if this.isClosed {
		return
	}
	if this.idPool != nil {
		this.idPool.DeallocateId(this.Id)
	}
	this.isClosed = true
}
