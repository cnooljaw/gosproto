package sproto

import (
	"encoding/binary"
	"math"
	"reflect"
)

type Tag struct {
	Tag uint16
	Val *uint16
}

func readUint16(buffer []byte) uint16 {
	return binary.LittleEndian.Uint16(buffer)
}

func readUint32(buffer []byte) uint32 {
	return binary.LittleEndian.Uint32(buffer)
}

func readUint64(buffer []byte) uint64 {
	return binary.LittleEndian.Uint64(buffer)
}

func readChunk(chunk []byte) (int, []byte, error) {
	if len(chunk) < 4 {
		return 0, nil, ErrDecode
	}
	size := uint64(readUint32(chunk))
	if size > uint64(len(chunk)-4) {
		return 0, nil, ErrDecode
	}
	end := 4 + int(size)
	return end, chunk[4:end], nil
}

func targetValue(value reflect.Value) reflect.Value {
	if value.Kind() == reflect.Ptr {
		value.Set(reflect.New(value.Type().Elem()))
		return value.Elem()
	}
	return value
}

func setInteger(value reflect.Value, bits uint64) error {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		signed := int64(bits)
		if value.OverflowInt(signed) {
			return ErrDecode
		}
		value.SetInt(signed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if int64(bits) < 0 || value.OverflowUint(bits) {
			return ErrDecode
		}
		value.SetUint(bits)
	default:
		return ErrDecode
	}
	return nil
}

func decodeBool(inline uint16, hasInline bool, _ []byte, _ *SprotoField, value reflect.Value, _ *decodeBudget) error {
	if !hasInline {
		return ErrDecode
	}
	targetValue(value).SetBool(inline != 0)
	return nil
}

func decodeInt(inline uint16, hasInline bool, data []byte, _ *SprotoField, value reflect.Value, _ *decodeBudget) error {
	var bits uint64
	if hasInline {
		bits = uint64(inline)
	} else {
		switch len(data) {
		case 4:
			bits = uint64(int64(int32(readUint32(data))))
		case 8:
			bits = readUint64(data)
		default:
			return ErrDecode
		}
	}
	return setInteger(targetValue(value), bits)
}

func decodeDouble(inline uint16, hasInline bool, data []byte, _ *SprotoField, value reflect.Value, _ *decodeBudget) error {
	var bits uint64
	if hasInline {
		bits = uint64(inline)
	} else {
		switch len(data) {
		case 4:
			bits = uint64(int64(int32(readUint32(data))))
		case 8:
			bits = readUint64(data)
		default:
			return ErrDecode
		}
	}
	targetValue(value).SetFloat(math.Float64frombits(bits))
	return nil
}

func decodeString(_ uint16, _ bool, data []byte, _ *SprotoField, value reflect.Value, _ *decodeBudget) error {
	targetValue(value).SetString(string(data))
	return nil
}

func decodeBytes(_ uint16, _ bool, data []byte, _ *SprotoField, value reflect.Value, _ *decodeBudget) error {
	value.SetBytes(append([]byte{}, data...))
	return nil
}

func decodeBoolSlice(_ uint16, _ bool, data []byte, _ *SprotoField, value reflect.Value, budget *decodeBudget) error {
	if err := budget.add(len(data)); err != nil {
		return err
	}
	result := reflect.MakeSlice(value.Type(), len(data), len(data))
	for i, b := range data {
		result.Index(i).SetBool(b != 0)
	}
	value.Set(result)
	return nil
}

func decodeIntSlice(_ uint16, _ bool, data []byte, field *SprotoField, value reflect.Value, budget *decodeBudget) error {
	// 官方接受空内容或仅有宽度前缀的空数组；非空数组必须先校验宽度。
	if len(data) <= 1 {
		value.Set(reflect.MakeSlice(value.Type(), 0, 0))
		return nil
	}
	width := int(data[0])
	data = data[1:]
	if (width != 4 && width != 8) || len(data)%width != 0 {
		return ErrDecode
	}
	count := len(data) / width
	if err := budget.add(count); err != nil {
		return err
	}
	result := reflect.MakeSlice(value.Type(), count, count)
	for i := 0; i < count; i++ {
		var bits uint64
		if width == 4 {
			bits = uint64(int64(int32(readUint32(data[i*width:]))))
		} else {
			bits = readUint64(data[i*width:])
		}
		if field.Wire == WireDoubleName {
			result.Index(i).SetFloat(math.Float64frombits(bits))
		} else if err := setInteger(result.Index(i), bits); err != nil {
			return err
		}
	}
	value.Set(result)
	return nil
}

// countChunks 在任何数组分配前检查完整边界和元素预算。
func countChunks(data []byte, budget *decodeBudget) (int, error) {
	count := 0
	for len(data) > 0 {
		used, _, err := readChunk(data)
		if err != nil {
			return 0, err
		}
		if err := budget.add(1); err != nil {
			return 0, err
		}
		count++
		data = data[used:]
	}
	return count, nil
}

func decodeStringSlice(_ uint16, _ bool, data []byte, _ *SprotoField, value reflect.Value, budget *decodeBudget) error {
	count, err := countChunks(data, budget)
	if err != nil {
		return err
	}
	result := reflect.MakeSlice(value.Type(), count, count)
	for index := 0; index < count; index++ {
		used, content, err := readChunk(data)
		if err != nil {
			return err
		}
		result.Index(index).SetString(string(content))
		data = data[used:]
	}
	value.Set(result)
	return nil
}

func decodeStruct(_ uint16, _ bool, data []byte, field *SprotoField, value reflect.Value, budget *decodeBudget) error {
	result := reflect.New(value.Type().Elem())
	used, err := decodeMessage(data, field.st, result, budget)
	if err != nil {
		return err
	}
	if used != len(data) {
		return ErrDecode
	}
	value.Set(result)
	return nil
}

func decodeStructSlice(_ uint16, _ bool, data []byte, field *SprotoField, value reflect.Value, budget *decodeBudget) error {
	count, err := countChunks(data, budget)
	if err != nil {
		return err
	}
	result := reflect.MakeSlice(value.Type(), count, count)
	for index := 0; index < count; index++ {
		used, content, err := readChunk(data)
		if err != nil {
			return err
		}
		current := reflect.New(value.Type().Elem().Elem())
		consumed, err := decodeMessage(content, field.st, current, budget)
		if err != nil {
			return err
		}
		if consumed != len(content) {
			return ErrDecode
		}
		result.Index(index).Set(current)
		data = data[used:]
	}
	value.Set(result)
	return nil
}

func decodeMessage(chunk []byte, schema *SprotoType, value reflect.Value, budget *decodeBudget) (int, error) {
	budget.depth++
	defer func() { budget.depth-- }()
	if budget.depth > budget.limits.MaxDepth {
		return 0, ErrLimit
	}
	if len(chunk) < 2 {
		return 0, ErrDecode
	}
	count := int(readUint16(chunk))
	total := 2 + count*2
	if total > len(chunk) {
		return 0, ErrDecode
	}
	if err := budget.add(count); err != nil {
		return 0, err
	}
	tag := 0
	for index := 0; index < count; index++ {
		entry := int(readUint16(chunk[2+index*2:]))
		if entry%2 == 1 {
			tag += (entry + 1) / 2
			continue
		}
		if tag > 65535 {
			return 0, ErrDecode
		}
		hasInline := entry != 0
		inline := uint16(entry/2 - 1)
		var content []byte
		if !hasInline {
			used, data, err := readChunk(chunk[total:])
			if err != nil {
				return 0, err
			}
			total += used
			content = data
		}
		field := schema.FieldByTag(tag)
		tag++
		if field == nil {
			continue
		}
		if hasInline && (field.Array || field.Wire == WireBytesName || field.Wire == WireStructName) {
			return 0, ErrDecode
		}
		if err := field.dec(inline, hasInline, content, field, value.Elem().FieldByIndex(field.index), budget); err != nil {
			return 0, err
		}
	}
	return total, nil
}

// Decode 解码一个结构体，使用默认有限预算，返回已消费的字节数。
func Decode(data []byte, message interface{}) (int, error) {
	return DecodeWithLimits(data, message, Limits{})
}

// DecodeWithLimits 在递归与分配前检查预算；失败时保留调用方原值。
func DecodeWithLimits(data []byte, message interface{}, limits Limits) (int, error) {
	limits, err := normalizeLimits(limits)
	if err != nil {
		return 0, err
	}
	if len(data) > limits.MaxBytes {
		return 0, ErrLimit
	}
	typ, value, err := getbase(message)
	if err != nil {
		return 0, err
	}
	schema, err := GetSprotoType(typ.Elem())
	if err != nil {
		return 0, err
	}
	result := reflect.New(typ.Elem())
	used, err := decodeMessage(data, schema, result, &decodeBudget{limits: limits})
	if err != nil {
		return 0, err
	}
	value.Elem().Set(result.Elem())
	return used, nil
}

func MustDecode(data []byte, message interface{}) int {
	used, err := Decode(data, message)
	if err != nil {
		panic(err)
	}
	return used
}
