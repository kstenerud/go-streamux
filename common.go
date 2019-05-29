package streamux

const (
	shiftResponseBit = 1
	shiftId          = 2
)

func useBytes(byteCount int, buffer []byte) (usedPortion []byte, remainingPortion []byte) {
	byteCount = getMaxByteCountAvailable(buffer, byteCount)
	usedPortion = buffer[:byteCount]
	remainingPortion = buffer[byteCount:len(buffer)]
	return usedPortion, remainingPortion
}

func getMaxByteCountAvailable(buffer []byte, hardMaximum int) int {
	byteCount := len(buffer)
	if byteCount > hardMaximum {
		return hardMaximum
	}
	return byteCount
}

func fillBuffer(finalByteCount int, dst []byte, src []byte) (filledBuffer []byte, remainingPortion []byte) {
	byteCount := getMaxByteCountAvailable(src, finalByteCount-len(dst))
	usedPortion, remainingPortion := useBytes(byteCount, src)
	filledBuffer = append(dst, usedPortion...)
	return filledBuffer, remainingPortion
}
