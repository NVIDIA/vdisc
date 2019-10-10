package safecast

import (
	"math"
)

func Int8ToUint(v int8) uint {
	if v < 0 {
		panic("out of bounds")
	}
	return uint(v)
}

func Int8ToUint8(v int8) uint8 {
	if v < 0 {
		panic("out of bounds")
	}
	return uint8(v)
}

func Int8ToUint16(v int8) uint16 {
	if v < 0 {
		panic("out of bounds")
	}
	return uint16(v)
}

func Int8ToUint32(v int8) uint32 {
	if v < 0 {
		panic("out of bounds")
	}
	return uint32(v)
}

func Int8ToUint64(v int8) uint64 {
	if v < 0 {
		panic("out of bounds")
	}
	return uint64(v)
}

func Int16ToInt8(v int16) int8 {
	r := int8(v)
	if int16(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int16ToUint(v int16) uint {
	if v < 0 {
		panic("out of bounds")
	}
	return uint(v)
}

func Int16ToUint8(v int16) uint8 {
	if v < 0 {
		panic("out of bounds")
	}
	r := uint8(v)
	if int16(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int16ToUint16(v int16) uint16 {
	if v < 0 {
		panic("out of bounds")
	}
	r := uint16(v)
	if int16(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int16ToUint32(v int16) uint32 {
	if v < 0 {
		panic("out of bounds")
	}
	return uint32(v)
}

func Int16ToUint64(v int16) uint64 {
	if v < 0 {
		panic("out of bounds")
	}
	return uint64(v)
}

func Int32ToInt8(v int32) int8 {
	r := int8(v)
	if int32(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int32ToInt16(v int32) int16 {
	r := int16(v)
	if int32(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int32ToUint(v int32) uint {
	if v < 0 {
		panic("out of bounds")
	}
	return uint(v)
}

func Int32ToUint8(v int32) uint8 {
	if v < 0 {
		panic("out of bounds")
	}
	r := uint8(v)
	if int32(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int32ToUint16(v int32) uint16 {
	if v < 0 {
		panic("out of bounds")
	}
	r := uint16(v)
	if int32(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int32ToUint32(v int32) uint32 {
	if v < 0 {
		panic("out of bounds")
	}
	return uint32(v)
}

func Int32ToUint64(v int32) uint64 {
	if v < 0 {
		panic("out of bounds")
	}
	return uint64(v)
}

func Int64ToInt(v int64) int {
	r := int(v)
	if int64(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int64ToInt8(v int64) int8 {
	r := int8(v)
	if int64(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int64ToInt16(v int64) int16 {
	r := int16(v)
	if int64(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int64ToInt32(v int64) int32 {
	r := int32(v)
	if int64(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int64ToUint(v int64) uint {
	if v < 0 {
		panic("out of bounds")
	}
	r := uint(v)
	if int64(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int64ToUint8(v int64) uint8 {
	if v < 0 {
		panic("out of bounds")
	}
	r := uint8(v)
	if int64(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int64ToUint16(v int64) uint16 {
	if v < 0 {
		panic("out of bounds")
	}
	r := uint16(v)
	if int64(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int64ToUint32(v int64) uint32 {
	if v < 0 {
		panic("out of bounds")
	}
	r := uint32(v)
	if int64(r) != v {
		panic("out of bounds")
	}
	return r
}

func Int64ToUint64(v int64) uint64 {
	if v < 0 {
		panic("out of bounds")
	}
	return uint64(v)
}

func IntToInt8(v int) int8 {
	r := int8(v)
	if int(r) != v {
		panic("out of bounds")
	}
	return r
}

func IntToInt16(v int) int16 {
	r := int16(v)
	if int(r) != v {
		panic("out of bounds")
	}
	return r
}

func IntToInt32(v int) int32 {
	r := int32(v)
	if int(r) != v {
		panic("out of bounds")
	}
	return r
}

func IntToUint8(v int) uint8 {
	if v < 0 {
		panic("out of bounds")
	}
	r := uint8(v)
	if int(r) != v {
		panic("out of bounds")
	}
	return r
}

func IntToUint16(v int) uint16 {
	if v < 0 {
		panic("out of bounds")
	}
	r := uint16(v)
	if int(r) != v {
		panic("out of bounds")
	}
	return r
}

func IntToUint32(v int) uint32 {
	if v < 0 {
		panic("out of bounds")
	}
	r := uint32(v)
	if int(r) != v {
		panic("out of bounds")
	}
	return r
}

func IntToUint64(v int) uint64 {
	if v < 0 {
		panic("out of bounds")
	}
	return uint64(v)
}

func Uint8ToInt8(v uint8) int8 {
	if v > math.MaxInt8 {
		panic("out of bounds")
	}
	return int8(v)
}

func Uint16ToInt8(v uint16) int8 {
	if v > math.MaxInt8 {
		panic("out of bounds")
	}
	return int8(v)
}

func Uint16ToInt16(v uint16) int16 {
	if v > math.MaxInt16 {
		panic("out of bounds")
	}
	return int16(v)
}

func Uint16ToUint8(v uint16) uint8 {
	if v > math.MaxUint8 {
		panic("out of bounds")
	}
	return uint8(v)
}

func Uint32ToInt(v uint32) int {
	r := int(v)
	if r < 0 {
		panic("out of bounds")
	}
	return r
}

func Uint32ToInt8(v uint32) int8 {
	if v > math.MaxInt8 {
		panic("out of bounds")
	}
	return int8(v)
}

func Uint32ToInt16(v uint32) int16 {
	if v > math.MaxInt16 {
		panic("out of bounds")
	}
	return int16(v)
}

func Uint32ToInt32(v uint32) int32 {
	if v > math.MaxInt32 {
		panic("out of bounds")
	}
	return int32(v)
}

func Uint32ToUint8(v uint32) uint8 {
	if v > math.MaxUint8 {
		panic("out of bounds")
	}
	return uint8(v)
}

func Uint32ToUint16(v uint32) uint16 {
	if v > math.MaxUint16 {
		panic("out of bounds")
	}
	return uint16(v)
}

func Uint64ToInt(v uint64) int {
	r := int(v)
	if r < 0 {
		panic("out of bounds")
	} else if uint64(r) != v {
		panic("out of bounds")
	}
	return r
}

func Uint64ToInt8(v uint64) int8 {
	if v > math.MaxInt8 {
		panic("out of bounds")
	}
	return int8(v)
}

func Uint64ToInt16(v uint64) int16 {
	if v > math.MaxInt16 {
		panic("out of bounds")
	}
	return int16(v)
}

func Uint64ToInt32(v uint64) int32 {
	if v > math.MaxInt32 {
		panic("out of bounds")
	}
	return int32(v)
}

func Uint64ToInt64(v uint64) int64 {
	if v > math.MaxInt64 {
		panic("out of bounds")
	}
	return int64(v)
}

func Uint64ToUint(v uint64) uint {
	r := uint(v)
	if uint64(r) != v {
		panic("out of bounds")
	}
	return r
}

func Uint64ToUint8(v uint64) uint8 {
	if v > math.MaxUint8 {
		panic("out of bounds")
	}
	return uint8(v)
}

func Uint64ToUint16(v uint64) uint16 {
	if v > math.MaxUint16 {
		panic("out of bounds")
	}
	return uint16(v)
}

func Uint64ToUint32(v uint64) uint32 {
	if v > math.MaxUint32 {
		panic("out of bounds")
	}
	return uint32(v)
}

func UintToInt8(v uint) int8 {
	if v > math.MaxInt8 {
		panic("out of bounds")
	}
	return int8(v)
}

func UintToInt16(v uint) int16 {
	if v > math.MaxInt16 {
		panic("out of bounds")
	}
	return int16(v)
}

func UintToInt32(v uint) int32 {
	if v > math.MaxInt32 {
		panic("out of bounds")
	}
	return int32(v)
}

func UintToInt64(v uint) int64 {
	if v > math.MaxInt64 {
		panic("out of bounds")
	}
	return int64(v)
}

func UintToUint8(v uint) uint8 {
	if v > math.MaxUint8 {
		panic("out of bounds")
	}
	return uint8(v)
}

func UintToUint16(v uint) uint16 {
	if v > math.MaxUint16 {
		panic("out of bounds")
	}
	return uint16(v)
}

func UintToUint32(v uint) uint32 {
	if v > math.MaxUint32 {
		panic("out of bounds")
	}
	return uint32(v)
}
