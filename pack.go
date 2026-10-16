package sproto

import (
	"bytes"
	"encoding/binary"
	"math/bits"
)

func packSegment(dst []byte, src []byte, ffN int) int {

	var header uint8 = 0
	notzero := 0
	for i, v := range src {
		if v != 0 {
			notzero++
			header |= (1 << uint(i))
			dst[notzero] = v
		}
	}

	if ffN > 0 {
		if notzero >= 6 {
			return 8
		}
	}

	if notzero == 8 {
		return 10
	}

	dst[0] = header
	return notzero + 1
}

func writeFF(dst, src []byte, n int) {
	// 最后一个连续块可能不足八字节，不能读取切片长度之外的底层容量。
	n = min(n, len(src))
	align8 := (n + 7) & (^7)
	dst[0] = 0xff
	dst[1] = uint8(align8/8 - 1)
	copy(dst[2:], src[:n])
	for i := n; i < align8; i++ {
		dst[2+i] = 0
	}
}

// packWord 对稀疏 word 只复制有效字节，混合 word 固定位写出；FF 块仍批量复制。
func packWord(dst []byte, word uint64, ffN int) int {
	nonzero := (((word & 0x7f7f7f7f7f7f7f7f) + 0x7f7f7f7f7f7f7f7f) | word) & 0x8080808080808080
	header := uint8(((nonzero >> 7) * 0x0102040810204080) >> 56)
	if header == 0xff {
		if ffN > 0 {
			return 8
		}
		return 10
	}
	if ffN > 0 {
		zero := ^header
		remaining := zero & (zero - 1)
		if remaining&(remaining-1) == 0 {
			return 8
		}
	}
	// 去掉至多两个置位后是否为零，比普通路径完整计数更直接。
	remaining := header & (header - 1)
	if remaining&(remaining-1) == 0 {
		dst[0] = header
		index := 1
		for mask := header; mask != 0; mask &= mask - 1 {
			shift := bits.TrailingZeros8(mask) * 8
			dst[index] = byte(word >> shift)
			index++
		}
		return index
	}
	// Pack 预留八字节尾空间，零字节暂写的位置会被下一个有效字节覆盖，不进入返回切片。
	_ = dst[8]
	index := 1
	dst[index] = byte(word)
	index += int(header & 1)
	dst[index] = byte(word >> 8)
	index += int((header >> 1) & 1)
	dst[index] = byte(word >> 16)
	index += int((header >> 2) & 1)
	dst[index] = byte(word >> 24)
	index += int((header >> 3) & 1)
	dst[index] = byte(word >> 32)
	index += int((header >> 4) & 1)
	dst[index] = byte(word >> 40)
	index += int((header >> 5) & 1)
	dst[index] = byte(word >> 48)
	index += int((header >> 6) & 1)
	dst[index] = byte(word >> 56)
	index += int((header >> 7) & 1)
	dst[0] = header
	return index
}

// followingFFWords 只在已确认的 FF 起点后搜索，最多接纳 255 个完整非零 word。
// 含零 word 和不足八字节的尾部不计入，继续交给普通 FF 合并逻辑。
func followingFFWords(src []byte) int {
	available := min(2040, len(src)) &^ 7
	if zero := bytes.IndexByte(src[:available], 0); zero >= 0 {
		return zero / 8
	}
	return available / 8
}

func Pack(src []byte) []byte {
	// 每 256 个 word 最多增加一个双字节 FF 头，另保留末尾补齐空间。
	maxsz := len(src) + ((len(src)+2047)/2048)*2 + 8
	packed := make([]byte, maxsz, maxsz)
	offset := 0

	var ffS []byte
	var ffN, ffP int
	for len(src) > 0 {
		var used int
		if len(src) >= 8 {
			word := binary.LittleEndian.Uint64(src)
			if ffN > 0 && (word-0x0101010101010101)&^word&0x8080808080808080 == 0 {
				used = 8
			} else {
				used = packWord(packed[offset:], word, ffN)
			}
		} else {
			used = packSegment(packed[offset:], src, ffN)
		}
		if ffN == 0 {
			if used == 10 {
				ffS = src
				ffP = offset
				ffN = 1
				// 只在已确认的长 FF 起点批量搜索；遇零所在的 word 留给原合并规则。
				if len(src) >= 256 {
					words := followingFFWords(src[8:])
					ffN += words
					used += words * 8
					src = src[words*8:]
					if ffN == 256 {
						writeFF(packed[ffP:], ffS, 2048)
						ffN = 0
					}
				}
			}
		} else if used == 8 {
			ffN++
			if ffN == 256 {
				writeFF(packed[ffP:], ffS, ffN*8)
				ffN = 0
			}
		} else {
			writeFF(packed[ffP:], ffS, ffN*8)
			ffN = 0
		}
		offset += used
		if len(src) <= 8 {
			break
		}
		src = src[8:]
	}

	if ffN > 0 {
		writeFF(packed[ffP:], ffS, len(ffS))
	}
	return packed[:offset]
}

// Unpack 使用默认解包大小预算。
func Unpack(src []byte) ([]byte, error) {
	return UnpackWithLimit(src, 8<<20)
}

// UnpackWithLimit 在扩容前限制解包后含填充的长度；packed 输入大小由调用方限制。
func UnpackWithLimit(src []byte, maxBytes int) ([]byte, error) {
	if maxBytes < 1 {
		return nil, ErrInvalidLimits
	}
	// 首次只扫描 tag/连续块长度，在分配前确认完整边界及有限的确切输出长度。
	size := 0
	for remaining := src; len(remaining) > 0; {
		header := remaining[0]
		consumed, produced := 1+bits.OnesCount8(header), 8
		if header == 0xff {
			if len(remaining) < 2 {
				return nil, ErrUnpack
			}
			produced = (int(remaining[1]) + 1) * 8
			consumed = 2 + produced
		}
		if consumed > len(remaining) {
			return nil, ErrUnpack
		}
		if produced > maxBytes-size {
			return nil, ErrLimit
		}
		size += produced
		remaining = remaining[consumed:]
	}
	unpacked := make([]byte, size)
	offset := 0
	for len(src) > 0 {
		header := src[0]
		src = src[1:]
		if header == 0xff {
			length := (int(src[0]) + 1) * 8
			copy(unpacked[offset:], src[1:1+length])
			offset += length
			src = src[1+length:]
			continue
		}
		for mask := header; mask != 0; mask &= mask - 1 {
			index := bits.TrailingZeros8(mask)
			unpacked[offset+index] = src[0]
			src = src[1:]
		}
		offset += 8
	}
	return unpacked, nil
}
