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

// packFFMergeable 判断当前 word 是否可按官方规则继续 FF 段。
func packFFMergeable(header byte) bool {
	zero := ^header
	remaining := zero & (zero - 1)
	return remaining&(remaining-1) == 0
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

// Pack 按 Sproto 规则压缩字节，返回独占缓冲且不修改输入。
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
				nonzero := (((word & 0x7f7f7f7f7f7f7f7f) + 0x7f7f7f7f7f7f7f7f) | word) & 0x8080808080808080
				header := uint8(((nonzero >> 7) * 0x0102040810204080) >> 56)
				switch {
				case header == 0xff:
					used = 10
					if ffN > 0 {
						used = 8
					}
				case ffN > 0 && packFFMergeable(header):
					used = 8
				default:
					dst := packed[offset:]
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
						used = index
					} else {
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
						used = index
					}
				}
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

// zeroHeaderPrefix 按有效切片长度统计连续零 mask，遇到首个非零字节立即停止。
// 调用方只在已确认的零段后使用；预检传入的范围另受剩余输出预算限制。
func zeroHeaderPrefix(src []byte) int {
	index := 0
	for len(src)-index >= 8 {
		word := binary.LittleEndian.Uint64(src[index:])
		if word != 0 {
			return index + bits.TrailingZeros64(word)/8
		}
		index += 8
	}
	for index < len(src) && src[index] == 0 {
		index++
	}
	return index
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
		consumed, produced := 1, 8
		if header == 0 {
			if len(remaining) >= 2 && remaining[1] == 0 {
				words := (maxBytes - size) / 8
				if words < 2 {
					return nil, ErrLimit
				}
				// 多检查一个零 mask 即可判定超额，不为小预算扫描整段大输入。
				end := min(len(remaining), words+1)
				run := 2 + zeroHeaderPrefix(remaining[2:end])
				if run > words {
					return nil, ErrLimit
				}
				consumed, produced = run, run*8
			}
		} else if header == 0xff {
			if len(remaining) < 2 {
				return nil, ErrUnpack
			}
			produced = (int(remaining[1]) + 1) * 8
			consumed = 2 + produced
		} else {
			consumed += bits.OnesCount8(header)
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
		if header == 0 {
			run := 1
			if len(src) >= 2 && src[1] == 0 {
				run = 2 + zeroHeaderPrefix(src[2:])
			}
			// 输出由本次 make 清零，已预检的零段只需推进游标。
			offset += run * 8
			src = src[run:]
			continue
		}
		src = src[1:]
		if header == 0xff {
			length := (int(src[0]) + 1) * 8
			copy(unpacked[offset:], src[1:1+length])
			offset += length
			src = src[1+length:]
			continue
		}
		// 零 mask 已在上方消费，普通 mask 至少有一个有效字节。
		for mask := header; ; {
			index := bits.TrailingZeros8(mask)
			unpacked[offset+index] = src[0]
			src = src[1:]
			mask &= mask - 1
			if mask == 0 {
				break
			}
		}
		offset += 8
	}
	return unpacked, nil
}
