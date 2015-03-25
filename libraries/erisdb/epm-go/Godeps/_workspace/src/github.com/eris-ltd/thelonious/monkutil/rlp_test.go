package monkutil

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"math/big"
	"reflect"
	"testing"
)

func TestRlpValueEncoding(t *testing.T) {
	val := EmptyValue()
	val.AppendList().Append(1).Append(2).Append(3)
	val.Append("4").AppendList().Append(5)

	res := val.Encode()
	exp := Encode([]interface{}{[]interface{}{1, 2, 3}, "4", []interface{}{5}})
	if bytes.Compare(res, exp) != 0 {
		t.Errorf("expected %q, got %q", res, exp)
	}
}

func TestValueSlice(t *testing.T) {
	val := []interface{}{
		"value1",
		"valeu2",
		"value3",
	}

	value := NewValue(val)
	splitVal := value.SliceFrom(1)

	if splitVal.Len() != 2 {
		t.Error("SliceFrom: Expected len", 2, "got", splitVal.Len())
	}

	splitVal = value.SliceTo(2)
	if splitVal.Len() != 2 {
		t.Error("SliceTo: Expected len", 2, "got", splitVal.Len())
	}

	splitVal = value.SliceFromTo(1, 3)
	if splitVal.Len() != 2 {
		t.Error("SliceFromTo: Expected len", 2, "got", splitVal.Len())
	}
}

func TestLargeData(t *testing.T) {
	data := make([]byte, 100000)
	enc := Encode(data)
	value := NewValue(enc)
	value.Decode()

	if value.Len() != len(data) {
		t.Error("Expected data to be", len(data), "got", value.Len())
	}
}

func TestValue(t *testing.T) {
	value := NewValueFromBytes([]byte("\xcd\x83dog\x83god\x83cat\x01"))
	if value.Get(0).Str() != "dog" {
		t.Errorf("expected '%v', got '%v'", value.Get(0).Str(), "dog")
	}

	if value.Get(3).Uint() != 1 {
		t.Errorf("expected '%v', got '%v'", value.Get(3).Uint(), 1)
	}
}

func TestEncode(t *testing.T) {
	strRes := "\x83dog"
	bytes := Encode("dog")

	str := string(bytes)
	if str != strRes {
		t.Errorf("Expected %q, got %q", strRes, str)
	}

	sliceRes := "\xcc\x83dog\x83god\x83cat"
	strs := []interface{}{"dog", "god", "cat"}
	bytes = Encode(strs)
	slice := string(bytes)
	if slice != sliceRes {
		t.Error("Expected %q, got %q", sliceRes, slice)
	}

	intRes := "\x82\x04\x00"
	bytes = Encode(1024)
	if string(bytes) != intRes {
		t.Errorf("Expected %q, got %q", intRes, bytes)
	}
}

func TestDecodeDecompress(t *testing.T) {
	if !COMPRESS {
		return
	}
	toenc := []byte{0, 0, 0, 0, 0}
	expected := []byte{131, 0, 0, 5}
	enc := Encode(toenc)
	if bytes.Compare(enc, expected) != 0 {
		t.Errorf("Expected %v, got %v", expected, enc)
	}

	b, _ := Decode(enc, 0)
	fmt.Println(b)
	if bytes.Compare(b.([]byte), toenc) != 0 {
		t.Errorf("Expected %s, got %s", toenc, b)
	}

	toenc = []byte("create\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x00\x01\x00\x00")
	expected = []byte{141, 99, 114, 101, 97, 116, 101, 0, 0, 23, 1, 0, 0, 2}
	enc = Encode(toenc)
	if bytes.Compare(enc, expected) != 0 {
		t.Errorf("Expected %v, got %v", expected, enc)
	}

	b, _ = Decode(enc, 0)
	if bytes.Compare(b.([]byte), toenc) != 0 {
		t.Errorf("Expected %v, got %v", toenc, b)
	}

}

func TestDecode(t *testing.T) {
	single := []byte("\x01")
	b, _ := Decode(single, 0)

	if b.(uint8) != 1 {
		t.Errorf("Expected 1, got %q", b)
	}

	str := []byte("\x83dog")
	b, _ = Decode(str, 0)
	if bytes.Compare(b.([]byte), []byte("dog")) != 0 {
		t.Errorf("Expected dog, got %q", b)
	}

	slice := []byte("\xcc\x83dog\x83god\x83cat")
	res := []interface{}{"dog", "god", "cat"}
	b, _ = Decode(slice, 0)
	if reflect.DeepEqual(b, res) {
		t.Errorf("Expected %q, got %q", res, b)
	}
}

func TestEncodeDecodeBigInt(t *testing.T) {
	bigInt := big.NewInt(1391787038)
	encoded := Encode(bigInt)

	value := NewValueFromBytes(encoded)
	if value.BigInt().Cmp(bigInt) != 0 {
		t.Errorf("Expected %v, got %v", bigInt, value.BigInt())
	}
}

func TestEncodeDecodeBytes(t *testing.T) {
	b := NewValue([]interface{}{[]byte{1, 2, 3, 4, 5}, byte(6)})
	val := NewValueFromBytes(b.Encode())
	if !b.Cmp(val) {
		t.Errorf("Expected %v, got %v", val, b)
	}
}

func TestEncodeZero(t *testing.T) {
	b := NewValue(0).Encode()
	exp := []byte{0xc0}
	if bytes.Compare(b, exp) == 0 {
		t.Error("Expected", exp, "got", b)
	}
}

func randInt() int {
	one := make([]byte, 1)
	rand.Read(one)
	return int(one[0])
}

func makeTestData() *Value {
	dd := []interface{}{}
	for j := 0; j < 3; j++ {
		d := []byte{}
		for i := 0; i < 5; i++ {
			o := randInt() % 32
			m := make([]byte, o)
			rand.Read(m) // read a random number of bytes
			d = append(d, m...)
			o = randInt() % 32
			d = append(d, bytes.Repeat([]byte{0}, o)...) // add a random number of zeros
		}
		dd = append(dd, d)
	}
	return NewValue(dd)
}

//var prefix = string(bytes.Repeat([]byte{0}, 100))
//var testValues = []interface{}{prefix+"dog", prefix+"cat", prefix+"god"}
var testValues = makeTestData()

func BenchmarkEncode(b *testing.B) {
	COMPRESS = false
	for i := 0; i < b.N; i++ {
		testValues.Encode() //[]interface{}{"dog", "god", "cat"})
	}
}

func BenchmarkEncodeCompress(b *testing.B) {
	COMPRESS = true
	for i := 0; i < b.N; i++ {
		testValues.Encode() //[]interface{}{"dog", "god", "cat"})
	}
}

func BenchmarkDecode(b *testing.B) {
	COMPRESS = false
	by := Encode(testValues)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decode(by, 0)
	}
}

func BenchmarkDecodeCompress(b *testing.B) {
	COMPRESS = true
	by := testValues.Encode()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Decode(by, 0)
	}
}

func BenchmarkEncodeDecode(b *testing.B) {
	COMPRESS = false
	for i := 0; i < b.N; i++ {
		bytes := testValues.Encode() //[]interface{}{"dog", "god", "cat"})
		Decode(bytes, 0)
	}
}

func BenchmarkEncodeDecodeCompress(b *testing.B) {
	COMPRESS = true
	for i := 0; i < b.N; i++ {
		bytes := testValues.Encode() //[]interface{}{"dog", "god", "cat"})
		Decode(bytes, 0)
	}
}
