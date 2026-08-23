package bencode

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"sync"
)

// Parser
//
// Implements parsing but not the actions.  Those are
// carried out by the implementation of the builder interface.
// A builder represents the object being created.
// Calling a method like Int64(i) sets that object to i.
// Calling a method like Elem(i) or Key(s) creates a
// new builder for a subpiece of the object (logically,
// a slice element or a map key).
//
// There are two Builders, in other files.
// The decoder builds a generic bencode structures
// in which maps are maps.
// The structBuilder copies data into a possibly
// nested data structure, using the "map keys"
// as struct field names.

// A builder is an interface implemented by clients and passed
// to the bencode parser.  It gives clients full control over the
// eventual representation returned by the parser.
type builder interface {
	// Set value
	Int64(i int64)
	Uint64(i uint64)
	Float64(f float64)
	String(s string)
	Array()
	Map()

	// Create sub-Builders
	Elem(i int) builder
	Key(s string) builder

	// Flush changes to parent builder if necessary.
	Flush()
}

// Deprecated: This type is currently unused. It is exposed for backwards
// compatability. The public API that previously used this type,
//
//    Unmarshal(r Reader, val any) (err error)
//
// is now
//
//    Unmarshal(r io.Reader, val any) (err error)
//
// Which is compatible, since any Reader is also an io.Reader.
// Clients should drop their use of this type. It may be removed in the future.
type Reader interface {
	io.Reader
	io.ByteScanner
}

// Read bytes up until delim, return slice without delimiter byte.
func readSlice(r *bufio.Reader, delim byte) (data []byte, err error) {
	if data, err = r.ReadSlice(delim); err != nil {
		return
	}
	lenData := len(data)
	if lenData > 0 {
		data = data[:lenData-1]
	} else {
		err = io.ErrUnexpectedEOF
	}
	return
}

func decodeStringWithState(r *bufio.Reader, state *decodeState) (data string, err error) {
	lenBuf, err := readNumBytes(r, ':')
	if err != nil {
		return "", err
	}
	if state.opts.Strict {
		if err := validateStrictStringLength(lenBuf); err != nil {
			return "", err
		}
	}
	length, err := strconv.ParseInt(string(lenBuf), 10, 64)
	if err != nil {
		return "", fmt.Errorf("bencode: invalid string length %q: %w", string(lenBuf), err)
	}
	if err := state.checkStringLength(length); err != nil {
		return "", err
	}

	// Can we peek that much data out of r?
	if peekBuf, peekErr := r.Peek(int(length)); peekErr == nil {
		data = string(peekBuf)
		_, err = r.Discard(int(length))
		return
	}

	var buf = make([]byte, length)
	if _, err = io.ReadFull(r, buf); err != nil {
		return "", err
	}
	data = string(buf)
	return
}

func parseFromReader(r *bufio.Reader, build builder, state *decodeState) (err error) {
	var c byte
	if err = state.incElement(); err != nil {
		goto exit
	}
	c, err = r.ReadByte()
	if err != nil {
		goto exit
	}
	switch {
	case c >= '0' && c <= '9':
		// String
		err = r.UnreadByte()
		if err != nil {
			goto exit
		}
		var str string
		str, err = decodeStringWithState(r, state)
		if err != nil {
			goto exit
		}
		build.String(str)

	case c == 'd':
		// dictionary
		if err = state.incDepth(); err != nil {
			goto exit
		}
		defer state.decDepth()

		build.Map()
		var lastKey string
		var hasLastKey bool
		for {
			c, err = r.ReadByte()
			if err != nil {
				goto exit
			}
			if c == 'e' {
				break
			}
			err = r.UnreadByte()
			if err != nil {
				goto exit
			}
			var key string
			key, err = decodeStringWithState(r, state)
			if err != nil {
				goto exit
			}
			if state.opts.Strict {
				if hasLastKey {
					if key == lastKey {
						err = fmt.Errorf("bencode: duplicate dictionary key %q", key)
						goto exit
					}
					if key < lastKey {
						err = fmt.Errorf("bencode: dictionary keys not in ascending order: %q followed by %q", lastKey, key)
						goto exit
					}
				}
				lastKey = key
				hasLastKey = true
			}
			err = parseFromReader(r, build.Key(key), state)
			if err != nil {
				goto exit
			}
		}

	case c == 'i':
		var buf []byte
		buf, err = readNumBytes(r, 'e')
		if err != nil {
			goto exit
		}
		if state.opts.Strict {
			if err = validateStrictInteger(buf); err != nil {
				goto exit
			}
			var i int64
			i, err = strconv.ParseInt(string(buf), 10, 64)
			if err != nil {
				err = fmt.Errorf("bencode: invalid integer %q: %w", string(buf), err)
				goto exit
			}
			build.Int64(i)
		} else {
			str := string(buf)
			var i int64
			var i2 uint64
			var f float64
			if i, err = strconv.ParseInt(str, 10, 64); err == nil {
				build.Int64(i)
			} else if i2, err = strconv.ParseUint(str, 10, 64); err == nil {
				build.Uint64(i2)
			} else if f, err = strconv.ParseFloat(str, 64); err == nil {
				build.Float64(f)
			} else {
				err = fmt.Errorf("bencode: bad integer %q", str)
				goto exit
			}
		}

	case c == 'l':
		// array
		if err = state.incDepth(); err != nil {
			goto exit
		}
		defer state.decDepth()

		build.Array()
		n := 0
		for {
			c, err = r.ReadByte()
			if err != nil {
				goto exit
			}
			if c == 'e' {
				break
			}
			err = r.UnreadByte()
			if err != nil {
				goto exit
			}
			err = parseFromReader(r, build.Elem(n), state)
			if err != nil {
				goto exit
			}
			n++
		}
	default:
		err = fmt.Errorf("Unexpected character: '%v'", c)
	}
exit:
	build.Flush()
	return
}

// Parse parses the bencode stream and makes calls to
// the builder to construct a parsed representation.
func parse(reader io.Reader, builder builder, state *decodeState) (err error) {
	// Check to see if the reader already fulfills the bufio.Reader interface.
	// Wrap it in a bufio.Reader if it doesn't.
	r, ok := reader.(*bufio.Reader)
	if !ok {
		r = newBufioReader(reader)
		defer bufioReaderPool.Put(r)
	}

	return parseFromReader(r, builder, state)
}

var bufioReaderPool sync.Pool

func newBufioReader(r io.Reader) *bufio.Reader {
	if v := bufioReaderPool.Get(); v != nil {
		br := v.(*bufio.Reader)
		br.Reset(r)
		return br
	}
	return bufio.NewReader(r)
}
