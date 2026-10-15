package sproto

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
)

type PackTestCase struct {
	Name     string
	Unpacked []byte
	Origin   []byte
	Packed   []byte
}

var packTestCases []*PackTestCase = []*PackTestCase{
	&PackTestCase{
		Name:     "SimplePack",
		Origin:   []byte{0x08, 0x00, 0x00, 0x00, 0x03, 0x00, 0x02, 0x00, 0x19, 0x00, 0x00, 0x00, 0xaa, 0x01, 0x00, 0x00},
		Packed:   []byte{0x51, 0x08, 0x03, 0x02, 0x31, 0x19, 0xaa, 0x01},
		Unpacked: []byte{0x08, 0x00, 0x00, 0x00, 0x03, 0x00, 0x02, 0x00, 0x19, 0x00, 0x00, 0x00, 0xaa, 0x01, 0x00, 0x00},
	},
	&PackTestCase{
		Name: "FFPack",
		Origin: bytes.Join([][]byte{
			bytes.Repeat([]byte{0x8a}, 30),
			[]byte{0x00, 0x00},
		}, nil),
		Packed: bytes.Join([][]byte{
			[]byte{0xff, 0x03},
			bytes.Repeat([]byte{0x8a}, 30),
			[]byte{0x00, 0x00},
		}, nil),
		Unpacked: bytes.Join([][]byte{
			bytes.Repeat([]byte{0x8a}, 30),
			[]byte{0x00, 0x00},
		}, nil),
	},
	&PackTestCase{
		Name:     "FFPack2",
		Origin:   []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E},
		Packed:   []byte{0xFF, 0x01, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x00, 0x00},
		Unpacked: []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x00, 0x00},
	},
}

// Pack 保持官方固定向量及所有八字节 mask 的位序和原始字节。
func TestPack(t *testing.T) {
	for _, tc := range packTestCases {
		packed := Pack(tc.Origin)
		if !bytes.Equal(packed, tc.Packed) {
			t.Log("packed:", packed)
			t.Log("expected:", tc.Packed)
			t.Fatalf("test case *%s* failed", tc.Name)
		}
	}
	for mask := 0; mask < 256; mask++ {
		for _, nonzero := range []byte{1, 0x7f, 0x80, 0xff} {
			raw := make([]byte, 8)
			want := []byte{byte(mask)}
			if mask == 255 {
				want = append(want, 0)
			}
			for index := range raw {
				if mask&(1<<index) != 0 {
					raw[index] = nonzero
					want = append(want, nonzero)
				}
			}
			if packed := Pack(raw); !bytes.Equal(packed, want) {
				t.Fatalf("mask=%02x value=%02x: %x != %x", mask, nonzero, packed, want)
			}
		}
	}
}

func TestUnpack(t *testing.T) {
	var allUnpacked, allPacked []byte
	for _, tc := range packTestCases {
		unpacked, err := Unpack(tc.Packed)
		if err != nil {
			t.Fatalf("test case *%s* failed with error:%s", tc.Name, err)
		}
		if !bytes.Equal(unpacked, tc.Unpacked) {
			t.Log("unpacked:", unpacked)
			t.Log("expected:", tc.Unpacked)
			t.Fatalf("test case *%s* failed", tc.Name)
		}
		allUnpacked = Append(allUnpacked, tc.Unpacked)
		allPacked = Append(allPacked, tc.Packed)
	}
	unpacked, err := Unpack(allPacked)
	if err != nil {
		t.Fatalf("test case *total* failed with error:%s", err)
	}
	if !bytes.Equal(unpacked, allUnpacked) {
		t.Log("unpacked:", unpacked)
		t.Log("expected:", allUnpacked)
		t.Fatal("test case *total* failed")
	}
}

// 验证预算只计算解包后含填充的长度，等于预算时允许接纳，超过预算时拒绝。
func TestUnpackOutputBudget(t *testing.T) {
	for _, source := range [][]byte{
		bytes.Repeat([]byte{1}, 8),
		bytes.Repeat([]byte{0}, 8),
		{1, 0, 2},
		bytes.Repeat([]byte{1}, 2048),
	} {
		packed := Pack(source)
		expected := append([]byte{}, source...)
		expected = append(expected, make([]byte, (8-len(source)%8)%8)...)
		output, err := UnpackWithLimit(packed, len(expected))
		if err != nil || !bytes.Equal(output, expected) {
			t.Fatalf("exact budget=%d packed=%d output=%x error=%v", len(expected), len(packed), output, err)
		}
		if _, err := UnpackWithLimit(packed, len(expected)-1); !errors.Is(err, ErrLimit) {
			t.Fatalf("short budget: %v", err)
		}
	}
	if _, err := UnpackWithLimit(nil, 0); !errors.Is(err, ErrInvalidLimits) {
		t.Fatalf("invalid budget: %v", err)
	}
	for _, malformed := range [][]byte{{255}, {255, 0, 1}, {1}, {128}} {
		if _, err := UnpackWithLimit(malformed, 32); !errors.Is(err, ErrUnpack) {
			t.Fatalf("truncated %x: %v", malformed, err)
		}
	}
}

// 验证连续块达到 256 块边界且最后只有 6/7 字节时，不越界读取并正确补零。
func TestPackPartialRunAtBlockLimit(t *testing.T) {
	for _, length := range []int{2046, 2047, 2048, 2049, 4094, 4095, 4096} {
		t.Run(fmt.Sprint(length), func(t *testing.T) {
			for _, extraCapacity := range []int{0, 16} {
				source := make([]byte, length, length+extraCapacity)
				backing := source[:cap(source)]
				for index := range backing {
					backing[index] = 0xee
				}
				for index := range source {
					source[index] = 1
				}
				packed := Pack(source)
				output, err := Unpack(packed)
				expected := append([]byte{}, source...)
				expected = append(expected, make([]byte, (8-length%8)%8)...)
				if err != nil || !bytes.Equal(output, expected) {
					t.Fatalf("partial run: source=%d extraCapacity=%d output=%d error=%v", length, extraCapacity, len(output), err)
				}
			}
		})
	}
}
