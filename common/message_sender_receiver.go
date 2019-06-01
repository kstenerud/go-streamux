package common

import "time"

type MessageReceiver interface {
	OnRequestChunkReceived(messageId int, isEnd bool, data []byte) error
	OnResponseChunkReceived(messageId int, isEnd bool, data []byte) error
	OnPingReceived(messageId int) error
	OnPingAckReceived(messageId int, latency time.Duration) error
	OnCancelReceived(messageId int) error
	OnCancelAckReceived(messageId int) error
	OnEmptyResponseReceived(id int) error
}

type MessageSender interface {
	OnAbleToSend()
	OnMessageChunkToSend(priority int, chunk []byte) error
}
