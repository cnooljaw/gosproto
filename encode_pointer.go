package sproto

import (
	"reflect"
	"unsafe"
)

// encodePointer 仅在当前同步 Encode 内借用 Go 对象；不缓存消息地址，也不将地址转成 uintptr 跨调用保留。
// unsafe 只用于 schema 已验证的原生 *int64 字段读取，不参与 wire 和预算处理。
type encodePointer struct {
	value unsafe.Pointer
}

func messageEncodePointer(schema *SprotoType, value reflect.Value) encodePointer {
	if !schema.nativeInt64Pointer {
		return encodePointer{}
	}
	return encodePointer{value: value.UnsafePointer()}
}

// 字段描述可被替换；只有由本结构体初始化且仍保持原形状的字段能使用 offset。
func (field *SprotoField) nativeInteger(schema *SprotoType) bool {
	return field.nativeInt64Pointer && field.encOwner == schema &&
		field.Wire == WireVarintName && !field.Array
}

func (pointer encodePointer) integer(offset uintptr) (uint64, bool) {
	value := *(**int64)(unsafe.Add(pointer.value, offset))
	if value == nil {
		return 0, false
	}
	return uint64(*value), true
}
