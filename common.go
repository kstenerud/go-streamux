package streamux

func useBytes(byteCount int, buffer []byte) (usedPortion []byte, remainingPortion []byte) {
	usedPortion = buffer[:byteCount]
	remainingPortion = buffer[byteCount : len(buffer)-1]
	return usedPortion, remainingPortion
}

func fillBuffer(finalByteCount int, dst []byte, src []byte) (filledBuffer []byte, remainingPortion []byte) {
	byteCount := finalByteCount - len(dst)
	bytesAvailable := len(src)
	if bytesAvailable < byteCount {
		byteCount = bytesAvailable
	}
	usedPortion, remainingPortion := useBytes(byteCount, src)
	filledBuffer = append(dst, usedPortion...)
	return filledBuffer, remainingPortion
}
