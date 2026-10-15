package sproto

import (
	"encoding/binary"
	"math"
	"reflect"
)

const (
	EncodeBufferSize = 4096
	MaxEmbeddedInt   = 0x7fff - 1
	MaxInt32         = 0x7fffffff
	MinInt32         = -0x80000000
)

// encodeLayout 只属于一次 Encode 调用，记录每个结构体的头项数和非空整数数组宽度。
// 小消息使用栈内记录；额外记录的数量由同一次预检的字段/元素预算约束。
type encodeLayout struct {
	local  [32]int
	extra  []int
	count  int
	cursor int
}

func (layout *encodeLayout) add(value int) int {
	index := layout.count
	if index < len(layout.local) {
		layout.local[index] = value
	} else {
		if layout.extra == nil {
			layout.extra = make([]int, 0, 128)
		}
		layout.extra = append(layout.extra, value)
	}
	layout.count++
	return index
}

func (layout *encodeLayout) set(index, value int) {
	if index < len(layout.local) {
		layout.local[index] = value
	} else {
		layout.extra[index-len(layout.local)] = value
	}
}

func (layout *encodeLayout) next() int {
	index := layout.cursor
	layout.cursor++
	if index < len(layout.local) {
		return layout.local[index]
	}
	return layout.extra[index-len(layout.local)]
}

func writeUint16(buffer []byte, value uint16) {
	binary.LittleEndian.PutUint16(buffer, value)
}

func writeUint32(buffer []byte, value uint32) {
	binary.LittleEndian.PutUint32(buffer, value)
}

func writeUint64(buffer []byte, value uint64) {
	binary.LittleEndian.PutUint64(buffer, value)
}

func integerBits(field *SprotoField, value reflect.Value) uint64 {
	if field.unsigned {
		return value.Uint()
	}
	return uint64(value.Int())
}

func integerWidth(bits uint64) int {
	number := int64(bits)
	if number >= MinInt32 && number <= MaxInt32 {
		return 4
	}
	return 8
}

func omittedField(field *SprotoField, value reflect.Value) bool {
	return field.nilable && value.IsNil()
}

func skipTag(tag, nextTag int) uint16 {
	if nextTag > tag+1 {
		return uint16((nextTag-tag-2)*2 + 1)
	}
	return 0
}

// encodeMessageInto 将子结构直接写入同一个已验证尺寸的输出区间。
func encodeMessageInto(schema *SprotoType, value reflect.Value, output []byte, layout *encodeLayout) int {
	count := layout.next()
	writeUint16(output, uint16(count))
	if value.IsNil() {
		return 2
	}
	record := value.Elem()
	headerOffset, bodyOffset, previous := 2, 2+count*2, -1
	for _, index := range schema.order {
		field := schema.Fields[index]
		if field.Tag < 0 {
			continue
		}
		current := record.Field(field.index[0])
		if omittedField(field, current) {
			continue
		}
		if skip := skipTag(previous, field.Tag); skip > 0 {
			writeUint16(output[headerOffset:], skip)
			headerOffset += 2
		}
		var header uint16
		var length int
		if field.Array || field.Wire == WireStructName {
			length = encodeFieldInto(field, current, output[bodyOffset+4:], layout)
		} else {
			header, length = encodeScalarInto(field, current, output[bodyOffset:])
		}
		writeUint16(output[headerOffset:], header)
		headerOffset += 2
		previous = field.Tag
		if header == 0 {
			writeUint32(output[bodyOffset:], uint32(length))
			bodyOffset += 4 + length
		}
	}
	return bodyOffset
}

func encodeFieldInto(field *SprotoField, value reflect.Value, output []byte, layout *encodeLayout) int {
	if field.Array {
		if value.Len() == 0 {
			return 0
		}
		offset := 0
		switch field.Wire {
		case WireVarintName:
			width := layout.next()
			if field.int64Slice {
				return encodeInt64Slice(*value.Addr().Interface().(*[]int64), width, output)
			}
			output[0] = byte(width)
			for index := 0; index < value.Len(); index++ {
				number := integerBits(field, value.Index(index))
				if width == 4 {
					writeUint32(output[1+index*4:], uint32(number))
				} else {
					writeUint64(output[1+index*8:], number)
				}
			}
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
				length := encodeMessageInto(field.st, value.Index(index), output[offset+4:], layout)
				writeUint32(output[offset:], uint32(length))
				offset += 4 + length
			}
		}
		return offset
	}
	return encodeMessageInto(field.st, value, output, layout)
}

// encodeScalarInto 只读取一次当前标量，同时确定 inline 头并写出非 inline 内容。
func encodeScalarInto(field *SprotoField, value reflect.Value, output []byte) (uint16, int) {
	if field.pointer {
		value = value.Elem()
	}
	switch field.Wire {
	case WireBooleanName:
		if value.Bool() {
			return 4, 0
		}
		return 2, 0
	case WireVarintName:
		bits := integerBits(field, value)
		if bits <= MaxEmbeddedInt {
			return uint16((bits + 1) * 2), 0
		}
		width := integerWidth(bits)
		if width == 4 {
			writeUint32(output[4:], uint32(bits))
		} else {
			writeUint64(output[4:], bits)
		}
		return 0, width
	case WireDoubleName:
		writeUint64(output[4:], math.Float64bits(value.Float()))
		return 0, 8
	case WireBytesName:
		if value.Kind() == reflect.String {
			return 0, copy(output[4:], value.String())
		}
		return 0, copy(output[4:], value.Bytes())
	}
	return 0, 0
}

// encodeInt64Slice 借用调用方切片读取，不创建反射元素或中间数值数组。
func encodeInt64Slice(values []int64, width int, output []byte) int {
	output[0] = byte(width)
	if width == 4 {
		for index, number := range values {
			writeUint32(output[1+index*4:], uint32(number))
		}
	} else {
		for index, number := range values {
			writeUint64(output[1+index*8:], uint64(number))
		}
	}
	return 1 + width*len(values)
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
