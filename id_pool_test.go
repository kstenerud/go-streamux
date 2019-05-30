package streamux

import (
	"testing"
)

func TestIdPool0BitExhausted(t *testing.T) {
	pool := newIdPool(0)

	assertDoesNotPanic(t, func() {
		pool.AllocateId()
	})

	assertDoesPanic(t, func() {
		pool.AllocateId()
	})
}

func TestIdPool0BitReplace(t *testing.T) {
	pool := newIdPool(0)

	assertDoesNotPanic(t, func() {
		id := pool.AllocateId()
		pool.DeallocateId(id)
		pool.AllocateId()
	})

	assertDoesPanic(t, func() {
		pool.AllocateId()
	})
}

func TestIdPool1BitExhausted(t *testing.T) {
	pool := newIdPool(1)

	assertDoesNotPanic(t, func() {
		pool.AllocateId()
		pool.AllocateId()
	})

	assertDoesPanic(t, func() {
		pool.AllocateId()
	})
}

func TestIdPool1BitReplace(t *testing.T) {
	pool := newIdPool(1)

	assertDoesNotPanic(t, func() {
		id1 := pool.AllocateId()
		id2 := pool.AllocateId()
		pool.DeallocateId(id1)
		pool.AllocateId()
		pool.DeallocateId(id2)
		pool.AllocateId()
	})

	assertDoesPanic(t, func() {
		pool.AllocateId()
	})
}

func TestIdPool2BitExhausted(t *testing.T) {
	pool := newIdPool(2)

	assertDoesNotPanic(t, func() {
		pool.AllocateId()
		pool.AllocateId()
		pool.AllocateId()
		pool.AllocateId()
	})

	assertDoesPanic(t, func() {
		pool.AllocateId()
	})
}

func TestIdPool2BitReplace(t *testing.T) {
	pool := newIdPool(2)

	assertDoesNotPanic(t, func() {
		id1 := pool.AllocateId()
		id2 := pool.AllocateId()
		pool.DeallocateId(id1)
		pool.AllocateId()
		id3 := pool.AllocateId()
		pool.AllocateId()
		pool.DeallocateId(id3)
		id4 := pool.AllocateId()
		pool.DeallocateId(id4)
		pool.AllocateId()
		pool.DeallocateId(id2)
		pool.AllocateId()
	})

	assertDoesPanic(t, func() {
		pool.AllocateId()
	})
}
