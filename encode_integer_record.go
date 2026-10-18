package sproto

import "fmt"

// encodedIntegerRecordSize 只用于数组入口已核对实际 Go 类型及全部字段 owner/形状的记录。
// 保留与通用结构编码相同的深度、字段数、tag 间隔和字节预算，不缓存任何标量值。
func encodedIntegerRecordSize(schema *SprotoType, pointer encodePointer, budget *decodeBudget, depth int, layout *encodeLayout) (int, error) {
	if depth > budget.limits.MaxDepth || budget.limits.MaxBytes < 2 {
		return 0, ErrLimit
	}
	layoutIndex := layout.add(0)
	if pointer.value == nil {
		return 2, nil
	}
	size, headers, lastTag := 2, 0, -1
	for _, index := range schema.order {
		field := schema.Fields[index]
		if field.Tag < 0 {
			continue
		}
		bits, present := pointer.integer(field.encOffset)
		if !present {
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
			headers++
		}
		size += 2
		headers++
		lastTag = field.Tag
		if bits > MaxEmbeddedInt {
			size += 4 + integerWidth(bits)
		}
		if size > budget.limits.MaxBytes {
			return 0, ErrLimit
		}
	}
	layout.set(layoutIndex, headers)
	return size, nil
}

// encodeIntegerRecordInto 复用同一次预检的布局，字段值仍逐个读取并写入独占输出。
func encodeIntegerRecordInto(schema *SprotoType, pointer encodePointer, output []byte, layout *encodeLayout) int {
	count := layout.next()
	writeUint16(output, uint16(count))
	if pointer.value == nil {
		return 2
	}
	headerOffset, bodyOffset, previous := 2, 2+count*2, -1
	for _, index := range schema.order {
		field := schema.Fields[index]
		if field.Tag < 0 {
			continue
		}
		bits, present := pointer.integer(field.encOffset)
		if !present {
			continue
		}
		if skip := skipTag(previous, field.Tag); skip > 0 {
			writeUint16(output[headerOffset:], skip)
			headerOffset += 2
		}
		header, length := encodeIntegerInto(bits, output[bodyOffset:])
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
