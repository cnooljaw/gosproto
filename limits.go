package sproto

import (
	"errors"
	"fmt"
	"math"
	"reflect"
)

// Limits 是编解码的本地资源预算；零值使用有限默认值，不改变 wire 格式。
type Limits struct {
	MaxBytes    int
	MaxDepth    int
	MaxElements int
}

var (
	// ErrLimit 表示载荷、深度或集合超过接纳预算。
	ErrLimit = errors.New("sproto: resource limit exceeded")
	// ErrInvalidLimits 表示显式预算无效。
	ErrInvalidLimits = errors.New("sproto: invalid limits")
	// ErrEncode 表示输入值不能按当前 schema 安全编码。
	ErrEncode = errors.New("sproto: invalid encode value")
)

func normalizeLimits(limits Limits) (Limits, error) {
	if limits.MaxBytes == 0 {
		limits.MaxBytes = 8 << 20
	}
	if limits.MaxDepth == 0 {
		limits.MaxDepth = 64
	}
	if limits.MaxElements == 0 {
		limits.MaxElements = 100000
	}
	if limits.MaxBytes < 1 || limits.MaxDepth < 1 || limits.MaxDepth > 128 || limits.MaxElements < 1 {
		return Limits{}, ErrInvalidLimits
	}
	return limits, nil
}

type decodeBudget struct {
	limits          Limits
	depth, elements int
}

func (budget *decodeBudget) add(count int) error {
	if count < 0 || count > budget.limits.MaxElements-budget.elements {
		return ErrLimit
	}
	budget.elements += count
	return nil
}

// EncodeWithLimits 在任何递归编码和大块分配前验证值、深度与预计输出大小。
func EncodeWithLimits(message interface{}, limits Limits) ([]byte, error) {
	limits, err := normalizeLimits(limits)
	if err != nil {
		return nil, err
	}
	typ, value, err := getbase(message)
	if err != nil {
		return nil, err
	}
	schema, err := GetSprotoType(typ.Elem())
	if err != nil {
		return nil, err
	}
	budget := &decodeBudget{limits: limits}
	_, err = encodedSize(schema, value, budget, 1)
	if err != nil {
		return nil, err
	}
	return encodeMessage(schema, value), nil
}

func encodedSize(schema *SprotoType, value reflect.Value, budget *decodeBudget, depth int) (int, error) {
	if depth > budget.limits.MaxDepth {
		return 0, ErrLimit
	}
	size, lastTag := 2, -1
	if size > budget.limits.MaxBytes {
		return 0, ErrLimit
	}
	if value.IsNil() {
		return size, nil
	}
	for _, index := range schema.order {
		field := schema.Fields[index]
		if field.Tag < 0 {
			continue
		}
		current := value.Elem().FieldByIndex(field.index)
		if (current.Kind() == reflect.Ptr || current.Kind() == reflect.Slice) && current.IsNil() {
			continue
		}
		if err := budget.add(1); err != nil {
			return 0, err
		}
		gap := field.Tag - lastTag - 1
		if gap > 32768 {
			return 0, fmt.Errorf("%w: tag gap", ErrEncode)
		}
		if gap > 0 {
			size += 2
		}
		size += 2
		lastTag = field.Tag
		part, inline, err := fieldSize(field, current, budget, depth)
		if err != nil {
			return 0, err
		}
		if !inline {
			size += 4 + part
		}
		if size > budget.limits.MaxBytes {
			return 0, ErrLimit
		}
	}
	return size, nil
}

func fieldSize(field *SprotoField, value reflect.Value, budget *decodeBudget, depth int) (int, bool, error) {
	if field.Array {
		if err := budget.add(value.Len()); err != nil {
			return 0, false, err
		}
		size := 0
		switch field.Wire {
		case WireVarintName:
			width := 4
			for i := 0; i < value.Len(); i++ {
				if err := validateInteger(value.Index(i)); err != nil {
					return 0, false, err
				}
				_, current := extractInt(value.Index(i))
				if current > width {
					width = current
				}
			}
			if value.Len() > 0 {
				size = 1 + width*value.Len()
			}
		case WireDoubleName:
			if value.Len() > 0 {
				size = 1 + 8*value.Len()
			}
		case WireBooleanName:
			size = value.Len()
		case WireBytesName:
			for i := 0; i < value.Len(); i++ {
				size += 4 + value.Index(i).Len()
				if size > budget.limits.MaxBytes {
					return 0, false, ErrLimit
				}
			}
		case WireStructName:
			for i := 0; i < value.Len(); i++ {
				part, err := encodedSize(field.st, value.Index(i), budget, depth+1)
				if err != nil {
					return 0, false, err
				}
				size += 4 + part
				if size > budget.limits.MaxBytes {
					return 0, false, ErrLimit
				}
			}
		}
		if size > budget.limits.MaxBytes {
			return 0, false, ErrLimit
		}
		return size, false, nil
	}
	if field.Wire == WireStructName {
		size, err := encodedSize(field.st, value, budget, depth+1)
		return size, false, err
	}
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	switch field.Wire {
	case WireVarintName:
		if err := validateInteger(value); err != nil {
			return 0, false, err
		}
		bits, width := extractInt(value)
		return width, bits <= MaxEmbeddedInt, nil
	case WireDoubleName:
		return 8, false, nil
	case WireBooleanName:
		return 0, true, nil
	case WireBytesName:
		return value.Len(), false, nil
	}
	return 0, false, ErrEncode
}

func validateInteger(value reflect.Value) error {
	switch value.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if value.Uint() > math.MaxInt64 {
			return fmt.Errorf("%w: integer exceeds signed 64-bit range", ErrEncode)
		}
	}
	return nil
}

// Validate 检查显式预算；零值合法并在编解码时使用默认值。
func (limits Limits) Validate() error {
	_, err := normalizeLimits(limits)
	return err
}
