package internal

import (
	"testing"

	"github.com/kstenerud/go-streamux/test"
)

func TestIdPool0BitExhausted(t *testing.T) {
	pool := NewIdPool(0)

	test.AssertDoesNotPanic(t, func() {
		pool.AllocateId()
	})

	test.AssertDoesPanic(t, func() {
		pool.AllocateId()
	})
}

func TestIdPool0BitReplace(t *testing.T) {
	pool := NewIdPool(0)

	test.AssertDoesNotPanic(t, func() {
		id := pool.AllocateId()
		pool.DeallocateId(id)
		pool.AllocateId()
	})

	test.AssertDoesPanic(t, func() {
		pool.AllocateId()
	})
}

func TestIdPool1BitExhausted(t *testing.T) {
	pool := NewIdPool(1)

	test.AssertDoesNotPanic(t, func() {
		pool.AllocateId()
		pool.AllocateId()
	})

	test.AssertDoesPanic(t, func() {
		pool.AllocateId()
	})
}

func TestIdPool1BitReplace(t *testing.T) {
	pool := NewIdPool(1)

	test.AssertDoesNotPanic(t, func() {
		id1 := pool.AllocateId()
		id2 := pool.AllocateId()
		pool.DeallocateId(id1)
		pool.AllocateId()
		pool.DeallocateId(id2)
		pool.AllocateId()
	})

	test.AssertDoesPanic(t, func() {
		pool.AllocateId()
	})
}

func TestIdPool2BitExhausted(t *testing.T) {
	pool := NewIdPool(2)

	test.AssertDoesNotPanic(t, func() {
		pool.AllocateId()
		pool.AllocateId()
		pool.AllocateId()
		pool.AllocateId()
	})

	test.AssertDoesPanic(t, func() {
		pool.AllocateId()
	})
}

func TestIdPool2BitReplace(t *testing.T) {
	pool := NewIdPool(2)

	test.AssertDoesNotPanic(t, func() {
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

	test.AssertDoesPanic(t, func() {
		pool.AllocateId()
	})
}
