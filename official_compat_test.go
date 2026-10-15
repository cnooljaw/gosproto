package sproto_test

import (
	"bytes"
	"encoding/hex"
	"os"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	sproto "github.com/cnooljaw/gosproto"
)

// 验证有符号整数数组通过编码器往返后保留负值。
func TestSignedIntegerArray(t *testing.T) {
	type request struct {
		Values []int64 `sproto:"integer,0,array"`
	}
	input := &request{Values: []int64{-1, -2147483648, 42}}
	encoded, err := sproto.Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	var output request
	if _, err := sproto.Decode(encoded, &output); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(input.Values, output.Values) {
		t.Fatalf("want %v, got %v", input.Values, output.Values)
	}
}

// 验证编码器可建立 double 字段的编码类型。
func TestDoubleField(t *testing.T) {
	type request struct {
		Value float64 `sproto:"double,0"`
	}
	if _, err := sproto.GetSprotoType(reflect.TypeOf(request{})); err != nil {
		t.Fatal(err)
	}
}

// 验证畸形整数数组返回错误而不是触发 panic。
func TestMalformedIntegerArray(t *testing.T) {
	type request struct {
		Values []int64 `sproto:"integer,0,array"`
	}
	defer func() {
		if cause := recover(); cause != nil {
			t.Errorf("decode panicked: %v", cause)
		}
	}()
	var output request
	_, err := sproto.Decode([]byte{1, 0, 0, 0, 2, 0, 0, 0, 0, 1}, &output)
	if err == nil {
		t.Fatal("malformed integer array accepted")
	}
}

// 验证官方四字节负整数标量解码后仍为负值。
func TestOfficialNegativeScalar(t *testing.T) {
	type request struct {
		Value int64 `sproto:"integer,0"`
	}
	var output request
	if _, err := sproto.Decode([]byte{1, 0, 0, 0, 4, 0, 0, 0, 255, 255, 255, 255}, &output); err != nil {
		t.Fatal(err)
	}
	if output.Value != -1 {
		t.Fatalf("want -1, got %d", output.Value)
	}
}

// 验证官方空整数数组的规范编码可被解码。
func TestOfficialEmptyArray(t *testing.T) {
	type request struct {
		Values []int64 `sproto:"integer,0,array"`
	}
	var output request
	if _, err := sproto.Decode([]byte{1, 0, 0, 0, 0, 0, 0, 0}, &output); err != nil {
		t.Fatal(err)
	}
	if len(output.Values) != 0 {
		t.Fatalf("want empty array, got %v", output.Values)
	}
}

type oracleChild struct {
	Number *string `sproto:"string,99"`
	Kind   *int64  `sproto:"integer,1000"`
}

type oracleCollection struct {
	Name     string         `sproto:"string,0"`
	Names    []string       `sproto:"string,1,array"`
	Flags    []bool         `sproto:"boolean,2,array"`
	Children []*oracleChild `sproto:"struct,3,array"`
	Single   *oracleChild   `sproto:"struct,4"`
	Binary   []byte         `sproto:"string,5"`
}

// 验证官方集合与稀疏字段向量，并可选地实际交叉检查 C/Lua 编解码及 pack/unpack。
func TestOfficialCollections(t *testing.T) {
	input := &oracleCollection{
		Name: "中文🙂", Names: []string{"ABC", "", "def"}, Flags: []bool{false, true, false},
		Children: []*oracleChild{{Number: sproto.String("123"), Kind: sproto.Int64(1)}, {}},
		Single:   &oracleChild{Kind: sproto.Int64(-1)}, Binary: []byte{0, 255, 1},
	}
	encoded, err := sproto.Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := hex.DecodeString("06000000000000000000000000000a000000e4b8ade69687f09f998212000000030000004142430000000003000000646566030000000001001b000000110000000400c500000007070400030000003132330200000000000e0000000200cf07000004000000ffffffff0300000000ff01")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, expected) {
		t.Fatalf("official collection bytes differ: %x", encoded)
	}
	var output oracleCollection
	if used, err := sproto.Decode(expected, &output); err != nil || used != len(expected) || !reflect.DeepEqual(input, &output) {
		t.Fatalf("official collection decode: %+v consumed=%d error=%v", output, used, err)
	}
	t.Run("C-Lua", func(t *testing.T) {
		dir := os.Getenv("SPROTO_ORACLE_DIR")
		if dir == "" {
			t.Skip("set SPROTO_ORACLE_DIR for official C/Lua cross-check")
		}
		script := `local sproto=require "sproto"
local sp=sproto.parse [[.Child { number 99 : string kind 1000 : integer }
.Collection { name 0 : string names 1 : *string flags 2 : *boolean children 3 : *Child single 4 : Child binary 5 : binary }]]
local function hex(s) return (s:gsub(".",function(c) return string.format("%02x",string.byte(c)) end)) end
local function unhex(s) return (s:gsub("..",function(c) return string.char(tonumber(c,16)) end)) end
local decoded=sp:decode("Collection",unhex(arg[1]))
assert(decoded.name=="中文🙂" and decoded.names[1]=="ABC" and decoded.names[2]=="" and decoded.names[3]=="def")
assert(decoded.flags[1]==false and decoded.flags[2]==true and decoded.flags[3]==false)
assert(#decoded.children==2 and decoded.children[1].number=="123" and decoded.children[1].kind==1 and decoded.children[2].number==nil)
assert(decoded.single.kind==-1 and decoded.binary==string.char(0,255,1))
local value={name="中文🙂",names={"ABC","","def"},flags={false,true,false},children={{number="123",kind=1},{}},single={kind=-1},binary=string.char(0,255,1)}
local encoded=sp:encode("Collection",value)
print(hex(encoded))
print(hex(sproto.pack(encoded)))
print(hex(sproto.unpack(unhex(arg[2]))))
for index=3,#arg,2 do
  local source=unhex(arg[index])
  local packed=sproto.pack(source)
  assert(hex(packed)==arg[index+1], "packing boundary length="..#source)
  local expected=source..string.rep(string.char(0),(8-#source%8)%8)
  assert(sproto.unpack(packed)==expected, "unpacking boundary length="..#source)
end`
		packed := sproto.Pack(encoded)
		arguments := []string{"-", hex.EncodeToString(encoded), hex.EncodeToString(packed)}
		for _, length := range []int{2046, 2047, 2048, 2049, 4094, 4095, 4096} {
			source := bytes.Repeat([]byte{1}, length)
			arguments = append(arguments, hex.EncodeToString(source), hex.EncodeToString(sproto.Pack(source)))
		}
		command := exec.Command("lua", arguments...)
		command.Stdin = bytes.NewBufferString(script)
		command.Env = append(os.Environ(), "LUA_PATH="+dir+"/?.lua;;", "LUA_CPATH="+dir+"/?.so;;")
		result, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("official collection oracle: %v %s", err, result)
		}
		unpacked, err := sproto.Unpack(packed)
		if err != nil {
			t.Fatal(err)
		}
		lines := strings.Fields(string(result))
		if len(lines) != 3 {
			t.Fatalf("unexpected oracle output: %s", result)
		}
		for index, expected := range [][]byte{encoded, packed, unpacked} {
			actual, err := hex.DecodeString(lines[index])
			if err != nil || !bytes.Equal(actual, expected) {
				t.Fatalf("oracle result %d: %s expected=%x error=%v", index, lines[index], expected, err)
			}
		}
	})
}
