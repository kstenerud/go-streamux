package internal

import (
	"testing"
)

func assertAllocateSucceeds(t *testing.T, pool *IdPool) int {
	if id, ok := pool.AllocateId(); ok {
		return id
	} else {
		t.Errorf("ID pool exhausted")
		return 0
	}
}

func assertAllocateFails(t *testing.T, pool *IdPool) int {
	if id, ok := pool.AllocateId(); ok {
		t.Errorf("ID allocation should have failed, but instead returned %v", id)
		return 0
	} else {
		return 0
	}
}

// =============================================================================

func TestIdPool0BitExhausted(t *testing.T) {
	pool := NewIdPool(0)

	assertAllocateSucceeds(t, pool)
	assertAllocateFails(t, pool)
}

func TestIdPool0BitReplace(t *testing.T) {
	pool := NewIdPool(0)

	id := assertAllocateSucceeds(t, pool)
	pool.DeallocateId(id)
	assertAllocateSucceeds(t, pool)
	assertAllocateFails(t, pool)
}

func TestIdPool1BitExhausted(t *testing.T) {
	pool := NewIdPool(1)

	assertAllocateSucceeds(t, pool)
	assertAllocateSucceeds(t, pool)
	assertAllocateFails(t, pool)
}

func TestIdPool1BitReplace(t *testing.T) {
	pool := NewIdPool(1)

	id1 := assertAllocateSucceeds(t, pool)
	id2 := assertAllocateSucceeds(t, pool)
	pool.DeallocateId(id1)
	assertAllocateSucceeds(t, pool)
	pool.DeallocateId(id2)
	assertAllocateSucceeds(t, pool)
	assertAllocateFails(t, pool)
}

func TestIdPool2BitExhausted(t *testing.T) {
	pool := NewIdPool(2)

	assertAllocateSucceeds(t, pool)
	assertAllocateSucceeds(t, pool)
	assertAllocateSucceeds(t, pool)
	assertAllocateSucceeds(t, pool)
	assertAllocateFails(t, pool)
}

func TestIdPool2BitReplace(t *testing.T) {
	pool := NewIdPool(2)

	id1 := assertAllocateSucceeds(t, pool)
	id2 := assertAllocateSucceeds(t, pool)
	pool.DeallocateId(id1)
	assertAllocateSucceeds(t, pool)
	id3 := assertAllocateSucceeds(t, pool)
	assertAllocateSucceeds(t, pool)
	pool.DeallocateId(id3)
	id4 := assertAllocateSucceeds(t, pool)
	pool.DeallocateId(id4)
	assertAllocateSucceeds(t, pool)
	pool.DeallocateId(id2)
	assertAllocateSucceeds(t, pool)
	assertAllocateFails(t, pool)
}
