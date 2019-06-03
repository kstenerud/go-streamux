package internal

import (
	"fmt"
	"sync"
)

type RequestStateMachine struct {
	idPool   *IdPool
	requests map[int]requestState
	mutex    sync.Mutex
}

// API

func NewRequestStateMachine(idPool *IdPool) *RequestStateMachine {
	this := new(RequestStateMachine)
	this.Init(idPool)
	return this
}

func (this *RequestStateMachine) Init(IdPool *IdPool) {
	this.idPool = IdPool
	this.requests = make(map[int]requestState)
}

func (this *RequestStateMachine) TryPing(f func(id int)) error {
	this.mutex.Lock()
	id, ok := this.idPool.AllocateId()
	if ok {
		this.requests[id] = requestStateAllocated
	}
	this.mutex.Unlock()

	if !ok {
		return fmt.Errorf("Could not allocate a new message ID. Please initialize with more ID bits")
	}

	f(id)

	this.mutex.Lock()
	this.idPool.DeallocateId(id)
	this.mutex.Unlock()

	return nil
}

func (this *RequestStateMachine) TryBeginRequest(f func(id int)) error {
	this.mutex.Lock()
	id, ok := this.idPool.AllocateId()
	if ok {
		this.requests[id] = requestStateAllocated
	}
	this.mutex.Unlock()

	if !ok {
		return fmt.Errorf("Could not allocate a new message ID. Please initialize with more ID bits")
	}

	f(id)
	return nil
}

func (this *RequestStateMachine) TrySendRequestChunk(id int, isTerminated bool, f func(id int, isTerminated bool)) error {
	this.mutex.Lock()
	state := this.getRequestState(id)
	if state == requestStateAllocated || state == requestStateSending {
		if isTerminated {
			this.requests[id] = requestStateAwaitingResponse
		} else {
			this.requests[id] = requestStateSending
		}
	}
	this.mutex.Unlock()

	// fmt.Printf("### RSM %p: Send request chunk id %v, term %v, state %v -> %v\n", this, id, isTerminated, state, this.requests[id])
	switch state {
	default:
		return fmt.Errorf("Request %v is in an unhandled state (%v)", id, state)
	case requestStateDeallocated:
		return fmt.Errorf("Cannot send request chunk: ID %v is not allocated", id)
	case requestStateAwaitingResponse, requestStateReceivingResponse:
		return fmt.Errorf("Cannot send request chunk: Request %v has already been terminated", id)
	case requestStateAwaitingCancelAck:
		return fmt.Errorf("Cannot send request chunk: Request %v has been canceled", id)
	case requestStateAllocated, requestStateSending:
		f(id, isTerminated)
	}
	return nil
}

func (this *RequestStateMachine) TryCancelRequest(id int, f func(id int)) error {
	this.mutex.Lock()
	state := this.getRequestState(id)
	if state == requestStateSending ||
		state == requestStateAwaitingResponse ||
		state == requestStateReceivingResponse {

		this.requests[id] = requestStateAwaitingCancelAck
	}
	this.mutex.Unlock()

	// fmt.Printf("### Cancel request %v, State = %v\n", id, state)
	switch state {
	default:
		return fmt.Errorf("Request %v is in an unhandled state (%v)", id, state)
	case requestStateDeallocated:
		// Ignore. There may be a race condition where the request was deallocated.
	case requestStateAllocated:
		// Message hasn't been sent yet, so nothing to do.
	case requestStateAwaitingCancelAck:
		// We've already requested a cancel, so nothing to do.
	case requestStateSending, requestStateAwaitingResponse, requestStateReceivingResponse:
		f(id)
	}
	return nil
}

func (this *RequestStateMachine) TryReceiveResponseChunk(id int, isTerminated bool, f func(id int, isTerminated bool)) error {
	this.mutex.Lock()
	state := this.getRequestState(id)
	if state == requestStateAwaitingResponse {
		this.requests[id] = requestStateReceivingResponse
	}
	if (state == requestStateAwaitingResponse || state == requestStateReceivingResponse) && isTerminated {
		this.removeId(id)
	}
	this.mutex.Unlock()

	// fmt.Printf("### RSM %p: Receive chunk id %v, term %v, state %v -> %v\n", this, id, isTerminated, state, this.requests[id])
	switch state {
	default:
		return fmt.Errorf("Request %v is in an unhandled state (%v)", id, state)
	case requestStateDeallocated:
		return fmt.Errorf("Cannot receive response %v: No such message", id)
	case requestStateAllocated:
		return fmt.Errorf("Cannot receive response %v: Message has not been sent yet", id)
	case requestStateSending:
		return fmt.Errorf("Cannot receive response %v: Message has not been completely sent", id)
	case requestStateAwaitingCancelAck:
		// Ignore
		// fmt.Printf("########## Awaiting cancel ack\n")
	case requestStateAwaitingResponse, requestStateReceivingResponse:
		f(id, isTerminated)
	}
	return nil
}

func (this *RequestStateMachine) TryReceiveCancelAck(id int, f func(id int)) error {
	this.mutex.Lock()
	state := this.getRequestState(id)
	if state == requestStateAwaitingCancelAck {
		this.removeId(id)
	}
	this.mutex.Unlock()

	switch state {
	default:
		return fmt.Errorf("Request %v is in an unhandled state (%v)", id, state)
	case requestStateDeallocated, requestStateAllocated, requestStateSending,
		requestStateAwaitingResponse, requestStateReceivingResponse:
		// Shouldn't happen, but no harm done.
	case requestStateAwaitingCancelAck:
		f(id)
	}
	return nil
}

// Internal

type requestState int

const (
	requestStateDeallocated requestState = iota
	requestStateAllocated
	requestStateSending
	requestStateAwaitingResponse
	requestStateReceivingResponse
	requestStateAwaitingCancelAck
)

func (this *RequestStateMachine) getRequestState(id int) requestState {
	if state, ok := this.requests[id]; ok {
		return state
	}
	return requestStateDeallocated
}

func (this *RequestStateMachine) removeId(id int) {
	delete(this.requests, id)
	this.idPool.DeallocateId(id)
}
