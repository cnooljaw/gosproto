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

// encodeBinding 只在当前调用/数组遍历内传递已验证事实，不保存消息值。
// exact 表示实际 Go 结构类型匹配；同构数组另在入口核对所有字段的 owner/形状。
type encodeBinding struct {
	exact  bool
	native bool
}

// 根值已由 getbase/GetSprotoType 核对，或子值类型已由私有构造信息/反射核对。
func knownEncodeType(schema *SprotoType) encodeBinding {
	return encodeBinding{exact: true, native: schema.nativeInt64Pointer}
}

// trusted 同时要求父类型、字段 owner 和数组/指针形状成立，由调用方从私有构造信息判定。
func bindEncodeChild(field *SprotoField, value reflect.Value, trusted bool) encodeBinding {
	if trusted {
		return knownEncodeType(field.st)
	}
	return checkEncodeChild(field, value)
}

// 公开描述被替换、形状改变或父类型不匹配时，不能继承原先的 Go 布局关系。
func checkEncodeChild(field *SprotoField, value reflect.Value) encodeBinding {
	typ := value.Type()
	if field.Array {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Ptr || typ.Elem() != field.st.typ {
		return encodeBinding{}
	}
	return knownEncodeType(field.st)
}

// 只在结构体数组入口核对一次同构字段，之后仍逐元素读取当前值并执行预算检查。
func (schema *SprotoType) nativeIntegerFields() bool {
	for _, index := range schema.order {
		field := schema.Fields[index]
		if field.Tag >= 0 && !field.nativeInteger(schema) {
			return false
		}
	}
	return true
}

func messageEncodePointer(value reflect.Value, binding encodeBinding) encodePointer {
	if !binding.native {
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
