package sproto_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"math"
	"os"
	"os/exec"
	"reflect"
	"testing"

	sproto "github.com/cnooljaw/gosproto"
)

type tree struct {
	Next *tree `sproto:"struct,0"`
}
type numeric struct {
	Integer int64     `sproto:"integer,0"`
	Values  []int64   `sproto:"integer,1,array"`
	Double  float64   `sproto:"double,2"`
	Doubles []float64 `sproto:"double,3,array"`
	Text    string    `sproto:"string,4"`
	Binary  []byte    `sproto:"string,5"`
}

// 验证预算在解包扩容、递归和元素分配前拒绝输入，且解码失败不修改原值。
func TestLimitsBeforeAllocation(t *testing.T) {
	if _, err := sproto.UnpackWithLimit([]byte{0, 0}, 8); !errors.Is(err, sproto.ErrLimit) {
		t.Fatalf("unpack limit: %v", err)
	}
	value := &tree{Next: &tree{Next: &tree{}}}
	if _, err := sproto.EncodeWithLimits(value, sproto.Limits{MaxDepth: 2}); !errors.Is(err, sproto.ErrLimit) {
		t.Fatalf("encode depth: %v", err)
	}
	encoded, err := sproto.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	output := tree{Next: &tree{}}
	original := output.Next
	if _, err := sproto.DecodeWithLimits(encoded, &output, sproto.Limits{MaxDepth: 2}); !errors.Is(err, sproto.ErrLimit) {
		t.Fatalf("decode depth: %v", err)
	}
	if output.Next != original {
		t.Fatal("failed decode changed caller value")
	}
	encoded, err = sproto.Encode(&numeric{Values: []int64{1, 2, 3}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := sproto.DecodeWithLimits(encoded, &numeric{}, sproto.Limits{MaxElements: 2}); !errors.Is(err, sproto.ErrLimit) {
		t.Fatalf("element limit: %v", err)
	}
	if _, err := sproto.EncodeWithLimits(&numeric{Text: "long"}, sproto.Limits{MaxBytes: 4}); !errors.Is(err, sproto.ErrLimit) {
		t.Fatalf("encode byte limit: %v", err)
	}
}

// 验证非法 schema 不污染缓存，窄整数溢出返回错误而非截断。
func TestInvalidSchemaAndIntegerRange(t *testing.T) {
	type duplicate struct {
		A int64 `sproto:"integer,0"`
		B int64 `sproto:"integer,0"`
	}
	for i := 0; i < 2; i++ {
		if _, err := sproto.GetSprotoType(reflect.TypeOf(duplicate{})); err == nil {
			t.Fatal("invalid schema cached")
		}
	}
	type narrow struct {
		Value int8 `sproto:"integer,0"`
	}
	encoded, _ := hex.DecodeString("01000202")
	if _, err := sproto.Decode(encoded, &narrow{}); err == nil {
		t.Fatal("integer overflow accepted")
	}
	type unsigned struct {
		Value uint64 `sproto:"integer,0"`
	}
	if _, err := sproto.Encode(&unsigned{Value: math.MaxUint64}); !errors.Is(err, sproto.ErrEncode) {
		t.Fatalf("unsigned wire range: %v", err)
	}
	input := &unsigned{Value: math.MaxInt64}
	encoded, err := sproto.Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	var output unsigned
	if _, err := sproto.Decode(encoded, &output); err != nil || output != *input {
		t.Fatalf("unsigned: %+v %v", output, err)
	}
	type small int8
	type unsignedArrays struct {
		Wide   []uint64 `sproto:"integer,0,array"`
		Narrow []uint32 `sproto:"integer,1,array"`
		Signed []small  `sproto:"integer,2,array"`
	}
	arrays := &unsignedArrays{Wide: []uint64{0, math.MaxInt64}, Narrow: []uint32{math.MaxUint32}, Signed: []small{-128, 127}}
	encoded, err = sproto.Encode(arrays)
	if err != nil {
		t.Fatal(err)
	}
	var decoded unsignedArrays
	if _, err := sproto.Decode(encoded, &decoded); err != nil || !reflect.DeepEqual(arrays, &decoded) {
		t.Fatalf("integer array types: %+v %v", decoded, err)
	}
	arrays.Wide = []uint64{1, math.MaxUint64}
	if _, err := sproto.Encode(arrays); !errors.Is(err, sproto.ErrEncode) {
		t.Fatalf("unsigned array wire range: %v", err)
	}
}

// 验证字段未定义时能安全跳过，不输出日志或误解后续字段。
func TestUnknownFieldAndMalformedScalar(t *testing.T) {
	type empty struct{}
	encoded, _ := hex.DecodeString("0100000004000000ffffffff")
	if _, err := sproto.Decode(encoded, &empty{}); err != nil {
		t.Fatal(err)
	}
	type flag struct {
		Value bool `sproto:"boolean,0"`
	}
	if _, err := sproto.Decode(encoded, &flag{}); err == nil {
		t.Fatal("data field accepted as boolean")
	}
	type source struct {
		Low     int64 `sproto:"integer,0"`
		Unknown int64 `sproto:"integer,1"`
		Middle  int64 `sproto:"integer,30000"`
		High    int64 `sproto:"integer,60000"`
	}
	type destination struct {
		Low    int64 `sproto:"integer,0"`
		Unused int64 `sproto:"integer,2"`
		High   int64 `sproto:"integer,60000"`
	}
	encoded, err := sproto.Encode(&source{Low: 1, Unknown: 9, Middle: 21, High: 42})
	if err != nil {
		t.Fatal(err)
	}
	var result destination
	if used, err := sproto.Decode(encoded, &result); err != nil || used != len(encoded) || result != (destination{Low: 1, High: 42}) {
		t.Fatalf("dense gap and sparse fields: %+v %v", result, err)
	}
}

// 验证官方固定向量的二进制编码完全一致，涵盖 64 位边界、double、UTF-8 与 binary。
func TestOfficialGoldenVectors(t *testing.T) {
	type scalar struct {
		Value float64 `sproto:"double,0"`
	}
	encoded, err := sproto.Encode(&scalar{Value: 0.01171875})
	if err != nil {
		t.Fatal(err)
	}
	expected, _ := hex.DecodeString("0100000008000000000000000000883f")
	if !bytes.Equal(encoded, expected) {
		t.Fatalf("double: %x", encoded)
	}
	input := &numeric{Integer: math.MinInt64, Values: []int64{math.MinInt64, -1, 0, math.MaxInt64}, Double: 0.01171875, Doubles: []float64{0, -0.5, math.Inf(1)}, Text: "中文🙂", Binary: []byte{0, 255, 1}}
	encoded, err = sproto.Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	expected, _ = hex.DecodeString("060000000000000000000000000008000000000000000000008021000000080000000000000080ffffffffffffffff0000000000000000ffffffffffffff7f08000000000000000000883f19000000080000000000000000000000000000e0bf000000000000f07f0a000000e4b8ade69687f09f99820300000000ff01")
	if !bytes.Equal(encoded, expected) {
		t.Fatalf("numeric official vector: %x", encoded)
	}
	var output numeric
	if _, err := sproto.Decode(encoded, &output); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(input, &output) {
		t.Fatalf("round trip: %+v", output)
	}
}

// 验证 Go 编码由官方 C/Lua 解码，官方编码再由 Go 解码；默认测试使用已提交向量。
func TestOfficialOracle(t *testing.T) {
	dir := os.Getenv("SPROTO_ORACLE_DIR")
	if dir == "" {
		t.Skip("set SPROTO_ORACLE_DIR for official C/Lua cross-check")
	}
	script := `local sproto=require "sproto"
local sp=sproto.parse [[.Values { values 0 : *integer } .Double { value 0 : double }
.Numeric { integer 0 : integer values 1 : *integer double 2 : double doubles 3 : *double text 4 : string binary 5 : binary }]]
local function hex(s) return (s:gsub(".",function(c) return string.format("%02x",string.byte(c)) end)) end
local function unhex(s) return (s:gsub("..",function(c) return string.char(tonumber(c,16)) end)) end
local decoded=sp:decode("Values",unhex(arg[1]))
assert(decoded.values[1]==-1 and decoded.values[2]==-2147483648 and decoded.values[3]==42)
-- 独立从 Lua 数值编码，避免官方 todword 的低 32 位符号扩展缺陷污染向量。
local v={integer=math.mininteger, values={math.mininteger,-1,0,math.maxinteger},
double=0.01171875, doubles={0,-0.5,math.huge}, text="中文🙂", binary=string.char(0,255,1)}
print(hex(sp:encode("Numeric",v)))`
	type values struct {
		Values []int64 `sproto:"integer,0,array"`
	}
	encoded, err := sproto.Encode(&values{Values: []int64{-1, -2147483648, 42}})
	if err != nil {
		t.Fatal(err)
	}
	input := &numeric{Integer: math.MinInt64, Values: []int64{math.MinInt64, -1, 0, math.MaxInt64}, Double: 0.01171875, Doubles: []float64{0, -0.5, math.Inf(1)}, Text: "中文🙂", Binary: []byte{0, 255, 1}}
	full, err := sproto.Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command("lua", "-", hex.EncodeToString(encoded))
	command.Stdin = bytes.NewBufferString(script)
	command.Env = append(os.Environ(), "LUA_PATH="+dir+"/?.lua;;", "LUA_CPATH="+dir+"/?.so;;")
	result, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("official oracle: %v %s", err, result)
	}
	data, err := hex.DecodeString(string(bytes.TrimSpace(result)))
	if err != nil {
		t.Fatal(err)
	}
	var decoded numeric
	if _, err := sproto.Decode(data, &decoded); err != nil || !reflect.DeepEqual(input, &decoded) {
		t.Fatalf("official decode: %+v %v", decoded, err)
	}
	if !bytes.Equal(data, full) {
		t.Fatalf("official bytes differ: %x vs %x", data, full)
	}
}

// 验证有限坏包输入不会造成 panic 或越过解码预算。
func FuzzDecode(f *testing.F) {
	for _, seed := range []string{"010000000d00000004ffffffff000000802a000000", "0100000000000000", "01000000020000000001", "ffff", "0000"} {
		data, _ := hex.DecodeString(seed)
		f.Add(data)
	}
	seed, err := sproto.Encode(makeFuzzDocument([]byte{3, 1, 255}, 0x7ff8000000000042))
	if err != nil {
		f.Fatal(err)
	}
	f.Add(seed)
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			return
		}
		var output numeric
		_, _ = sproto.DecodeWithLimits(data, &output, sproto.Limits{MaxBytes: 4096, MaxDepth: 8, MaxElements: 128})
		var recursive fuzzDocument
		_, _ = sproto.DecodeWithLimits(data, &recursive, sproto.Limits{MaxBytes: 4096, MaxDepth: 8, MaxElements: 128})
		_, _ = sproto.UnpackWithLimit(data, 4096)
	})
}

// 验证小于最小报文的预算被拒绝，非法指针集合返回 schema 错误，具名字符串数组可安全往返。
func TestSchemaAndMinimumByteBudget(t *testing.T) {
	if _, err := sproto.EncodeWithLimits(&struct{}{}, sproto.Limits{MaxBytes: 1}); !errors.Is(err, sproto.ErrLimit) {
		t.Fatalf("minimum bytes: %v", err)
	}
	type pointerSlice struct {
		Values *[]int64 `sproto:"integer,0,array"`
	}
	if _, err := sproto.GetSprotoType(reflect.TypeOf(pointerSlice{})); err == nil {
		t.Fatal("pointer slice accepted")
	}
	if _, err := sproto.GetSprotoType(nil); err == nil {
		t.Fatal("nil schema accepted")
	}
	type text string
	type texts struct {
		Values []text  `sproto:"string,0,array"`
		Native *string `sproto:"string,1"`
		Named  *text   `sproto:"string,2"`
	}
	nativeText, namedText := "中文🙂", text("")
	input := &texts{Values: []text{"中文", ""}, Native: &nativeText, Named: &namedText}
	encoded, err := sproto.Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	var output texts
	if _, err := sproto.Decode(encoded, &output); err != nil || !reflect.DeepEqual(input, &output) {
		t.Fatalf("named string array: %+v %v", output, err)
	}
	type number int64
	type numbers []number
	type integerArrays struct {
		Native        []int64 `sproto:"integer,0,array"`
		Named         numbers `sproto:"integer,1,array"`
		NativePointer *int64  `sproto:"integer,2"`
		NamedPointer  *number `sproto:"integer,3"`
	}
	for _, values := range [][]int64{{}, {1, -1, math.MaxInt32}, {1, math.MaxInt64, -1}, {math.MinInt64, 1}} {
		input := &integerArrays{Native: append([]int64{}, values...), Named: make(numbers, len(values))}
		for index, value := range values {
			input.Named[index] = number(value)
		}
		var native int64
		if len(values) > 0 {
			native = values[0]
		}
		named := number(native)
		input.NativePointer, input.NamedPointer = &native, &named
		encoded, err := sproto.Encode(input)
		if err != nil {
			t.Fatal(err)
		}
		var output integerArrays
		if _, err := sproto.Decode(encoded, &output); err != nil || !reflect.DeepEqual(input, &output) {
			t.Fatalf("native/named integer arrays: %+v %v", output, err)
		}
	}
}

// 原生指针字段即使输入共用同一个指针，解码后的各字段及不同调用仍各自独占。
func TestNativePointerDecodeOwnership(t *testing.T) {
	type pointers struct {
		FirstInt   *int64  `sproto:"integer,0"`
		SecondInt  *int64  `sproto:"integer,1"`
		FirstText  *string `sproto:"string,2"`
		SecondText *string `sproto:"string,3"`
	}
	for _, number := range []int64{-42, 0, 1, 32766, 32767, math.MaxInt64} {
		text := "中文🙂"
		input := &pointers{
			FirstInt: &number, SecondInt: &number,
			FirstText: &text, SecondText: &text,
		}
		wire, err := sproto.Encode(input)
		if err != nil {
			t.Fatal(err)
		}
		var first, second pointers
		for _, output := range []*pointers{&first, &second} {
			if _, err := sproto.Decode(wire, output); err != nil || !reflect.DeepEqual(output, input) {
				t.Fatalf("decode pointers: %+v %v", output, err)
			}
			if output.FirstInt == output.SecondInt || output.FirstText == output.SecondText {
				t.Fatal("decoded fields share mutable pointers")
			}
		}
		if first.FirstInt == second.FirstInt || first.FirstText == second.FirstText {
			t.Fatal("decode calls share mutable pointers")
		}
		*first.FirstInt, *first.FirstText = 99, "changed"
		if *first.SecondInt != number || *first.SecondText != text || !reflect.DeepEqual(&second, input) {
			t.Fatal("changing one result mutated another field or decode call")
		}
	}
}

// 同一对象修改 presence、数组宽度和嵌套数量后，编码必须读取当前值且旧输出保持独占。
func TestEncodeTracksCurrentValues(t *testing.T) {
	type child struct {
		Value *int64 `sproto:"integer,0"`
	}
	type document struct {
		Flag     *bool    `sproto:"boolean,0"`
		Values   []int64  `sproto:"integer,1,array"`
		Children []*child `sproto:"struct,3,array"`
	}
	flag, number := true, int64(1)
	input := &document{Flag: &flag, Values: []int64{1, 2}, Children: []*child{{Value: &number}}}
	first, err := sproto.Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := append([]byte(nil), first...)
	for _, values := range [][]int64{{1, math.MaxInt64, -1}, {}, nil} {
		input.Flag = nil
		input.Values = values
		input.Children = append([]*child{nil, {}, {Value: &number}}, make([]*child, 96)...)
		encoded, err := sproto.Encode(input)
		if err != nil {
			t.Fatal(err)
		}
		var output document
		if _, err := sproto.Decode(encoded, &output); err != nil || !reflect.DeepEqual(input.Values, output.Values) ||
			output.Flag != nil || len(output.Children) != len(input.Children) || output.Children[0].Value != nil ||
			output.Children[1].Value != nil || output.Children[2].Value == nil || *output.Children[2].Value != number {
			t.Fatalf("changed message: %+v %v", output, err)
		}
	}
	if !bytes.Equal(first, snapshot) {
		t.Fatal("previous encoded output changed")
	}
}

// 验证字节、深度和元素预算恰好到边界时通过，少一个额度时失败且不改目标。
func TestCodecBudgetBoundaries(t *testing.T) {
	type text struct {
		Value string `sproto:"string,0"`
	}
	input := &text{Value: "ab"}
	encoded, err := sproto.EncodeWithLimits(input, sproto.Limits{MaxBytes: 10})
	if err != nil || len(encoded) != 10 {
		t.Fatalf("exact encode byte limit: %x %v", encoded, err)
	}
	if _, err := sproto.EncodeWithLimits(input, sproto.Limits{MaxBytes: 9}); !errors.Is(err, sproto.ErrLimit) {
		t.Fatalf("short encode byte limit: %v", err)
	}
	var output text
	if _, err := sproto.DecodeWithLimits(encoded, &output, sproto.Limits{MaxBytes: 10}); err != nil || output != *input {
		t.Fatalf("exact decode byte limit: %+v %v", output, err)
	}
	output.Value = "original"
	if _, err := sproto.DecodeWithLimits(encoded, &output, sproto.Limits{MaxBytes: 9}); !errors.Is(err, sproto.ErrLimit) || output.Value != "original" {
		t.Fatalf("short decode byte limit: %+v %v", output, err)
	}
	value := &tree{Next: &tree{}}
	encoded, err = sproto.EncodeWithLimits(value, sproto.Limits{MaxDepth: 2})
	if err != nil {
		t.Fatalf("exact encode depth: %v", err)
	}
	var decoded tree
	if _, err := sproto.DecodeWithLimits(encoded, &decoded, sproto.Limits{MaxDepth: 2}); err != nil || !reflect.DeepEqual(value, &decoded) {
		t.Fatalf("exact decode depth: %+v %v", decoded, err)
	}
	if _, err := sproto.EncodeWithLimits(value, sproto.Limits{MaxDepth: 1}); !errors.Is(err, sproto.ErrLimit) {
		t.Fatalf("short encode depth: %v", err)
	}
	if _, err := sproto.DecodeWithLimits(encoded, &decoded, sproto.Limits{MaxDepth: 1}); !errors.Is(err, sproto.ErrLimit) {
		t.Fatalf("short decode depth: %v", err)
	}
	type list struct {
		Values []int64 `sproto:"integer,0,array"`
	}
	items := &list{Values: []int64{-1, 2}}
	encoded, err = sproto.EncodeWithLimits(items, sproto.Limits{MaxElements: 3})
	if err != nil {
		t.Fatalf("exact encode elements: %v", err)
	}
	var decodedList list
	if _, err := sproto.DecodeWithLimits(encoded, &decodedList, sproto.Limits{MaxElements: 3}); err != nil || !reflect.DeepEqual(items, &decodedList) {
		t.Fatalf("exact decode elements: %+v %v", decodedList, err)
	}
	if _, err := sproto.EncodeWithLimits(items, sproto.Limits{MaxElements: 2}); !errors.Is(err, sproto.ErrLimit) {
		t.Fatalf("short encode elements: %v", err)
	}
	if _, err := sproto.DecodeWithLimits(encoded, &decodedList, sproto.Limits{MaxElements: 2}); !errors.Is(err, sproto.ErrLimit) {
		t.Fatalf("short decode elements: %v", err)
	}
	for _, limits := range []sproto.Limits{{MaxBytes: -1}, {MaxDepth: 129}, {MaxElements: -1}} {
		if err := limits.Validate(); !errors.Is(err, sproto.ErrInvalidLimits) {
			t.Fatalf("invalid limits: %+v %v", limits, err)
		}
	}
}

// 验证未知 inline、数据和嵌套字段可跳过，保留后续字段，新增缺失字段回到零值。
func TestSchemaEvolution(t *testing.T) {
	type nested struct {
		Value string `sproto:"string,0"`
	}
	type newer struct {
		ID       int64     `sproto:"integer,0"`
		Inline   int64     `sproto:"integer,1"`
		Data     []byte    `sproto:"string,2"`
		Nested   *nested   `sproto:"struct,3"`
		Children []*nested `sproto:"struct,4,array"`
		Tail     string    `sproto:"string,5"`
	}
	type older struct {
		ID   int64  `sproto:"integer,0"`
		Tail string `sproto:"string,5"`
	}
	input := &newer{ID: -1, Inline: 7, Data: []byte{0, 255}, Nested: &nested{Value: "new"}, Children: []*nested{{Value: "child"}}, Tail: "kept"}
	encoded, err := sproto.Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	var old older
	used, err := sproto.Decode(encoded, &old)
	if err != nil || used != len(encoded) || old.ID != input.ID || old.Tail != input.Tail {
		t.Fatalf("new to old: %+v consumed=%d error=%v", old, used, err)
	}
	encoded, err = sproto.Encode(&old)
	if err != nil {
		t.Fatal(err)
	}
	decoded := *input
	if _, err := sproto.Decode(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ID != old.ID || decoded.Tail != old.Tail || decoded.Inline != 0 || decoded.Data != nil || decoded.Nested != nil || decoded.Children != nil {
		t.Fatalf("old to new left stale fields: %+v", decoded)
	}
}

// 验证正负零、最小非正规数、无穷和不同 NaN payload 在标量、指针与数组中保留位模式。
func TestDoubleBitPatterns(t *testing.T) {
	type floats struct {
		Scalar  float64   `sproto:"double,0"`
		Pointer *float64  `sproto:"double,1"`
		Values  []float64 `sproto:"double,2,array"`
	}
	patterns := []uint64{0, 1 << 63, 1, 0x8000000000000001, 0x7fefffffffffffff, 0x7ff0000000000000, 0xfff0000000000000, 0x7ff8000000000001, 0x7ff8000000000042, 0x7ff0000000000001}
	for _, bits := range patterns {
		value := math.Float64frombits(bits)
		input := &floats{Scalar: value, Pointer: &value, Values: []float64{value}}
		encoded, err := sproto.Encode(input)
		if err != nil {
			t.Fatal(err)
		}
		var output floats
		if _, err := sproto.Decode(encoded, &output); err != nil {
			t.Fatal(err)
		}
		if output.Pointer == nil || len(output.Values) != 1 || math.Float64bits(output.Scalar) != bits || math.Float64bits(*output.Pointer) != bits || math.Float64bits(output.Values[0]) != bits {
			t.Fatalf("double bits changed: %016x -> %+v", bits, output)
		}
	}
}

// 同构整数记录数组与命名类型 wire 一致，稀疏 tag、空元素及预算边界不能因批量路径放宽。
func TestIntegerRecordArrayBudgetsAndTags(t *testing.T) {
	type record struct {
		First  *int64 `sproto:"integer,0"`
		Bridge *int64 `sproto:"integer,20000"`
		Last   *int64 `sproto:"integer,40000"`
	}
	type records struct {
		Items []*record `sproto:"struct,0,array"`
	}
	type number int64
	type namedRecord struct {
		First  *number `sproto:"integer,0"`
		Bridge *number `sproto:"integer,20000"`
		Last   *number `sproto:"integer,40000"`
	}
	type namedRecords struct {
		Items []*namedRecord `sproto:"struct,0,array"`
	}
	first, bridge, last := int64(-1), int64(0), int64(math.MaxInt64)
	namedFirst, namedBridge, namedLast := number(first), number(bridge), number(last)
	input := &records{Items: []*record{{First: &first, Bridge: &bridge, Last: &last}, nil, {}}}
	reference := &namedRecords{Items: []*namedRecord{{First: &namedFirst, Bridge: &namedBridge, Last: &namedLast}, nil, {}}}
	want, err := sproto.Encode(reference)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := sproto.EncodeWithLimits(input, sproto.Limits{MaxBytes: len(want), MaxDepth: 2, MaxElements: 7})
	if err != nil || !bytes.Equal(encoded, want) {
		t.Fatalf("integer records: %x != %x, %v", encoded, want, err)
	}
	for _, limits := range []sproto.Limits{{MaxBytes: len(want) - 1}, {MaxDepth: 1}, {MaxElements: 6}} {
		encoded, err := sproto.EncodeWithLimits(input, limits)
		if !errors.Is(err, sproto.ErrLimit) || encoded != nil {
			t.Fatalf("integer record budget %+v: %x, %v", limits, encoded, err)
		}
	}
	input.Items[0].Bridge = nil
	if encoded, err := sproto.Encode(input); !errors.Is(err, sproto.ErrEncode) || encoded != nil {
		t.Fatalf("missing tag bridge: %x, %v", encoded, err)
	}
}
