package protocol

import "sync"

var bytesPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 3)
	},
}

func IntToBytes(n int) []byte {
	buf := bytesPool.Get().([]byte)
	buf[0] = byte((n >> 16) & 0xFF)
	buf[1] = byte((n >> 8) & 0xFF)
	buf[2] = byte(n & 0xFF)

	result := make([]byte, 3)
	copy(result, buf)

	bytesPool.Put(buf)
	return result
}

func BytesToInt(b []byte) int {
	if len(b) >= 3 {
		return int(b[0])<<16 | int(b[1])<<8 | int(b[2])
	}

	result := 0
	for _, v := range b {
		result = result<<8 + int(v)
	}
	return result
}
