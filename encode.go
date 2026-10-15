package sproto

import (
	"math"
	"reflect"
)

const (
	EncodeBufferSize = 4096
	MaxEmbeddedInt   = 0x7fff - 1
	MaxInt32         = 0x7fffffff
	MinInt32         = -0x80000000
)

// little endian
// put int into []byte
func writeUint16(buf []byte, v uint16) {
	buf[0] = uint8(v & 0xff)
	buf[1] = uint8(v >> 8)
}

func writeUint32(buf []byte, v uint32) {
	buf[0] = uint8(v & 0xff)
	buf[1] = uint8((v >> 8) & 0xff)
	buf[2] = uint8((v >> 16) & 0xff)
	buf[3] = uint8((v >> 24) & 0xff)
}

func writeUint64(buf []byte, v uint64) {
	buf[0] = uint8(v & 0xff)
	buf[1] = uint8((v >> 8) & 0xff)
	buf[2] = uint8((v >> 16) & 0xff)
	buf[3] = uint8((v >> 24) & 0xff)
	buf[4] = uint8((v >> 32) & 0xff)
	buf[5] = uint8((v >> 40) & 0xff)
	buf[6] = uint8((v >> 48) & 0xff)
	buf[7] = uint8((v >> 56) & 0xff)
}

func extractInt(v reflect.Value) (n uint64, sz int) {
	switch v.Kind() {
	case reflect.Int8, reflect.Int16, reflect.Int32:
		n = uint64(v.Int())
		sz = 4
	case reflect.Int, reflect.Int64:
		n1 := v.Int()
		n = uint64(n1)
		if n1 >= -0x80000000 && n1 <= 0x7fffffff {
			sz = 4
		} else {
			sz = 8
		}
	case reflect.Uint8, reflect.Uint16:
		n = v.Uint()
		sz = 4
	default: //case reflect.Uint32, reflect.Uint64, reflect.Uint:
		n = v.Uint()
		if n <= MaxInt32 {
			sz = 4
		} else {
			sz = 8
		}
	}
	return
}

// inlineHeader 只读取已经过预算和类型检查的字段，不分配中间结果。
func inlineHeader(field *SprotoField, value reflect.Value) (uint16, bool) {
	if field.Array {
		return 0, false
	}
	if value.Kind() == reflect.Ptr && field.Wire != WireStructName {
		value = value.Elem()
	}
	switch field.Wire {
	case WireBooleanName:
		if value.Bool() {
			return 4, true
		}
		return 2, true
	case WireVarintName:
		bits, _ := extractInt(value)
		if bits <= MaxEmbeddedInt {
			return uint16((bits + 1) * 2), true
		}
	}
	return 0, false
}

func omittedField(value reflect.Value) bool {
	return (value.Kind() == reflect.Ptr || value.Kind() == reflect.Slice) && value.IsNil()
}

func skipTag(tag, nextTag int) uint16 {
	if nextTag > tag+1 {
		return uint16((nextTag-tag-2)*2 + 1)
	}
	return 0
}

// encodeMessageInto 将子结构直接写入同一个已验证尺寸的输出区间。
func encodeMessageInto(schema *SprotoType, value reflect.Value, output []byte) int {
	if value.IsNil() {
		writeUint16(output, 0)
		return 2
	}
	count, previous := 0, -1
	for _, index := range schema.order {
		field := schema.Fields[index]
		if field.Tag < 0 {
			continue
		}
		current := value.Elem().FieldByIndex(field.index)
		if omittedField(current) {
			continue
		}
		if skip := skipTag(previous, field.Tag); skip > 0 {
			writeUint16(output[2+count*2:], skip)
			count++
		}
		header, _ := inlineHeader(field, current)
		writeUint16(output[2+count*2:], header)
		count++
		previous = field.Tag
	}
	writeUint16(output, uint16(count))
	offset := 2 + count*2
	for _, index := range schema.order {
		field := schema.Fields[index]
		if field.Tag < 0 {
			continue
		}
		current := value.Elem().FieldByIndex(field.index)
		if omittedField(current) {
			continue
		}
		if _, inline := inlineHeader(field, current); inline {
			continue
		}
		length := encodeFieldInto(field, current, output[offset+4:])
		writeUint32(output[offset:], uint32(length))
		offset += 4 + length
	}
	return offset
}

func encodeFieldInto(field *SprotoField, value reflect.Value, output []byte) int {
	if field.Array {
		if value.Len() == 0 {
			return 0
		}
		offset := 0
		switch field.Wire {
		case WireVarintName:
			width := 4
			for index := 0; index < value.Len(); index++ {
				bits, currentWidth := extractInt(value.Index(index))
				if width == 4 && currentWidth == 8 {
					// 只在首个宽值出现时升级一次；从后向前扩展，避免覆盖未搬移的 word。
					for previous := index - 1; previous >= 0; previous-- {
						signed := int64(int32(readUint32(output[1+previous*4:])))
						writeUint64(output[1+previous*8:], uint64(signed))
					}
					width = 8
				}
				if width == 4 {
					writeUint32(output[1+index*width:], uint32(bits))
				} else {
					writeUint64(output[1+index*width:], bits)
				}
			}
			output[0] = byte(width)
			offset = 1 + width*value.Len()
		case WireDoubleName:
			output[0] = 8
			offset = 1
			for index := 0; index < value.Len(); index++ {
				writeUint64(output[offset:], math.Float64bits(value.Index(index).Float()))
				offset += 8
			}
		case WireBooleanName:
			for index := 0; index < value.Len(); index++ {
				if value.Index(index).Bool() {
					output[index] = 1
				} else {
					output[index] = 0
				}
			}
			offset = value.Len()
		case WireBytesName:
			for index := 0; index < value.Len(); index++ {
				text := value.Index(index).String()
				writeUint32(output[offset:], uint32(len(text)))
				offset += 4 + copy(output[offset+4:], text)
			}
		case WireStructName:
			for index := 0; index < value.Len(); index++ {
				length := encodeMessageInto(field.st, value.Index(index), output[offset+4:])
				writeUint32(output[offset:], uint32(length))
				offset += 4 + length
			}
		}
		return offset
	}
	if field.Wire == WireStructName {
		return encodeMessageInto(field.st, value, output)
	}
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	switch field.Wire {
	case WireVarintName:
		bits, width := extractInt(value)
		if width == 4 {
			writeUint32(output, uint32(bits))
		} else {
			writeUint64(output, bits)
		}
		return width
	case WireDoubleName:
		writeUint64(output, math.Float64bits(value.Float()))
		return 8
	case WireBytesName:
		if value.Kind() == reflect.String {
			return copy(output, value.String())
		}
		return copy(output, value.Bytes())
	}
	return 0
}

// Encode 使用默认有限预算编码结构体。
func Encode(sp interface{}) ([]byte, error) {
	return EncodeWithLimits(sp, Limits{})
}

// MustEncode 在输入不能编码时 panic，适用于静态且已确认合法的数据。
func MustEncode(sp interface{}) []byte {
	encoded, err := Encode(sp)
	if err != nil {
		panic(err)
	}
	return encoded
}
