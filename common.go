package streamux

import (
	"math"
)

const priorityOOB = math.MaxInt32

const (
	shiftResponseBit = 1
	shiftId          = 2
)

const protocolVersion = 1

const (
	shiftQuickInitRequest      = 30
	shiftQuickInitAllowed      = 29
	shiftLengthBitsMin         = 24
	shiftLengthBitsMax         = 19
	shiftLengthBitsRecommended = 14
	shiftIdBitsMin             = 10
	shiftIdBitsMax             = 5
)

const (
	maskMin         = 0x0f
	maskMax         = 0x1f
	maskRecommended = 0x1f
)

func minInt(a, b int) int {
	if a > b {
		return b
	}
	return a
}

func maxInt(a, b int) int {
	if b > a {
		return b
	}
	return a
}

func midpointInt(a, b int) int {
	min := a
	max := b
	if min > max {
		min = b
		max = a
	}
	diff := max - min
	result := diff/2 + min
	if diff&1 == 1 {
		result++
	}
	return result
}
