package bencode

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

type any interface{}

func checkMarshal(expected string, data any) (err error) {
	var b bytes.Buffer
	if err = Marshal(&b, data); err != nil {
		return
	}
	s := b.String()
	if expected != s {
		err = fmt.Errorf("Expected %s got %s", expected, s)
		return
	}
	return
}

func check(expected string, data any) (err error) {
	if err = checkMarshal(expected, data); err != nil {
		return
	}
	b2 := bytes.NewBufferString(expected)
	val, err := Decode(b2)
	if err != nil {
		err = errors.New(fmt.Sprint("Failed decoding ", expected, " ", err))
		return
	}
	if err = checkFuzzyEqual(data, val); err != nil {
		return
	}
	return
}

func checkFuzzyEqual(a any, b any) (err error) {
	if !fuzzyEqual(a, b) {
		err = errors.New(fmt.Sprint(a, " != ", b,
			": ", reflect.ValueOf(a), "!=", reflect.ValueOf(b)))
	}
	return
}

func fuzzyEqual(a, b any) bool {
	return fuzzyEqualValue(reflect.ValueOf(a), reflect.ValueOf(b))
}

func checkFuzzyEqualValue(a, b reflect.Value) (err error) {
	if !fuzzyEqualValue(a, b) {
		err = fmt.Errorf("Wanted %v(%v) got %v(%v)", a, a.Interface(), b, b.Interface())
	}
	return
}

func fuzzyEqualInt64(a int64, b reflect.Value) bool {
	switch vb := b; vb.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return a == (vb.Int())
	}
	return false
}

func fuzzyEqualArrayOrSlice(va reflect.Value, b reflect.Value) bool {
	switch vb := b; vb.Kind() {
	case reflect.Array:
		return fuzzyEqualArrayOrSlice2(va, vb)
	case reflect.Slice:
		return fuzzyEqualArrayOrSlice2(va, vb)
	}
	return false
}

func deInterface(a reflect.Value) reflect.Value {
	switch va := a; va.Kind() {
	case reflect.Interface:
		return va.Elem()
	}
	return a
}

func fuzzyEqualArrayOrSlice2(a reflect.Value, b reflect.Value) bool {
	if a.Len() != b.Len() {
		return false
	}

	for i := 0; i < a.Len(); i++ {
		ea := deInterface(a.Index(i))
		eb := deInterface(b.Index(i))
		if !fuzzyEqualValue(ea, eb) {
			return false
		}
	}
	return true
}

func fuzzyEqualMap(a reflect.Value, b reflect.Value) bool {
	key := a.Type().Key()
	if key.Kind() != reflect.String {
		return false
	}
	key = b.Type().Key()
	if key.Kind() != reflect.String {
		return false
	}

	aKeys, bKeys := a.MapKeys(), b.MapKeys()

	if len(aKeys) != len(bKeys) {
		return false
	}

	for _, k := range aKeys {
		if !fuzzyEqualValue(a.MapIndex(k), b.MapIndex(k)) {
			return false
		}
	}
	return true
}

func fuzzyEqualStruct(a reflect.Value, b reflect.Value) bool {
	numA, numB := a.NumField(), b.NumField()
	if numA != numB {
		return false
	}

	for i := 0; i < numA; i++ {
		if !fuzzyEqualValue(a.Field(i), b.Field(i)) {
			return false
		}
	}
	return true
}

func fuzzyEqualValue(a, b reflect.Value) bool {
	switch va := a; va.Kind() {
	case reflect.String:
		switch vb := b; vb.Kind() {
		case reflect.String:
			return va.String() == vb.String()
		default:
			return false
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fuzzyEqualInt64(va.Int(), b)
	case reflect.Array:
		return fuzzyEqualArrayOrSlice(va, b)
	case reflect.Slice:
		return fuzzyEqualArrayOrSlice(va, b)
	case reflect.Map:
		switch vb := b; vb.Kind() {
		case reflect.Map:
			return fuzzyEqualMap(va, vb)
		default:
			return false
		}
	case reflect.Struct:
		switch vb := b; vb.Kind() {
		case reflect.Struct:
			return fuzzyEqualStruct(va, vb)
		default:
			return false
		}
	case reflect.Interface:
		switch vb := b; vb.Kind() {
		case reflect.Interface:
			return fuzzyEqualValue(va.Elem(), vb.Elem())
		default:
			return false
		}
	}
	return false
}

func checkUnmarshal(expected string, data any) (err error) {
	dataValue := reflect.ValueOf(data)
	newOne := reflect.New(reflect.TypeOf(data))
	buf := bytes.NewBufferString(expected)
	if err = unmarshalValue(buf, newOne); err != nil {
		return
	}
	if err = checkFuzzyEqualValue(dataValue, newOne.Elem()); err != nil {
		return
	}
	return
}

type SVPair struct {
	s string
	v any
}

var decodeTests = []SVPair{
	SVPair{"i0e", int64(0)},
	SVPair{"i0e", 0},
	SVPair{"i100e", 100},
	SVPair{"i-100e", -100},
	SVPair{"1:a", "a"},
	SVPair{"2:a\"", "a\""},
	SVPair{"11:0123456789a", "0123456789a"},
	SVPair{"le", []int64{}},
	SVPair{"li1ei2ee", []int{1, 2}},
	SVPair{"l3:abc3:defe", []string{"abc", "def"}},
	SVPair{"li42e3:abce", []any{42, "abc"}},
	SVPair{"de", map[string]any{}},
	SVPair{"d3:cati1e3:dogi2ee", map[string]any{"cat": 1, "dog": 2}},
}

func TestDecode(t *testing.T) {
	for _, sv := range decodeTests {
		if err := check(sv.s, sv.v); err != nil {
			t.Error(err.Error())
		}
	}
}

func BenchmarkDecodeAll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, sv := range decodeTests {
			check(sv.s, sv.v)
		}
	}
}

type structA struct {
	A int    "a"
	B string `example:"data" bencode:"b"`
	C string `example:"data2" bencode:"sea monster"`
}

type structNested struct {
	T string            "t"
	Y string            "y"
	Q string            "q"
	A map[string]string "a"
}

var (
	unmarshalInnerDict        = map[string]string{"id": "abcdefghij0123456789"}
	unmarshalNestedDictionary = structNested{"aa", "q", "ping", unmarshalInnerDict}
	unmarshalTests            = []SVPair{
		SVPair{"i100e", 100},
		SVPair{"i-100e", -100},
		SVPair{"i7.5e", 7},
		SVPair{"i-7.5e", -7},
		SVPair{"i7.574E+2e", 757},
		SVPair{"i-7.574E+2e", -757},
		SVPair{"i7.574E+20e", -9223372036854775808},
		SVPair{"i-7.574E+20e", -9223372036854775808},
		SVPair{"i7.574E-2e", 0},
		SVPair{"i-7.574E-2e", 0},
		SVPair{"i7.574E-20e", 0},
		SVPair{"i-7.574E-20e", 0},
		SVPair{"1:a", "a"},
		SVPair{"2:a\"", "a\""},
		SVPair{"11:0123456789a", "0123456789a"},
		SVPair{"le", []int64{}},
		SVPair{"li1ei2ee", []int{1, 2}},
		SVPair{"l3:abc3:defe", []string{"abc", "def"}},
		SVPair{"li42e3:abce", []any{42, "abc"}},
		SVPair{"de", map[string]any{}},
		SVPair{"d3:cati1e3:dogi2ee", map[string]any{"cat": 1, "dog": 2}},
		SVPair{"d1:ai10e1:b3:foo11:sea monster3:bare", structA{10, "foo", "bar"}},
		SVPair{"d1:ad2:id20:abcdefghij0123456789e1:q4:ping1:t2:aa1:y1:qe", unmarshalNestedDictionary},
	}
)

func TestUnmarshal(t *testing.T) {
	for _, sv := range unmarshalTests {
		if err := checkUnmarshal(sv.s, sv.v); err != nil {
			t.Error(err.Error())
		}
	}
}

func BenchmarkUnmarshalAll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, sv := range unmarshalTests {
			checkUnmarshal(sv.s, sv.v)
		}
	}
}

type identity struct {
	Age       int
	FirstName string
	Ignored   string `bencode:"-"`
	LastName  string
}

func TestMarshalWithIgnoredField(t *testing.T) {
	id := identity{42, "Jack", "Why are you ignoring me?", "Daniel"}
	var buf bytes.Buffer
	err := Marshal(&buf, id)
	if err != nil {
		t.Fatal(err)
	}
	var id2 identity
	err = Unmarshal(&buf, &id2)
	if err != nil {
		t.Fatal(err)
	}
	if id.Age != id2.Age {
		t.Fatalf("Age should be the same, expected %d, got %d", id.Age, id2.Age)
	}
	if id.FirstName != id2.FirstName {
		t.Fatalf("FirstName should be the same, expected %s, got %s", id.FirstName, id2.FirstName)
	}
	if id.LastName != id2.LastName {
		t.Fatalf("LastName should be the same, expected %s, got %s", id.LastName, id2.LastName)
	}
	if id2.Ignored != "" {
		t.Fatalf("Ignored should be empty, got %s", id2.Ignored)
	}
}

type omitEmpty struct {
	Age       int
	Array     []string `bencode:",omitempty"`
	FirstName string
	Ignored   string `bencode:",omitempty"`
	LastName  string
	Renamed   string `bencode:"otherName,omitempty"`
}

func TestMarshalWithOmitEmptyFieldEmpty(t *testing.T) {
	oe := omitEmpty{42, []string{}, "Jack", "", "Daniel", ""}
	var buf bytes.Buffer
	err := Marshal(&buf, oe)
	if err != nil {
		t.Fatal(err)
	}
	buf2 := "d3:Agei42e9:FirstName4:Jack8:LastName6:Daniele"
	if string(buf.Bytes()) != buf2 {
		t.Fatalf("Wrong encoding, expected first line got second line\n`%s`\n`%s`\n", buf2, string(buf.Bytes()))
	}
}

func TestMarshalWithOmitEmptyFieldNonEmpty(t *testing.T) {
	oe := omitEmpty{42, []string{"first", "second"}, "Jack", "Not ignored", "Daniel", "Whisky"}
	var buf bytes.Buffer
	err := Marshal(&buf, oe)
	if err != nil {
		t.Fatal(err)
	}
	buf2 := "d3:Agei42e5:Arrayl5:first6:seconde9:FirstName4:Jack7:Ignored11:Not ignored8:LastName6:Daniel9:otherName6:Whiskye"
	if string(buf.Bytes()) != buf2 {
		t.Fatalf("Wrong encoding, expected first line got second line\n`%s`\n`%s`\n", buf2, string(buf.Bytes()))
	}
}

func TestMatshalDifferentTypes(t *testing.T) {

	buf := new(bytes.Buffer)
	Marshal(buf, []byte{'1', '2', '3'})
	if buf.String() != "3:123" {
		t.Fatalf("Incorrectly encoded byte array, got %s", buf.String())
	}

	buf = new(bytes.Buffer)
	Marshal(buf, []int{1, 2, 3})
	if buf.String() != "li1ei2ei3ee" {
		t.Fatalf("Incorrectly encoded byte array, got %s", buf.String())
	}
}
