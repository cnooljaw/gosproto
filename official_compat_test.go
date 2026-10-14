package sproto_test

import (
	sproto "github.com/cnooljaw/gosproto"
	"reflect"
	"testing"
)

// 验证有符号整数数组通过候选库往返后保留负值。
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

// 验证候选库可建立 double 字段的编码类型。
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
