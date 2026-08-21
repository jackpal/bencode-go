// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bencode

import (
	"bytes"
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
