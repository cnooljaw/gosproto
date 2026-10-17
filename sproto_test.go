package sproto

import (
	"bytes"
	"errors"
	"math"
	"reflect"
	"testing"
)

type PhoneNumber struct {
	Number *string `sproto:"string,0,name=number"`
	Type   *int    `sproto:"integer,1,name=type"`
}

type Person struct {
	Name  *string        `sproto:"string,0,name=name"`
	Id    *int           `sproto:"integer,1,name=id"`
	Email *string        `sproto:"string,2,name=email"`
	Phone []*PhoneNumber `sproto:"struct,3,array,name=phone"`
}

type AddressBook struct {
	Person []*Person `sproto:"struct,0,array,name=person"`
}

type Human struct {
	Name     *string  `sproto:"string,0,name=name"`
	Age      *int     `sproto:"integer,1,name=age"`
	Marital  *bool    `sproto:"boolean,2,name=marital"`
	Children []*Human `sproto:"struct,3,array,name=children"`
}

type Data struct {
	Numbers   []int64  `sproto:"integer,0,array,name=numbers"`
	Bools     []bool   `sproto:"boolean,1,array,name=bools"`
	Number    *int     `sproto:"integer,2,name=number"`
	BigNumber *int64   `sproto:"integer,3,name=bignumber"`
	Strings   []string `sproto:"string,4,array,name=strings"`
	Bytes     []byte   `sproto:"string,5,name=bytes"`
}

type TestCase struct {
	Name   string
	Struct interface{}
	Data   []byte
}

var abData []byte = []byte{
	1, 0, 0, 0, 122, 0, 0, 0, 68, 0, 0, 0, 4, 0, 0, 0, 34, 78, 1, 0,
	0, 0, 5, 0, 0, 0, 65, 108, 105, 99, 101, 45, 0, 0, 0, 19, 0, 0, 0,
	2, 0, 0, 0, 4, 0, 9, 0, 0, 0, 49, 50, 51, 52, 53, 54, 55, 56, 57,
	18, 0, 0, 0, 2, 0, 0, 0, 6, 0, 8, 0, 0, 0, 56, 55, 54, 53, 52, 51,
	50, 49, 46, 0, 0, 0, 4, 0, 0, 0, 66, 156, 1, 0, 0, 0, 3, 0, 0, 0,
	66, 111, 98, 25, 0, 0, 0, 21, 0, 0, 0, 2, 0, 0, 0, 8, 0, 11, 0, 0,
	0, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 48,
}

var abDataPacked []byte = []byte{
	17, 1, 122, 17, 68, 4, 71, 34, 78, 1, 5, 252, 65, 108, 105, 99, 101,
	45, 136, 19, 2, 40, 4, 9, 254, 49, 50, 51, 52, 53, 54, 55, 71, 56, 57,
	18, 2, 20, 6, 8, 255, 0, 56, 55, 54, 53, 52, 51, 50, 49, 17, 46, 4, 71,
	66, 156, 1, 3, 60, 66, 111, 98, 25, 34, 21, 2, 138, 8, 11, 48, 255, 0,
	49, 50, 51, 52, 53, 54, 55, 56, 3, 57, 48,
}

var ab AddressBook = AddressBook{
	Person: []*Person{
		&Person{
			Name: String("Alice"),
			Id:   Int(10000),
			Phone: []*PhoneNumber{
				&PhoneNumber{
					Number: String("123456789"),
					Type:   Int(1),
				},
				&PhoneNumber{
					Number: String("87654321"),
					Type:   Int(2),
				},
			},
		},
		&Person{
			Name: String("Bob"),
			Id:   Int(20000),
			Phone: []*PhoneNumber{
				&PhoneNumber{
					Number: String("01234567890"),
					Type:   Int(3),
				},
			},
		},
	},
}

var testCases []*TestCase = []*TestCase{
	&TestCase{
		Name: "SimpleStruct",
		Struct: &Human{
			Name:    String("Alice"),
			Age:     Int(13),
			Marital: Bool(false),
		},
		Data: []byte{
			0x03, 0x00, // (fn = 3)
			0x00, 0x00, // (id = 0, value in data part)
			0x1C, 0x00, // (id = 1, value = 13)
			0x02, 0x00, // (id = 2, value = false)
			0x05, 0x00, 0x00, 0x00, // (sizeof "Alice")
			0x41, 0x6C, 0x69, 0x63, 0x65, // ("Alice")
		},
	},
	&TestCase{
		Name: "StructArray",
		Struct: &Human{
			Name: String("Bob"),
			Age:  Int(40),
			Children: []*Human{
				&Human{
					Name: String("Alice"),
					Age:  Int(13),
				},
				&Human{
					Name: String("Carol"),
					Age:  Int(5),
				},
			},
		},
		Data: []byte{
			0x04, 0x00, // (fn = 4)
			0x00, 0x00, // (id = 0, value in data part)
			0x52, 0x00, // (id = 1, value = 40)
			0x01, 0x00, // (skip id = 2)
			0x00, 0x00, // (id = 3, value in data part)
			0x03, 0x00, 0x00, 0x00, // (sizeof "Bob")
			0x42, 0x6F, 0x62, // ("Bob")
			0x26, 0x00, 0x00, 0x00, // (sizeof children)
			0x0F, 0x00, 0x00, 0x00, // (sizeof child 1)
			0x02, 0x00, //(fn = 2)
			0x00, 0x00, //(id = 0, value in data part)
			0x1C, 0x00, //(id = 1, value = 13)
			0x05, 0x00, 0x00, 0x00, // (sizeof "Alice")
			0x41, 0x6C, 0x69, 0x63, 0x65, //("Alice")
			0x0F, 0x00, 0x00, 0x00, // (sizeof child 2)
			0x02, 0x00, //(fn = 2)
			0x00, 0x00, //(id = 0, value in data part)
			0x0C, 0x00, //(id = 1, value = 5)
			0x05, 0x00, 0x00, 0x00, //(sizeof "Carol")
			0x43, 0x61, 0x72, 0x6F, 0x6C, //("Carol")
		},
	},
	&TestCase{
		Name: "NumberArray",
		Struct: &Data{
			Numbers: []int64{1, 2, 3, 4, 5},
		},
		Data: []byte{
			0x01, 0x00, // (fn = 1)
			0x00, 0x00, // (id = 0, value in data part)

			0x15, 0x00, 0x00, 0x00, // (sizeof numbers)
			0x04,                   //(sizeof int32)
			0x01, 0x00, 0x00, 0x00, //(1)
			0x02, 0x00, 0x00, 0x00, //(2)
			0x03, 0x00, 0x00, 0x00, //(3)
			0x04, 0x00, 0x00, 0x00, //(4)
			0x05, 0x00, 0x00, 0x00, //(5)
		},
	},
	&TestCase{
		Name: "BigNumberArray",
		Struct: &Data{
			Numbers: []int64{
				(1 << 32) + 1,
				(1 << 32) + 2,
				(1 << 32) + 3,
			},
		},
		Data: []byte{
			0x01, 0x00, // (fn = 1)
			0x00, 0x00, // (id = 0, value in data part)

			0x19, 0x00, 0x00, 0x00, // (sizeof numbers)
			0x08,                                           //(sizeof int32)
			0x01, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, //((1<<32) + 1)
			0x02, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, //((1<<32) + 2)
			0x03, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, //((1<<32) + 3)
		},
	},
	&TestCase{
		Name: "BoolArray",
		Struct: &Data{
			Bools: []bool{false, true, false},
		},
		Data: []byte{
			0x02, 0x00, // (fn = 2)
			0x01, 0x00, // (skip id = 0)
			0x00, 0x00, // (id = 1, value in data part)

			0x03, 0x00, 0x00, 0x00, // (sizeof bools)
			0x00, //(false)
			0x01, //(true)
			0x00, //(false)
		},
	},
	&TestCase{
		Name: "Bytes",
		Struct: &Data{
			Bytes: []byte{0x28, 0x29, 0x30, 0x31},
		},
		Data: []byte{
			0x02, 0x00, // (fn = 2)
			0x09, 0x00, // (skip id = 4)
			0x00, 0x00, // (id = 5, value in data part)

			0x04, 0x00, 0x00, 0x00, // (sizeof bytes)
			0x28, //(0x28)
			0x29, //(0x29)
			0x30, //(0x30)
			0x31, //(0x31)
		},
	},
	&TestCase{
		Name: "StringArray",
		Struct: &Data{
			Strings: []string{"Bob", "Alice", "Carol"},
		},
		Data: []byte{
			0x02, 0x00, // (fn = 2)
			0x07, 0x00, // (skip id = 3)
			0x00, 0x00, // (id = 4, value in data part)

			0x19, 0x00, 0x00, 0x00, // (sizeof []string)
			0x03, 0x00, 0x00, 0x00, // (sizeof "Bob")
			0x42, 0x6F, 0x62, // ("Bob")
			0x05, 0x00, 0x00, 0x00, // (sizeof "Alice")
			0x41, 0x6C, 0x69, 0x63, 0x65, //("Alice")
			0x05, 0x00, 0x00, 0x00, //(sizeof "Carol")
			0x43, 0x61, 0x72, 0x6F, 0x6C, //("Carol")
		},
	},
	&TestCase{
		Name: "Number",
		Struct: &Data{
			Number:    Int(100000),
			BigNumber: Int64(-10000000000),
		},
		Data: []byte{
			0x03, 0x00, // (fn = 3)
			0x03, 0x00, // (skip id = 1)
			0x00, 0x00, // (id = 2, value in data part)
			0x00, 0x00, // (id = 3, value in data part)

			0x04, 0x00, 0x00, 0x00, //(sizeof number, data part)
			0xA0, 0x86, 0x01, 0x00, //(100000, 32bit integer)

			0x08, 0x00, 0x00, 0x00, //(sizeof bignumber, data part)
			0x00, 0x1C, 0xF4, 0xAB, 0xFD, 0xFF, 0xFF, 0xFF, //(-10000000000, 64bit integer)
		},
	},
	&TestCase{
		Name:   "AddressBook",
		Struct: &ab,
		Data:   abData,
	},
}

func TestEncode(t *testing.T) {
	for _, tc := range testCases {
		output, err := Encode(tc.Struct)
		if err != nil {
			t.Fatalf("test case *%s* failed with error:%s", tc.Name, err)
		}
		if !bytes.Equal(output, tc.Data) {
			t.Log("encoded:", output)
			t.Log("expected:", tc.Data)
			t.Fatalf("test case %s failed", tc.Name)
		}
	}
}

func TestDecode(t *testing.T) {
	for _, tc := range testCases {
		sp := reflect.New(reflect.TypeOf(tc.Struct).Elem()).Interface()
		used, err := Decode(tc.Data, sp)
		if err != nil {
			t.Fatalf("test case *%s* failed with error:%s", tc.Name, err)
		}

		if used != len(tc.Data) {
			t.Fatalf("test case *%s* failed: data length mismatch", tc.Name)
		}

		output, err := Encode(sp)
		if err != nil {
			t.Fatalf("test case *%s* failed with error:%s", tc.Name, err)
		}
		if !bytes.Equal(output, tc.Data) {
			t.Log("encoded:", output)
			t.Log("expected:", tc.Data)
			t.Fatalf("test case %s failed", tc.Name)
		}
	}
}

type packedBufferMessage struct {
	Payload []byte `sproto:"string,0"`
}

// EncodePacked 在所有 mask、FF 分段和短尾块上保持分步编码的字节、容量与独占结果。
func TestEncodePackedWorkspace(t *testing.T) {
	check := func(payload []byte) {
		t.Helper()
		storage := bytes.Repeat([]byte{0xee}, len(payload)+19)
		copy(storage[3:], payload)
		input := &packedBufferMessage{Payload: storage[3 : 3+len(payload)]}
		snapshot := bytes.Clone(storage)
		encoded, err := Encode(input)
		if err != nil {
			t.Fatal(err)
		}
		want := Pack(encoded)
		got, err := EncodePacked(input)
		if err != nil || !bytes.Equal(got, want) || cap(got) != cap(want) || !bytes.Equal(storage, snapshot) {
			t.Fatalf("payload=%d len/cap=%d/%d want=%d/%d err=%v", len(payload), len(got), cap(got), len(want), cap(want), err)
		}
		previous := bytes.Clone(got)
		input.Payload[0] ^= 0xff
		if _, err := EncodePacked(input); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, previous) || !bytes.Equal(encoded[8:], payload) {
			t.Fatal("previous packed or encoded output changed")
		}
	}
	for _, words := range []int{0, 1, 255, 256, 257} {
		for mask := 0; mask < 256; mask++ {
			for tail := 0; tail < 8; tail++ {
				payload := bytes.Repeat([]byte{0xa5}, words*8)
				for bit := 0; bit < 8; bit++ {
					value := byte(0)
					if mask&(1<<bit) != 0 {
						value = byte(0x80 + bit)
					}
					payload = append(payload, value)
				}
				payload = append(payload, bytes.Repeat([]byte{0xb7}, tail)...)
				check(payload)
			}
		}
	}
	// 超过两个 FF 段后，初始编码头的压缩量不再足以抵消 FF 头增长。
	for _, size := range []int{4095, 4096, 4097, 65535, 65536, 65537} {
		check(bytes.Repeat([]byte{0xa5}, size))
	}
}

// EncodePacked 比分步 Encode/Pack 少一次缓冲分配；相对比较也适用于检查插桩构建。
func TestEncodePackedAllocation(t *testing.T) {
	input := &packedBufferMessage{Payload: bytes.Repeat([]byte{0x91}, 256)}
	var packed []byte
	allocations := testing.AllocsPerRun(100, func() {
		var err error
		packed, err = EncodePacked(input)
		if err != nil {
			t.Fatal(err)
		}
	})
	separate := testing.AllocsPerRun(100, func() {
		encoded, err := Encode(input)
		if err != nil {
			t.Fatal(err)
		}
		packed = Pack(encoded)
	})
	if len(packed) == 0 || separate-allocations != 1 {
		t.Fatalf("combined=%v separate=%v, want one fewer allocation", allocations, separate)
	}
}

// EncodePacked 在输入无效和默认预算耗尽时保持可判断错误且不返回部分输出。
func TestEncodePackedErrors(t *testing.T) {
	type unsigned struct {
		Value uint64 `sproto:"integer,0"`
	}
	for _, sample := range []struct {
		input any
		want  error
	}{
		{nil, ErrNil},
		{packedBufferMessage{}, ErrNonPtr},
		{&unsigned{Value: math.MaxUint64}, ErrEncode},
		{&packedBufferMessage{Payload: make([]byte, 8<<20)}, ErrLimit},
	} {
		got, err := EncodePacked(sample.input)
		if !errors.Is(err, sample.want) || got != nil {
			t.Fatalf("input=%T output=%x err=%v want=%v", sample.input, got, err, sample.want)
		}
	}
}

func BenchmarkEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Encode(&ab)
	}
}

func BenchmarkDecode(b *testing.B) {
	var ab AddressBook
	for i := 0; i < b.N; i++ {
		Decode(abData, &ab)
	}
}

func BenchmarkEncodePacked(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, err := EncodePacked(&ab)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDecodePacked(b *testing.B) {
	var ab AddressBook
	for i := 0; i < b.N; i++ {
		if err := DecodePacked(abDataPacked, &ab); err != nil {
			b.Fatal(err)
		}
	}
}
