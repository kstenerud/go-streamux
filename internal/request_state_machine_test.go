package internal

import (
	// "fmt"
	"testing"
)

func assertBeginRequestDoesCall(t *testing.T, rules *RequestStateMachine) (messageId int) {
	didCall := false
	err := rules.TryBeginRequest(func(id int) {
		messageId = id
		didCall = true
	})
	if err != nil {
		t.Errorf("Begin request failed: %v", err)
	} else if !didCall {
		t.Errorf("Begin request failed: callback was not called")
	}

	return messageId
}

func assertPingDoesCall(t *testing.T, rules *RequestStateMachine) (messageId int) {
	didCall := false
	err := rules.TryPing(func(id int) {
		messageId = id
		didCall = true
	})
	if err != nil {
		t.Errorf("Begin request failed: %v", err)
	} else if !didCall {
		t.Errorf("Begin request failed: callback was not called")
	}

	return messageId
}

func assertSendRequestChunkDoesCall(t *testing.T, rules *RequestStateMachine, messageId int, isTerminated bool) {
	didCall := false
	err := rules.TrySendRequestChunk(messageId, isTerminated, func(id int, terminated bool) {
		didCall = true
	})
	if err != nil {
		t.Errorf("Send request chunk id %v failed: %v", messageId, err)
	} else if !didCall {
		t.Errorf("Send request chunk id %v failed: callback was not called", messageId)
	}
}

func assertSendRequestChunkFails(t *testing.T, rules *RequestStateMachine, messageId int, isTerminated bool) {
	didCall := false
	err := rules.TrySendRequestChunk(messageId, isTerminated, func(id int, terminated bool) {
		didCall = true
	})
	if err == nil {
		t.Errorf("Send request chunk id %v should have failed, but didn't", messageId)
	}
	if didCall {
		t.Errorf("Send request chunk id %v callback should not have been called", messageId)
	}
}

func assertCancelDoesCall(t *testing.T, rules *RequestStateMachine, messageId int) {
	didCall := false
	err := rules.TryCancelRequest(messageId, func(id int) {
		didCall = true
	})
	if err != nil {
		t.Errorf("Cancel request id %v failed: %v", messageId, err)
	} else if !didCall {
		t.Errorf("Cancel request id %v failed: callback was not called", messageId)
	}
}

func assertCancelDoesNotCall(t *testing.T, rules *RequestStateMachine, messageId int) {
	didCall := false
	err := rules.TryCancelRequest(messageId, func(id int) {
		didCall = true
	})
	if err != nil {
		t.Errorf("Cancel request id %v failed: %v", messageId, err)
	} else if didCall {
		t.Errorf("Cancel request id %v was called when it shouldn't have", messageId)
	}
}

func assertCancelFails(t *testing.T, rules *RequestStateMachine, messageId int) {
	err := rules.TryCancelRequest(messageId, func(id int) {
	})
	if err == nil {
		t.Errorf("Cancel request id %v should have failed but didn't", messageId)
	}
}

func assertReceiveResponseChunkDoesCall(t *testing.T, rules *RequestStateMachine, messageId int, isTerminated bool) {
	didCall := false
	err := rules.TryReceiveResponseChunk(messageId, isTerminated, func(id int, terminated bool) {
		didCall = true
	})
	if err != nil {
		t.Errorf("Receive response chunk id %v failed: %v", messageId, err)
	} else if !didCall {
		t.Errorf("Receive response chunk id %v failed: callback was not called", messageId)
	}
}

func assertReceiveResponseChunkDoesNotCall(t *testing.T, rules *RequestStateMachine, messageId int, isTerminated bool) {
	didCall := false
	err := rules.TryReceiveResponseChunk(messageId, isTerminated, func(id int, terminated bool) {
		messageId = id
		didCall = true
	})
	if err != nil {
		t.Errorf("Receive response chunk id %v failed: %v", messageId, err)
	} else if didCall {
		t.Errorf("Receive response chunk id %v callback should not have been called", messageId)
	}
}

func assertReceiveResponseChunkFails(t *testing.T, rules *RequestStateMachine, messageId int, isTerminated bool) {
	didCall := false
	err := rules.TryReceiveResponseChunk(messageId, isTerminated, func(id int, terminated bool) {
		didCall = true
	})
	if err == nil {
		t.Errorf("Receive response chunk id %v should have failed, but didn't", messageId)
	}
	if didCall {
		t.Errorf("Receive response chunk id %v callback should not have been called", messageId)
	}
}

func assertReceiveCancelAckDoesCall(t *testing.T, rules *RequestStateMachine, messageId int) {
	didCall := false
	err := rules.TryReceiveCancelAck(messageId, func(id int) {
		didCall = true
	})
	if err != nil {
		t.Errorf("Receive cancel ack id %v failed: %v", messageId, err)
	} else if !didCall {
		t.Errorf("Receive cancel ack id %v failed: callback was not called", messageId)
	}
}

func assertReceiveCancelAckDoesNotCall(t *testing.T, rules *RequestStateMachine, messageId int) {
	didCall := false
	err := rules.TryReceiveCancelAck(messageId, func(id int) {
		messageId = id
		didCall = true
	})
	if err != nil {
		t.Errorf("Receive cancel ack id %v failed: %v", messageId, err)
	} else if didCall {
		t.Errorf("Receive cancel ack id %v callback should not have been called", messageId)
	}
}

func assertReceiveCancelAckFails(t *testing.T, rules *RequestStateMachine, messageId int) {
	didCall := false
	err := rules.TryReceiveCancelAck(messageId, func(id int) {
		didCall = true
	})
	if err == nil {
		t.Errorf("Receive cancel ack id %v should have failed, but didn't", messageId)
	}
	if didCall {
		t.Errorf("Receive cancel ack id %v callback should not have been called", messageId)
	}
}

// =============================================================================

func TestBeginRequest(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	assertBeginRequestDoesCall(t, rules)
}

func TestPing(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	assertPingDoesCall(t, rules)
}

func TestSendSingleChunkRequest(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertSendRequestChunkDoesCall(t, rules, id, true)
}

func TestSendMultiChunkRequest(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertSendRequestChunkDoesCall(t, rules, id, false)
	assertSendRequestChunkDoesCall(t, rules, id, true)
}

func TestSendNonexistentRequestFails(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := 1
	assertSendRequestChunkFails(t, rules, id, true)
}

func TestSendTerminatedRequestFails(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertSendRequestChunkDoesCall(t, rules, id, true)
	assertSendRequestChunkFails(t, rules, id, false)
}

func TestSendCanceledRequestFails(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertSendRequestChunkDoesCall(t, rules, id, false)
	assertCancelDoesCall(t, rules, id)
	assertSendRequestChunkFails(t, rules, id, false)
}

func TestCancelBeforeSendIgnored(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertCancelDoesNotCall(t, rules, id)
}

func TestCancelBeforeRequestFinish(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertSendRequestChunkDoesCall(t, rules, id, false)
	assertCancelDoesCall(t, rules, id)
}

func TestCancelAfterRequestFinish(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertSendRequestChunkDoesCall(t, rules, id, true)
	assertCancelDoesCall(t, rules, id)
}

func TestCancelDuringResponse(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertSendRequestChunkDoesCall(t, rules, id, true)
	assertReceiveResponseChunkDoesCall(t, rules, id, false)
	assertCancelDoesCall(t, rules, id)
}

func TestCancelAfterResponseIgnored(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertSendRequestChunkDoesCall(t, rules, id, true)
	assertReceiveResponseChunkDoesCall(t, rules, id, true)
	assertCancelDoesNotCall(t, rules, id)
}

func TestDoubleCancelIgnored(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertSendRequestChunkDoesCall(t, rules, id, true)
	assertCancelDoesCall(t, rules, id)
	assertCancelDoesNotCall(t, rules, id)
}

func TestCancelComplex(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id1 := assertBeginRequestDoesCall(t, rules)
	id2 := assertBeginRequestDoesCall(t, rules)
	assertSendRequestChunkDoesCall(t, rules, id1, true)
	assertSendRequestChunkDoesCall(t, rules, id2, true)
	assertCancelDoesCall(t, rules, id1)
	assertReceiveResponseChunkDoesCall(t, rules, id2, true)
	assertReceiveCancelAckDoesCall(t, rules, id1)
	assertCancelDoesNotCall(t, rules, id2)
}

func TestSendReceive(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertSendRequestChunkDoesCall(t, rules, id, true)
	assertReceiveResponseChunkDoesCall(t, rules, id, true)
}

func TestSendReceiveMulti(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertSendRequestChunkDoesCall(t, rules, id, false)
	assertSendRequestChunkDoesCall(t, rules, id, true)
	assertReceiveResponseChunkDoesCall(t, rules, id, false)
	assertReceiveResponseChunkDoesCall(t, rules, id, true)
}

func TestReceiveBeforeSendFinishedFails(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertSendRequestChunkDoesCall(t, rules, id, false)
	assertReceiveResponseChunkFails(t, rules, id, true)
}

func TestReceiveBeforeSendBeginsFails(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := assertBeginRequestDoesCall(t, rules)
	assertReceiveResponseChunkFails(t, rules, id, true)
}

func TestReceiveNonexistentMessageFails(t *testing.T) {
	rules := NewRequestStateMachine(NewIdPool(20))
	id := 1
	assertReceiveResponseChunkFails(t, rules, id, true)
}
