package sproto

import (
	"reflect"
	"unsafe"
)

// decodePointer 只在当前同步 Decode 内借用临时结果；不缓存地址或跨调用保留 uintptr。
// wire 与预算仍由通用 parser 校验，这里只负责已确认布局的原生 *int64 写入。
type decodePointer struct {
	value unsafe.Pointer
}

func messageDecodePointer(schema *SprotoType, value reflect.Value) decodePointer {
	if !schema.nativeInt64Pointer || value.Type().Elem() != schema.typ {
		return decodePointer{}
	}
	return decodePointer{value: value.UnsafePointer()}
}

// 每次写入都核对公开字段描述；外来 owner 或变化的形状必须保留反射索引语义。
func (pointer decodePointer) setInteger(schema *SprotoType, field *SprotoField, value *int64) bool {
	if pointer.value == nil || !field.nativeInteger(schema) {
		return false
	}
	// 使用 Go 类型化指针赋值，写屏障由编译器生成，不能用字节复制替代。
	*(**int64)(unsafe.Add(pointer.value, field.encOffset)) = value
	return true
}
