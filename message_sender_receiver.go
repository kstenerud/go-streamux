package streamux

import "time"

// Go disallows circular imports, which causes a problem: Either your interfaces
// can be used in subpackages, OR your interfaces are visible from the top level
// for users of your library, but not both.
//
// To work around that, we do an ugly hack: copy the interface code from package
// "common" to the top level.

// MessageReceiver receives complete messages or message chunks from the remote peer.
type MessageReceiver interface {
	OnRequestChunkReceived(messageId int, isEnd bool, data []byte) error
	OnResponseChunkReceived(messageId int, isEnd bool, data []byte) error
	OnPingReceived(messageId int) error
	OnPingAckReceived(messageId int, latency time.Duration) error
	OnCancelReceived(messageId int) error
	OnCancelAckReceived(messageId int) error
	OnEmptyResponseReceived(id int) error
}

// MessageSender is notified when data is available to send over your
// communications channel.
type MessageSender interface {
	// Signals that you may now begin sending messages via the Protocol object.
	// Until you receive this notification, attempts to send messages will fail.
	OnAbleToSend()

	// Signals that there is message data available to send over your
	// communications channel. Higher priority data should be sent before
	// lower priority data.
	OnMessageChunkToSend(priority int, messageId int, chunk []byte) error
}
