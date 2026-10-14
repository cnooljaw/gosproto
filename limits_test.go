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
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 4096 {
			return
		}
		var output numeric
		_, _ = sproto.DecodeWithLimits(data, &output, sproto.Limits{MaxBytes: 4096, MaxDepth: 8, MaxElements: 128})
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
		Values []text `sproto:"string,0,array"`
	}
	input := &texts{Values: []text{"中文", ""}}
	encoded, err := sproto.Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	var output texts
	if _, err := sproto.Decode(encoded, &output); err != nil || !reflect.DeepEqual(input, &output) {
		t.Fatalf("named string array: %+v %v", output, err)
	}
}
