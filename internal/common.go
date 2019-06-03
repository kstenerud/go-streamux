package internal

type InternalMessageSender interface {
	OnRequestChunkToSend(priority int, messageId int, isEnd bool, chunk []byte) error
	OnResponseChunkToSend(priority int, messageId int, isEnd bool, chunk []byte) error
}

type InternalMessageReceiver interface {
	OnRequestChunkReceived(messageId int, isEnd bool, data []byte) error
	OnResponseChunkReceived(messageId int, isEnd bool, data []byte) error
	OnZeroLengthMessageReceived(messageId int, messageType MessageType) error
}

type MessageType int

const (
	MessageTypeRequest MessageType = iota
	MessageTypeResponse
	MessageTypeCancel
	MessageTypeCancelAck
	MessageTypeRequestEmptyTermination
	MessageTypeEmptyResponse
)
