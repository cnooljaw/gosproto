package sproto

import (
	"bytes"
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

// 不影响已有功能测试：Ptr结构可以被编码
func TestPtrEncode(t *testing.T) {
	Reset()
	ptrMsgData, err := Encode(&ptrMsg)
	if err != nil {
		t.Error(err)
		return
	}
	// 测试解包结果
	Reset()
	ptrMsg2 := PtrMSG{}
	Decode(ptrMsgData, &ptrMsg2)
	if !reflect.DeepEqual(ptrMsg, ptrMsg2) {
		t.Error("ptrMsg is not equal to ptrMsg2")
	}
}

// 不影响已有功能测试：Ptr结构可以在包含nil的情况下被编码
func TestPtrNilEncode(t *testing.T) {
	// 测试对nil值的支持
	Reset()
	ptrMsg.Int = nil
	ptrMsg.Bool = nil
	ptrMsg.StructSlice = nil
	ptrMsg.Struct = nil
	ptrMsgData, err := Encode(&ptrMsg)
	if err != nil {
		t.Error(err)
		return
	}

	// 测试解包结果
	ptrMsg2 := PtrMSG{}
	Decode(ptrMsgData, &ptrMsg2)
	if !reflect.DeepEqual(ptrMsg, ptrMsg2) {
		t.Error("ptrMsg is not equal to ptrMsg2")
	}
}

// 新特性测试: 等价的Val结构和Ptr结构编码结果一致（不考虑ptr为nil）
func TestValueEncodeEqualToPtr(t *testing.T) {
	Reset()
	valMsgData, err := Encode(&valMSG)
	if err != nil {
		t.Error(err, valMsgData)
		return
	}

	ptrMsgData, err := Encode(&ptrMsg)
	if err != nil {
		t.Error(err)
		return
	}
	if !reflect.DeepEqual(valMsgData, ptrMsgData) {
		t.Error("ValMsgData exprect equal to PtrMsgData")
		return
	}

	Reset()
	// 预期val编码结果应该允许被等价结构的含ptr结构体接收
	ptrMsg2 := PtrMSG{}
	Decode(valMsgData, &ptrMsg2)
	if !reflect.DeepEqual(ptrMsg2, ptrMsg) {
		t.Error("预期val编码结果应该允许被等价结构的含ptr结构体接收")
	}
}

// 新特性测试:预期valMsgData可以被相同的val结构体接收
func TestValueDecode(t *testing.T) {
	Reset()
	valMsgData, err := Encode(&valMSG)
	if err != nil {
		t.Error(err, valMsgData)
		return
	}

	// 预期valMsgData可以被相同的val结构体接收
	valMsg2 := ValMSG{}
	_, err = Decode(valMsgData, &valMsg2)
	if err != nil {
		t.Error(err)
		return
	}

	// 预期结果应该保持一致
	if !reflect.DeepEqual(valMSG, valMsg2) {
		t.Error("ValMsg expect equal to ValMsg2")
		return
	}
}

// 新特性测试：预期ptrMsgData可以被等价的Val结构体接收（补充测试）
func TestValueEncodeToPtr(t *testing.T) {
	Reset()
	ptrMsgData, err := Encode(&ptrMsg)
	if err != nil {
		t.Error(err)
		return
	}

	// 预期可以被等价的val结构体接收
	valMsg2 := ValMSG{}
	_, err = Decode(ptrMsgData, &valMsg2)
	if err != nil {
		t.Error(err)
		return
	}

	// 通过比对Json结果间接比对接收结果
	ptrJson, _ := json.Marshal(ptrMsg)
	valJson, _ := json.Marshal(valMsg2)
	if !bytes.Equal(ptrJson, valJson) {
		t.Error("Expect ptrJson Euqal To valJson")
		return
	}
}

// 新特性测试：包含部分等价nil的消息可以被正确接收并设置为默认值
func TestValueDecodeNil(t *testing.T) {
	// 测试对nil值的支持
	Reset()
	ptrMsg.Int = nil
	ptrMsg.Bool = nil
	ptrMsg.StructSlice = nil
	ptrMsg.Struct = nil
	ptrMsgData, err := Encode(&ptrMsg)
	if err != nil {
		t.Error(err)
		return
	}

	// 测试Val对nil的接收
	valMsg2 := ValMSG{Bool: true, Int: 1024}
	_, err = Decode(ptrMsgData, &valMsg2)
	if err != nil {
		t.Error(err)
		return
	}
	// 测试默认值
	if valMsg2.Bool != false {
		t.Error("Expect Bool equal to false")
		return
	}
	if valMsg2.Int != 0 {
		t.Error("Expect Int equal to 0")
		return
	}
}

// 与原装的sproto进行比对，看编码结果是否一致
func TestOldAndNewEncode(t *testing.T) {
	Reset()
	// 云风的版本
	oldMsgData, err := Encode(&ptrMsg)
	if err != nil {
		t.Error(err)
	}
	// 新版本
	newMsgData, err := Encode(&ptrMsg)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(oldMsgData, newMsgData) {
		t.Error("新方案与旧方案的编码结果不一致。")
	}

	Reset()
	// 云风的版本
	ptrMsg.Int = nil
	ptrMsg.String = nil
	oldMsgData, err = Encode(&ptrMsg)
	if err != nil {
		t.Error(err)
	}
	// 新版本
	newMsgData, err = Encode(&ptrMsg)
	if err != nil {
		t.Error(err)
	}
	if !reflect.DeepEqual(oldMsgData, newMsgData) {
		t.Error("包含nil的情况下新方案与旧方案的编码结果不一致。")
	}
}

type PtrMSG struct {
	Int         *int          `sproto:"integer,0,name=Int"`
	String      *string       `sproto:"string,1,name=String"`
	Bool        *bool         `sproto:"boolean,2,name=Bool"`
	Struct      *HoldPtrMSG   `sproto:"struct,3,name=Struct"`
	ByteSlice   []byte        `sproto:"string,4,name=ByteSlice"`
	BoolSlice   []bool        `sproto:"boolean,5,array,name=BoolSlice"`
	IntSlice    []int         `sproto:"integer,6,array,name=IntSlice"`
	StringSlice []string      `sproto:"string,7,array,name=StringSlice"`
	StructSlice []*HoldPtrMSG `sproto:"struct,8,array,name=StructSlice"`
	IntNeg      *int          `sproto:"integer,9,name=IntNeg"`
}

type HoldPtrMSG struct {
	Int         *int     `sproto:"integer,0,name=Int"`
	String      *string  `sproto:"string,1,name=String"`
	Bool        *bool    `sproto:"boolean,2,name=Bool"`
	ByteSlice   []byte   `sproto:"string,3,name=ByteSlice"`
	BoolSlice   []bool   `sproto:"boolean,4,array,name=BoolSlice"`
	IntSlice    []int    `sproto:"integer,5,array,name=IntSlice"`
	StringSlice []string `sproto:"string,6,array,name=StringSlice"`
}

var ptrMsg PtrMSG

func Reset() {
	ptrMsg = PtrMSG{
		Int:         Int(math.MaxInt64),
		IntNeg:      Int(math.MinInt32 + 1),
		String:      String("Hello"),
		Bool:        Bool(true),
		ByteSlice:   []byte("World"),
		BoolSlice:   []bool{true, true, false, true, false},
		IntSlice:    []int{123, 321, 1234567},
		StringSlice: []string{"FOO", "BAR"},
		Struct: &HoldPtrMSG{
			Int:         Int(1),
			String:      String("Hello"),
			Bool:        Bool(true),
			ByteSlice:   []byte("World"),
			BoolSlice:   []bool{true, true, false, true, false},
			IntSlice:    []int{123, 321, 1234567},
			StringSlice: []string{"FOO", "BAR"},
		},
		StructSlice: []*HoldPtrMSG{
			&HoldPtrMSG{
				Int:         Int(2),
				String:      String("Hello2"),
				Bool:        Bool(true),
				ByteSlice:   []byte("World2"),
				BoolSlice:   []bool{true, true, false, true, false},
				IntSlice:    []int{123, 321, 1234567},
				StringSlice: []string{"FOO2", "BAR2"},
			},
			&HoldPtrMSG{
				Int:         Int(3),
				String:      String("Hello3"),
				Bool:        Bool(true),
				ByteSlice:   []byte("World3"),
				BoolSlice:   []bool{true, true, false, true, false},
				IntSlice:    []int{123, 321, 1234567},
				StringSlice: []string{"FOO3", "BAR3"},
			},
		},
	}
	valMSG = ValMSG{
		Int:         math.MaxInt64,
		IntNeg:      math.MinInt32 + 1,
		String:      "Hello",
		Bool:        true,
		ByteSlice:   []byte("World"),
		BoolSlice:   []bool{true, true, false, true, false},
		IntSlice:    []int{123, 321, 1234567},
		StringSlice: []string{"FOO", "BAR"},
		Struct: &HoldValMSG{
			Int:         1,
			String:      "Hello",
			Bool:        true,
			ByteSlice:   []byte("World"),
			BoolSlice:   []bool{true, true, false, true, false},
			IntSlice:    []int{123, 321, 1234567},
			StringSlice: []string{"FOO", "BAR"},
		},
		StructSlice: []*HoldValMSG{
			&HoldValMSG{
				Int:         2,
				String:      "Hello2",
				Bool:        true,
				ByteSlice:   []byte("World2"),
				BoolSlice:   []bool{true, true, false, true, false},
				IntSlice:    []int{123, 321, 1234567},
				StringSlice: []string{"FOO2", "BAR2"},
			},
			&HoldValMSG{
				Int:         3,
				String:      "Hello3",
				Bool:        true,
				ByteSlice:   []byte("World3"),
				BoolSlice:   []bool{true, true, false, true, false},
				IntSlice:    []int{123, 321, 1234567},
				StringSlice: []string{"FOO3", "BAR3"},
			},
		},
	}
}

type ValMSG struct {
	Int         int           `sproto:"integer,0,name=Int"`
	String      string        `sproto:"string,1,name=String"`
	Bool        bool          `sproto:"boolean,2,name=Bool"`
	Struct      *HoldValMSG   `sproto:"struct,3,name=Struct"`
	ByteSlice   []byte        `sproto:"string,4,name=ByteSlice"`
	BoolSlice   []bool        `sproto:"boolean,5,array,name=BoolSlice"`
	IntSlice    []int         `sproto:"integer,6,array,name=IntSlice"`
	StringSlice []string      `sproto:"string,7,array,name=StringSlice"`
	StructSlice []*HoldValMSG `sproto:"struct,8,array,name=StructSlice"`
	IntNeg      int           `sproto:"integer,9,name=IntNeg"`
}

type HoldValMSG struct {
	Int         int      `sproto:"integer,0,name=Int"`
	String      string   `sproto:"string,1,name=String"`
	Bool        bool     `sproto:"boolean,2,name=Bool"`
	ByteSlice   []byte   `sproto:"string,3,name=ByteSlice"`
	BoolSlice   []bool   `sproto:"boolean,4,array,name=BoolSlice"`
	IntSlice    []int    `sproto:"integer,5,array,name=IntSlice"`
	StringSlice []string `sproto:"string,6,array,name=StringSlice"`
}

var valMSG ValMSG

// 原生指针在填充字段和嵌套结构后仍与命名整数产生相同 wire，且每次读取当前值。
func TestEncodeNativePointerOffsets(t *testing.T) {
	type number int64
	type native struct {
		ignored [19]byte
		Value   *int64  `sproto:"integer,0"`
		Child   *native `sproto:"struct,2"`
	}
	type named struct {
		Value *number `sproto:"integer,0"`
		Child *named  `sproto:"struct,2"`
	}
	var nativeValue int64
	var namedValue number
	input := &native{Value: &nativeValue, Child: &native{Value: &nativeValue}}
	reference := &named{Value: &namedValue, Child: &named{Value: &namedValue}}
	for _, value := range []int64{0, MaxEmbeddedInt, MaxEmbeddedInt + 1, -1, MinInt32, MinInt32 - 1, math.MinInt64, math.MaxInt64} {
		nativeValue, namedValue = value, number(value)
		encoded, err := Encode(input)
		if err != nil {
			t.Fatal(err)
		}
		want, err := Encode(reference)
		if err != nil || !bytes.Equal(encoded, want) {
			t.Fatalf("value %d: native %x, named %x, error %v", value, encoded, want, err)
		}
	}
	input.Value, input.Child.Value = nil, nil
	reference.Value, reference.Child.Value = nil, nil
	encoded, err := Encode(input)
	want, referenceErr := Encode(reference)
	if err != nil || referenceErr != nil || !bytes.Equal(encoded, want) {
		t.Fatalf("nil fields: native %x, named %x, errors %v/%v", encoded, want, err, referenceErr)
	}
}

// 替换导出的字段描述时按当前字段索引取值，不借用另一个结构体的内存位置。
func TestEncodeReplacedFieldMetadata(t *testing.T) {
	type small struct {
		ignored byte
		Value   *int64 `sproto:"integer,0"`
	}
	type large struct {
		ignored [256]byte
		Value   *int64 `sproto:"integer,0"`
	}
	value := int64(-42)
	input := &small{Value: &value}
	want, err := Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := GetSprotoType(reflect.TypeFor[small]())
	if err != nil {
		t.Fatal(err)
	}
	other, err := GetSprotoType(reflect.TypeFor[large]())
	if err != nil {
		t.Fatal(err)
	}
	original := schema.Fields[1]
	schema.Fields[1] = other.Fields[1]
	t.Cleanup(func() { schema.Fields[1] = original })
	encoded, err := Encode(input)
	if err != nil || !bytes.Equal(encoded, want) {
		t.Fatalf("replaced field: %x != %x, %v", encoded, want, err)
	}
}

// 公开 Wire/Array 描述变化时编码仍使用当前描述，不复用之前绑定的标量操作。
func TestEncodeChangedFieldShape(t *testing.T) {
	type binary struct {
		Value []byte `sproto:"string,0"`
	}
	type integers struct {
		Value []uint8 `sproto:"integer,0,array"`
	}
	input := &binary{Value: []byte{0, 42, 255}}
	if _, err := Encode(input); err != nil {
		t.Fatal(err)
	}
	schema, err := GetSprotoType(reflect.TypeFor[binary]())
	if err != nil {
		t.Fatal(err)
	}
	field := schema.Fields[0]
	field.Wire, field.Array = WireVarintName, true
	t.Cleanup(func() { field.Wire, field.Array = WireBytesName, false })
	encoded, err := Encode(input)
	want, referenceErr := Encode(&integers{Value: input.Value})
	if err != nil || referenceErr != nil || !bytes.Equal(encoded, want) {
		t.Fatalf("changed shape: %x != %x, errors %v/%v", encoded, want, err, referenceErr)
	}
}

// 嵌套字段描述替换后，子 schema 的字段索引仍作用于实际 Go 值，不能借用另一种布局的 offset。
func TestEncodeReplacedNestedMetadata(t *testing.T) {
	type actualChild struct {
		ignored [32]byte
		Value   *int64 `sproto:"integer,0"`
	}
	type describedChild struct {
		ignored byte
		Value   *int64 `sproto:"integer,0"`
	}
	type actualParent struct {
		Child    *actualChild   `sproto:"struct,0"`
		Children []*actualChild `sproto:"struct,1,array"`
	}
	type describedParent struct {
		Child    *describedChild   `sproto:"struct,0"`
		Children []*describedChild `sproto:"struct,1,array"`
	}
	number := int64(42)
	input := &actualParent{Child: &actualChild{Value: &number}, Children: []*actualChild{{Value: &number}, nil}}
	want, err := Encode(input)
	if err != nil {
		t.Fatal(err)
	}
	schema, err := GetSprotoType(reflect.TypeFor[actualParent]())
	if err != nil {
		t.Fatal(err)
	}
	described, err := GetSprotoType(reflect.TypeFor[describedParent]())
	if err != nil {
		t.Fatal(err)
	}
	for index, name := range []string{"child", "children"} {
		t.Run(name, func(t *testing.T) {
			original := schema.Fields[index]
			schema.Fields[index] = described.Fields[index]
			t.Cleanup(func() { schema.Fields[index] = original })
			encoded, err := Encode(input)
			if err != nil || !bytes.Equal(encoded, want) {
				t.Fatalf("nested schema replacement: %x != %x, %v", encoded, want, err)
			}
		})
	}
	t.Run("child_fields", func(t *testing.T) {
		actual, err := GetSprotoType(reflect.TypeFor[actualChild]())
		if err != nil {
			t.Fatal(err)
		}
		other, err := GetSprotoType(reflect.TypeFor[describedChild]())
		if err != nil {
			t.Fatal(err)
		}
		original := actual.Fields[1]
		actual.Fields[1] = other.Fields[1]
		t.Cleanup(func() { actual.Fields[1] = original })
		encoded, err := Encode(input)
		if err != nil || !bytes.Equal(encoded, want) {
			t.Fatalf("nested field replacement: %x != %x, %v", encoded, want, err)
		}
	})
}
