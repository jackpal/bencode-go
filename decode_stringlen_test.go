// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bencode

import (
	"bufio"
	"bytes"
	"io"
	"strings"
	"testing"
)

// A malformed string length (negative, or so large it overflows the slice
// allocator) must be reported as an error, not crash the decoder.
func TestDecodeMalformedStringLength(t *testing.T) {
	for _, in := range []string{
		"-1:",
		"900000000000000000:",
		"5:abc", // truncated: claims 5 bytes, only 3 present
	} {
		if _, err := Decode(strings.NewReader(in)); err == nil {
			t.Errorf("Decode(%q) = nil error, want error", in)
		}
	}
}

// A well-formed string must still decode correctly after the length check.
func TestDecodeStringRoundTrip(t *testing.T) {
	got, err := Decode(bytes.NewReader([]byte("5:hello")))
	if err != nil {
		t.Fatalf("Decode: unexpected error: %v", err)
	}
	if got != "hello" {
		t.Errorf("Decode = %q, want %q", got, "hello")
	}
}

func TestMaxStringLength(t *testing.T) {
	input := "10:0123456789"

	// Default limit is 64MB, so 10 bytes succeeds
	val, err := Decode(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "0123456789" {
		t.Fatalf("expected 0123456789, got %v", val)
	}

	// Set limit to 5 bytes
	dec := NewDecoder(strings.NewReader(input)).SetMaxStringLength(5)
	_, err = dec.Decode()
	if err == nil {
		t.Fatal("expected error for string length exceeding limit, got nil")
	}
	if !strings.Contains(err.Error(), "string length 10 exceeds limit of 5 bytes") {
		t.Fatalf("error message missing limit info: %v", err)
	}

	// Test with Unmarshal
	var dest string
	dec = NewDecoder(strings.NewReader(input)).SetMaxStringLength(5)
	err = dec.Unmarshal(&dest)
	if err == nil {
		t.Fatal("expected error for string length exceeding limit in Unmarshal, got nil")
	}
	if !strings.Contains(err.Error(), "string length 10 exceeds limit of 5 bytes") {
		t.Fatalf("error message missing limit info: %v", err)
	}
}

func TestMaxDepth(t *testing.T) {
	// Nested list with depth 5: l l l l l i1e e e e e e
	input := "llllli1eeeeee"

	// Decode with max depth 3
	opts := DefaultOptions()
	opts.MaxDepth = 3
	_, err := DecodeWithOptions(strings.NewReader(input), opts)
	if err == nil {
		t.Fatal("expected depth limit error, got nil")
	}
	if !strings.Contains(err.Error(), "nesting depth 4 exceeds max depth limit of 3") {
		t.Fatalf("error message missing depth info: %v", err)
	}

	// Decode with max depth 10
	opts.MaxDepth = 10
	val, err := DecodeWithOptions(strings.NewReader(input), opts)
	if err != nil {
		t.Fatalf("unexpected error with depth 10: %v", err)
	}
	if val == nil {
		t.Fatal("expected non-nil decoded value")
	}

	// Test with Unmarshal
	type nestedList [][][][][]int
	var nl nestedList
	err = UnmarshalWithOptions(strings.NewReader(input), &nl, Options{MaxDepth: 3, MaxStringLength: -1, MaxElements: -1})
	if err == nil {
		t.Fatal("expected depth limit error in Unmarshal, got nil")
	}
	if !strings.Contains(err.Error(), "nesting depth 4 exceeds max depth limit of 3") {
		t.Fatalf("error message missing depth info: %v", err)
	}
}

func TestMaxElements(t *testing.T) {
	input := "li1ei2ei3ei4ei5ee"

	dec := NewDecoder(strings.NewReader(input)).SetMaxElements(3)
	_, err := dec.Decode()
	if err == nil {
		t.Fatal("expected element limit error, got nil")
	}
	if !strings.Contains(err.Error(), "element count 4 exceeds max elements limit of 3") {
		t.Fatalf("error message missing element limit info: %v", err)
	}

	// With limit 10
	dec = NewDecoder(strings.NewReader(input)).SetMaxElements(10)
	val, err := dec.Decode()
	if err != nil {
		t.Fatalf("unexpected error with elements limit 10: %v", err)
	}
	slice, ok := val.([]any)
	if !ok || len(slice) != 5 {
		t.Fatalf("expected slice of length 5, got %v", val)
	}
}

func TestStrictIntegerValidation(t *testing.T) {
	tests := []struct {
		input       string
		wantErrPart string
	}{
		{"i03e", "leading zeros not allowed in strict mode"},
		{"i-0e", "negative zero not allowed in strict mode"},
		{"i-03e", "negative zero not allowed in strict mode"},
		{"i+5e", "positive sign not allowed in strict mode"},
		{"i7.5e", "non-digit character in strict mode"},
		{"ie", "empty value in strict mode"},
	}

	for _, tc := range tests {
		opts := DefaultOptions()
		opts.Strict = true
		_, err := DecodeWithOptions(strings.NewReader(tc.input), opts)
		if err == nil {
			t.Errorf("DecodeWithOptions(%q, Strict=true) = nil error, want error containing %q", tc.input, tc.wantErrPart)
			continue
		}
		if !strings.Contains(err.Error(), tc.wantErrPart) {
			t.Errorf("DecodeWithOptions(%q, Strict=true) error %q does not contain %q", tc.input, err.Error(), tc.wantErrPart)
		}

		// Also test with Unmarshal
		var i int64
		err = UnmarshalWithOptions(strings.NewReader(tc.input), &i, opts)
		if err == nil {
			t.Errorf("UnmarshalWithOptions(%q, Strict=true) = nil error, want error containing %q", tc.input, tc.wantErrPart)
			continue
		}
		if !strings.Contains(err.Error(), tc.wantErrPart) {
			t.Errorf("UnmarshalWithOptions(%q, Strict=true) error %q does not contain %q", tc.input, err.Error(), tc.wantErrPart)
		}
	}
}

func TestStrictStringLengthValidation(t *testing.T) {
	tests := []struct {
		input       string
		wantErrPart string
	}{
		{"03:abc", "leading zeros not allowed in strict mode"},
		{"+3:abc", "sign not allowed in strict mode"},
		{"-1:abc", "sign not allowed in strict mode"},
	}

	for _, tc := range tests {
		opts := DefaultOptions()
		opts.Strict = true
		_, err := DecodeWithOptions(strings.NewReader(tc.input), opts)
		if err == nil {
			t.Errorf("DecodeWithOptions(%q, Strict=true) = nil error, want error containing %q", tc.input, tc.wantErrPart)
			continue
		}
		if !strings.Contains(err.Error(), tc.wantErrPart) {
			t.Errorf("DecodeWithOptions(%q, Strict=true) error %q does not contain %q", tc.input, err.Error(), tc.wantErrPart)
		}
	}
}

func TestStrictDictionaryKeyOrdering(t *testing.T) {
	// Unsorted keys: "z" before "a"
	unsorted := "d1:zi1e1:ai2ee"

	// Non-strict allows unsorted
	val, err := Decode(strings.NewReader(unsorted))
	if err != nil {
		t.Fatalf("Decode in non-strict mode failed unexpectedly: %v", err)
	}
	m, ok := val.(map[string]any)
	if !ok || len(m) != 2 {
		t.Fatalf("expected map with 2 keys, got %v", val)
	}

	// Strict rejects unsorted
	opts := DefaultOptions()
	opts.Strict = true
	_, err = DecodeWithOptions(strings.NewReader(unsorted), opts)
	if err == nil {
		t.Fatal("expected error for unsorted keys in strict mode, got nil")
	}
	if !strings.Contains(err.Error(), "dictionary keys not in ascending order") {
		t.Fatalf("expected ascending order error, got: %v", err)
	}

	// Strict rejects duplicate keys
	duplicate := "d1:ai1e1:ai2ee"
	_, err = DecodeWithOptions(strings.NewReader(duplicate), opts)
	if err == nil {
		t.Fatal("expected error for duplicate keys in strict mode, got nil")
	}
	if !strings.Contains(err.Error(), "duplicate dictionary key \"a\"") {
		t.Fatalf("expected duplicate key error, got: %v", err)
	}

	// Sorted keys succeed in strict mode
	sorted := "d1:ai1e1:zi2ee"
	val, err = DecodeWithOptions(strings.NewReader(sorted), opts)
	if err != nil {
		t.Fatalf("unexpected error for sorted keys in strict mode: %v", err)
	}
	if val == nil {
		t.Fatal("expected non-nil value")
	}
}

func TestTypeMismatchNoPanic(t *testing.T) {
	type Target struct {
		Flag bool   `bencode:"flag"`
		Name string `bencode:"name"`
	}

	// Incoming stream sends an integer for 'flag'
	input := "d4:flagi42e4:name5:alicee"
	var dest Target
	err := Unmarshal(strings.NewReader(input), &dest)
	if err != nil {
		t.Fatalf("unexpected error unmarshaling with type mismatch: %v", err)
	}
	if dest.Name != "alice" {
		t.Fatalf("expected Name='alice', got %q", dest.Name)
	}
	// Flag should remain its default zero value false
	if dest.Flag != false {
		t.Fatalf("expected Flag=false, got %v", dest.Flag)
	}
}

func TestDecoderConsecutive(t *testing.T) {
	input := "i10ei20e5:hello"
	dec := NewDecoder(strings.NewReader(input))

	v1, err := dec.Decode()
	if err != nil {
		t.Fatalf("failed decoding first value: %v", err)
	}
	if v1 != int64(10) {
		t.Fatalf("expected 10, got %v", v1)
	}

	v2, err := dec.Decode()
	if err != nil {
		t.Fatalf("failed decoding second value: %v", err)
	}
	if v2 != int64(20) {
		t.Fatalf("expected 20, got %v", v2)
	}

	v3, err := dec.Decode()
	if err != nil {
		t.Fatalf("failed decoding third value: %v", err)
	}
	if v3 != "hello" {
		t.Fatalf("expected 'hello', got %v", v3)
	}

	_, err = dec.Decode()
	if err != io.EOF {
		t.Fatalf("expected EOF at end of stream, got %v", err)
	}
}

func TestDecoderBuffered(t *testing.T) {
	input := "d4:name5:aliceeraw payload bytes here"
	dec := NewDecoder(strings.NewReader(input))

	var dest struct {
		Name string
	}
	if err := dec.Unmarshal(&dest); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if dest.Name != "alice" {
		t.Fatalf("expected Name='alice', got %q", dest.Name)
	}

	bufferedData, err := io.ReadAll(dec.Buffered())
	if err != nil {
		t.Fatalf("reading buffered data failed: %v", err)
	}
	if string(bufferedData) != "raw payload bytes here" {
		t.Fatalf("expected 'raw payload bytes here', got %q", string(bufferedData))
	}
}

func TestUnmarshalWithBufioReaderRetainsBuffer(t *testing.T) {
	input := "i42etrail"
	br := bufio.NewReader(strings.NewReader(input))

	var val int
	if err := Unmarshal(br, &val); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	if val != 42 {
		t.Fatalf("expected 42, got %d", val)
	}

	rem, err := io.ReadAll(br)
	if err != nil {
		t.Fatalf("reading remainder failed: %v", err)
	}
	if string(rem) != "trail" {
		t.Fatalf("expected 'trail', got %q", string(rem))
	}
}
