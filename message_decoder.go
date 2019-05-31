package streamux

import (
	"fmt"
)

type messageDecoder_ struct {
	header         messageHeader_
	bytesRemaining int
	callbacks      MessageReceiveCallbacks
}

func (this *messageDecoder_) reset() {
	this.header.ClearEncoded()
}

func (this *messageDecoder_) isMessageChunkComplete() bool {
	return this.bytesRemaining == 0
}

func newMessageDecoder(lengthBits int, idBits int, callbacks MessageReceiveCallbacks) *messageDecoder_ {
	this := new(messageDecoder_)
	this.Init(lengthBits, idBits, callbacks)
	return this
}

func (this *messageDecoder_) Init(lengthBits int, idBits int, callbacks MessageReceiveCallbacks) {
	this.header.Init(lengthBits, idBits)
	this.callbacks = callbacks
	this.reset()
}

func (this *messageDecoder_) notifyMessageData(chunk []byte) error {
	if this.header.IsResponse {
		if err := this.callbacks.OnResponseChunkReceived(this.header.Id, this.header.IsEndOfMessage, chunk); err != nil {
			return err
		}
	} else {
		if err := this.callbacks.OnRequestChunkReceived(this.header.Id, this.header.IsEndOfMessage, chunk); err != nil {
			return err
		}
	}
	if this.isMessageChunkComplete() {
		this.reset()
	}
	return nil
}

func (this *messageDecoder_) Feed(incomingStreamData []byte) (remainingData []byte, err error) {
	// fmt.Printf("### D %p: feed id %v. Data length %v. Is header decoded: %v\n", this, this.id, len(incomingStreamData), this.isHeaderDecoded)
	remainingData = incomingStreamData
	if !this.header.IsDecoded() {
		remainingData = this.header.Feed(remainingData)
		if !this.header.IsDecoded() {
			if len(remainingData) != 0 {
				return remainingData, fmt.Errorf("INTERNAL BUG: %v bytes in incoming stream, but header still not decoded", len(remainingData))
			}
			return remainingData, nil
		}
		this.bytesRemaining = this.header.Length
		switch this.header.MessageType {
		case messageTypeCancel:
			// fmt.Printf("### Cancel\n")
			// this.callbacks.OnCancelReceived(this.header.Id)
			// return remainingData, nil
		case messageTypeCancelAck:
			// fmt.Printf("### Cancel Ack\n")
			// this.callbacks.OnCancelAckReceived(this.header.Id)
			// return remainingData, nil
		case messageTypePing:
			// fmt.Printf("### Ping\n")
			// this.callbacks.OnPingReceived(this.header.Id)
			// return remainingData, nil
		case messageTypeEmptyResponse:
			// this.callbacks.OnEmptyResponseReceived(this.header.Id)
			// return remainingData, nil
		default:
		}
	}

	var decodedData []byte

	if this.isMessageChunkComplete() {
		// fmt.Printf("#### D %p: Message chunk complete. Length %v\n", this, len(decodedData))
		return remainingData, this.notifyMessageData(decodedData)
	}

	for len(remainingData) > 0 && !this.isMessageChunkComplete() {
		decodedData, remainingData = useBytes(this.bytesRemaining, remainingData)
		this.bytesRemaining -= len(decodedData)
		// fmt.Printf("#### D %p: Message data. Length %v\n", this, len(decodedData))
		if err := this.notifyMessageData(decodedData); err != nil {
			fmt.Printf("ERROR %v\n", err)
			return remainingData, err
		}
	}
	return remainingData, nil
}
