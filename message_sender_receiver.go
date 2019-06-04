package streamux

import "time"

// MessageReceiver receives complete messages or message chunks from the remote peer.
type MessageReceiver interface {
	// Signals that the other peer has sent you a request chunk.
	OnRequestChunkReceived(messageId int, isEnd bool, data []byte) error

	// Signals that the other peer has sent you a response chunk.
	OnResponseChunkReceived(messageId int, isEnd bool, data []byte) error

	// Signals that the other peer is pinging you.
	OnPingReceived(messageId int) error

	// Signals that the other peer has responded to your ping.
	OnPingAckReceived(messageId int, latency time.Duration) error

	// Signals that the other peer wishes to cancel an operation. Upon returning
	// from this callback, your send queue must be purged of all response message
	// chunks to this ID, and any as-of-yet unfinished operations must not produce
	// any further response chunks to it afterwards.
	// Note: this signal may arrive for an ID that doesn't exist. This is allowed.
	OnCancelReceived(messageId int) error

	// Signals that the remote peer has canceled this operation.
	// Note: this signal may arrive for an ID that doesn't exist. This is allowed.
	OnCancelAckReceived(messageId int) error

	// Signals successful completion of an operation, with no other data to report.
	OnEmptyResponseReceived(messageId int) error
}

// MessageSender is notified when communication is possible, and when data is
// available to send over your communications channel.
type MessageSender interface {
	// Signals that protocol negotiations are complete, and you may now begin
	// sending messages via the Protocol object.
	// Until you receive this notification, attempts to send messages will fail.
	OnAbleToSend()

	// Signals that there is message data available to send over your
	// communications channel. OnMessageChunkToSend is triggered as you call message
	// sending methods such as Protocol.SendRequest() and Protocol.SendResponse().
	// Higher priority data must be sent before lower priority data.
	OnMessageChunkToSend(priority int, messageId int, chunk []byte) error
}
