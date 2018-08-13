// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Represents bencode data structure using native Go types: booleans, floats,
// strings, slices, and maps.

package bencode

import (
	"bufio"
	"io"
)

// Decode a bencode stream

// Decode parses the stream r and returns the
// generic bencode object representation.  The object representation is a tree
// of Go data types.  The data return value may be one of string,
// int64, uint64, []interface{} or map[string]interface{}.  The slice and map
// elements may in turn contain any of the types listed above and so on.
//
// If Decode encounters a syntax error, it returns with err set to an
// instance of Error.
func Decode(reader io.Reader) (data interface{}, err error) {
	// Check to see if the reader already fulfills the bufio.Reader interface.
	// Wrap it in a bufio.Reader if it doesn't.
	bufioReader, ok := reader.(*bufio.Reader)
	if !ok {
		bufioReader = newBufioReader(reader)
		defer bufioReaderPool.Put(bufioReader)
	}

	return decodeFromReader(bufioReader)
}
