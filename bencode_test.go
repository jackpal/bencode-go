package bencode

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"reflect"
	"testing"
)

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
		// This test is architecture specific.
		// See https://stackoverflow.com/a/70259392
		// SVPair{"i7.574E+20e", -9223372036854775808},
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

func TestMarshalDifferentTypes(t *testing.T) {

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

type unexportedFields struct {
	Public     string
	unexported string
	privateInt int
	Age        int
}

func TestSkipUnexportedFields(t *testing.T) {
	u := unexportedFields{
		Public:     "hello",
		unexported: "secret",
		privateInt: 99,
		Age:        30,
	}

	var buf bytes.Buffer
	if err := Marshal(&buf, u); err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expected := "d3:Agei30e6:Public5:helloe"
	if buf.String() != expected {
		t.Fatalf("Expected %q, got %q", expected, buf.String())
	}

	// Also verify unmarshaling does not populate or match unexported fields
	input := "d3:Agei30e6:Public5:hello10:unexported7:changed10:privateInti42ee"
	var dest unexportedFields
	if err := Unmarshal(bytes.NewReader([]byte(input)), &dest); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if dest.Public != "hello" || dest.Age != 30 {
		t.Fatalf("Public fields not set correctly: %+v", dest)
	}
	if dest.unexported != "" || dest.privateInt != 0 {
		t.Fatalf("Unexported fields should not have been set: %+v", dest)
	}
}

type torrentMeta struct {
	Announce string     `bencode:"announce"`
	Comment  string     `bencode:"comment,omitempty"`
	Info     RawMessage `bencode:"info"`
}

func TestRawMessageTorrentInfoHash(t *testing.T) {
	// Raw bencode with an info dict
	infoDict := "d6:lengthi170917888e12:piece lengthi262144e4:name12:testfile.isoe"
	input := "d8:announce27:http://tracker.example.com/4:info" + infoDict + "e"

	var meta torrentMeta
	if err := Unmarshal(bytes.NewReader([]byte(input)), &meta); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if meta.Announce != "http://tracker.example.com/" {
		t.Fatalf("Expected announce URL, got %q", meta.Announce)
	}

	if string(meta.Info) != infoDict {
		t.Fatalf("Expected info raw bytes %q, got %q", infoDict, string(meta.Info))
	}

	// Verify info_hash SHA-1 computation on raw bytes
	expectedHash := sha1.Sum([]byte(infoDict))
	actualHash := sha1.Sum(meta.Info)
	if actualHash != expectedHash {
		t.Fatalf("Hash mismatch: expected %x, got %x", expectedHash, actualHash)
	}

	// Re-marshal and ensure exact byte-for-byte preservation of the raw info dict
	var buf bytes.Buffer
	if err := Marshal(&buf, meta); err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if buf.String() != input {
		t.Fatalf("Expected round-trip %q, got %q", input, buf.String())
	}
}

func TestRawMessageTopLevel(t *testing.T) {
	input := "d3:cati1e3:dogi2ee"
	var raw RawMessage
	if err := Unmarshal(bytes.NewReader([]byte(input)), &raw); err != nil {
		t.Fatalf("Unmarshal into RawMessage failed: %v", err)
	}
	if string(raw) != input {
		t.Fatalf("Expected %q, got %q", input, string(raw))
	}

	var buf bytes.Buffer
	if err := Marshal(&buf, raw); err != nil {
		t.Fatalf("Marshal RawMessage failed: %v", err)
	}
	if buf.String() != input {
		t.Fatalf("Expected %q, got %q", input, buf.String())
	}
}

func TestRawMessageMapAndSlice(t *testing.T) {
	// Map of RawMessage
	mapInput := "d1:ai10e1:b5:helloe"
	var m map[string]RawMessage
	if err := Unmarshal(bytes.NewReader([]byte(mapInput)), &m); err != nil {
		t.Fatalf("Unmarshal into map[string]RawMessage failed: %v", err)
	}
	if string(m["a"]) != "i10e" {
		t.Fatalf("Expected m[\"a\"] = 'i10e', got %q", string(m["a"]))
	}
	if string(m["b"]) != "5:hello" {
		t.Fatalf("Expected m[\"b\"] = '5:hello', got %q", string(m["b"]))
	}

	// Slice of RawMessage
	sliceInput := "li10e5:hellod1:ki1eee"
	var s []RawMessage
	if err := Unmarshal(bytes.NewReader([]byte(sliceInput)), &s); err != nil {
		t.Fatalf("Unmarshal into []RawMessage failed: %v", err)
	}
	if len(s) != 3 {
		t.Fatalf("Expected 3 items, got %d", len(s))
	}
	if string(s[0]) != "i10e" || string(s[1]) != "5:hello" || string(s[2]) != "d1:ki1ee" {
		t.Fatalf("Slice contents mismatch: %v", s)
	}
}

func TestRawMessagePointer(t *testing.T) {
	type withPtr struct {
		Data *RawMessage `bencode:"data"`
	}
	input := "d4:datai999ee"
	var wp withPtr
	if err := Unmarshal(bytes.NewReader([]byte(input)), &wp); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if wp.Data == nil || string(*wp.Data) != "i999e" {
		t.Fatalf("Expected pointer data 'i999e', got %v", wp.Data)
	}

	var buf bytes.Buffer
	if err := Marshal(&buf, wp); err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	if buf.String() != input {
		t.Fatalf("Expected %q, got %q", input, buf.String())
	}
}
