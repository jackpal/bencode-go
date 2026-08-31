// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Represents bencode data structure using native Go types: booleans, floats,
// strings, slices, and maps.

package bencode

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
)

// RawMessage is a raw encoded bencode value. It can be used to delay
// bencode decoding or to access the original bytes of a bencoded value
// (such as the "info" dictionary in a BitTorrent metainfo file).
type RawMessage []byte

// Default limits for decoding bencode data.
const (
	// DefaultMaxStringLength is the default maximum string length (64 MiB).
	DefaultMaxStringLength int64 = 64 * 1024 * 1024

	// DefaultMaxDepth is the default maximum nesting depth for lists and dictionaries.
	DefaultMaxDepth int = 100

	// DefaultMaxElements is the default maximum number of bencode values parsed.
	DefaultMaxElements int64 = 3_000_000

	// maxNumLen is the maximum number of bytes allowed when scanning for an integer or string length delimiter.
	maxNumLen int = 64
)

// Options specifies configuration and resource limits for decoding bencode data.
type Options struct {
	// MaxStringLength limits the maximum length in bytes of any bencode string.
	// Default is DefaultMaxStringLength (64 MiB). Set to -1 for unlimited.
	MaxStringLength int64

	// MaxDepth limits the maximum nesting depth for nested lists and dictionaries.
	// Default is DefaultMaxDepth (100). Set to -1 for unlimited.
	MaxDepth int

	// MaxElements limits the total number of bencode values (integers, strings, lists, dicts) parsed.
	// Default is DefaultMaxElements (3,000,000). Set to -1 for unlimited.
	MaxElements int64

	// Strict enables strict adherence to the BitTorrent BEP 3 bencode specification:
	// - Rejects integer representations with leading zeros (e.g. "i03e" is rejected; "i0e" is allowed).
	// - Rejects negative zero ("i-0e").
	// - Rejects explicit positive sign ("i+5e").
	// - Rejects non-integer numbers (such as floats or exponential notation).
	// - Enforces lexicographically sorted dictionary keys without duplicates.
	// - Rejects string lengths with leading zeros (e.g. "03:abc").
	Strict bool
}

// DefaultOptions returns a new Options struct initialized with standard limits.
func DefaultOptions() Options {
	return Options{
		MaxStringLength: DefaultMaxStringLength,
		MaxDepth:        DefaultMaxDepth,
		MaxElements:     DefaultMaxElements,
		Strict:          false,
	}
}

// Decoder reads and decodes bencode values from an input stream.
type Decoder struct {
	r    *bufio.Reader
	opts Options
}

// NewDecoder returns a new Decoder reading from r with DefaultOptions.
func NewDecoder(r io.Reader) *Decoder {
	return NewDecoderWithOptions(r, DefaultOptions())
}

// NewDecoderWithOptions returns a new Decoder reading from r with the given Options.
func NewDecoderWithOptions(r io.Reader, opts Options) *Decoder {
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = bufio.NewReader(r)
	}
	return &Decoder{
		r:    br,
		opts: opts,
	}
}

// Buffered returns a reader of the data remaining in the Decoder's buffer.
// The reader is valid until the next call to Decode or Unmarshal.
func (d *Decoder) Buffered() io.Reader {
	n := d.r.Buffered()
	if n == 0 {
		return bytes.NewReader(nil)
	}
	buf, err := d.r.Peek(n)
	if err != nil {
		return bytes.NewReader(nil)
	}
	return bytes.NewReader(buf)
}

// SetMaxStringLength sets the maximum string length in bytes and returns the Decoder.
func (d *Decoder) SetMaxStringLength(limit int64) *Decoder {
	d.opts.MaxStringLength = limit
	return d
}

// SetMaxDepth sets the maximum nesting depth and returns the Decoder.
func (d *Decoder) SetMaxDepth(depth int) *Decoder {
	d.opts.MaxDepth = depth
	return d
}

// SetMaxElements sets the maximum total elements limit and returns the Decoder.
func (d *Decoder) SetMaxElements(elements int64) *Decoder {
	d.opts.MaxElements = elements
	return d
}

// SetStrict sets whether strict BEP 3 validation is enabled and returns the Decoder.
func (d *Decoder) SetStrict(strict bool) *Decoder {
	d.opts.Strict = strict
	return d
}

// Decode reads the next bencode value from the stream and returns its generic Go representation.
func (d *Decoder) Decode() (data any, err error) {
	state := &decodeState{opts: d.opts}
	return state.decode(d.r)
}

// Unmarshal reads the next bencode value and parses it into val (which must be a pointer).
func (d *Decoder) Unmarshal(val any) error {
	if reflect.TypeOf(val).Kind() != reflect.Ptr {
		return errors.New("Attempt to unmarshal into a non-pointer")
	}
	state := &decodeState{opts: d.opts}
	return unmarshalValueWithState(d.r, reflect.Indirect(reflect.ValueOf(val)), state)
}

// Decode parses the stream r and returns the generic bencode object representation.
// The object representation is a tree of Go data types: string, int64, uint64,
// []any, or map[string]any.
func Decode(reader io.Reader) (data any, err error) {
	br, ok := reader.(*bufio.Reader)
	if !ok {
		br = newBufioReader(reader)
		defer bufioReaderPool.Put(br)
	}
	state := &decodeState{opts: DefaultOptions()}
	return state.decode(br)
}

// DecodeWithOptions parses the stream r with custom Options and returns the generic bencode representation.
func DecodeWithOptions(reader io.Reader, opts Options) (data any, err error) {
	br, ok := reader.(*bufio.Reader)
	if !ok {
		br = newBufioReader(reader)
		defer bufioReaderPool.Put(br)
	}
	state := &decodeState{opts: opts}
	return state.decode(br)
}

// Unmarshal reads and parses the bencode syntax data from r into val using DefaultOptions.
func Unmarshal(r io.Reader, val any) (err error) {
	if reflect.TypeOf(val).Kind() != reflect.Ptr {
		return errors.New("Attempt to unmarshal into a non-pointer")
	}
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = newBufioReader(r)
		defer bufioReaderPool.Put(br)
	}
	state := &decodeState{opts: DefaultOptions()}
	return unmarshalValueWithState(br, reflect.Indirect(reflect.ValueOf(val)), state)
}

// UnmarshalWithOptions reads and parses bencode syntax data from r into val with custom Options.
func UnmarshalWithOptions(r io.Reader, val any, opts Options) error {
	if reflect.TypeOf(val).Kind() != reflect.Ptr {
		return errors.New("Attempt to unmarshal into a non-pointer")
	}
	br, ok := r.(*bufio.Reader)
	if !ok {
		br = newBufioReader(r)
		defer bufioReaderPool.Put(br)
	}
	state := &decodeState{opts: opts}
	return unmarshalValueWithState(br, reflect.Indirect(reflect.ValueOf(val)), state)
}

type decodeState struct {
	opts     Options
	depth    int
	elements int64
}

func (s *decodeState) incDepth() error {
	s.depth++
	if s.opts.MaxDepth >= 0 && s.depth > s.opts.MaxDepth {
		return fmt.Errorf("bencode: nesting depth %d exceeds max depth limit of %d", s.depth, s.opts.MaxDepth)
	}
	return nil
}

func (s *decodeState) decDepth() {
	if s.depth > 0 {
		s.depth--
	}
}

func (s *decodeState) incElement() error {
	s.elements++
	if s.opts.MaxElements >= 0 && s.elements > s.opts.MaxElements {
		return fmt.Errorf("bencode: element count %d exceeds max elements limit of %d", s.elements, s.opts.MaxElements)
	}
	return nil
}

func (s *decodeState) checkStringLength(length int64) error {
	if length < 0 {
		return fmt.Errorf("bencode: invalid negative string length: %d", length)
	}
	if s.opts.MaxStringLength >= 0 && length > s.opts.MaxStringLength {
		return fmt.Errorf("bencode: string length %d exceeds limit of %d bytes", length, s.opts.MaxStringLength)
	}
	return nil
}

func readNumBytes(r *bufio.Reader, delim byte) ([]byte, error) {
	peekLen := maxNumLen
	buf, err := r.Peek(peekLen)
	if err != nil && len(buf) == 0 {
		return nil, err
	}
	if i := bytes.IndexByte(buf, delim); i >= 0 {
		res := make([]byte, i)
		copy(res, buf[:i])
		_, err = r.Discard(i + 1)
		if err != nil {
			return nil, err
		}
		return res, nil
	}
	if len(buf) >= maxNumLen {
		return nil, fmt.Errorf("bencode: number representation exceeds max length limit of %d bytes", maxNumLen)
	}
	if err == io.EOF {
		return nil, io.ErrUnexpectedEOF
	}
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("bencode: number representation exceeds max length limit of %d bytes", maxNumLen)
}

func validateStrictInteger(buf []byte) error {
	if len(buf) == 0 {
		return errors.New("bencode: invalid integer: empty value in strict mode")
	}
	s := string(buf)
	if buf[0] == '+' {
		return fmt.Errorf("bencode: invalid integer %q: positive sign not allowed in strict mode", s)
	}
	if buf[0] == '-' {
		if len(buf) == 1 {
			return fmt.Errorf("bencode: invalid integer %q: missing digits after minus sign in strict mode", s)
		}
		if buf[1] == '0' {
			return fmt.Errorf("bencode: invalid integer %q: negative zero not allowed in strict mode", s)
		}
		for _, b := range buf[1:] {
			if b < '0' || b > '9' {
				return fmt.Errorf("bencode: invalid integer %q: non-digit character in strict mode", s)
			}
		}
		return nil
	}
	if buf[0] == '0' && len(buf) > 1 {
		return fmt.Errorf("bencode: invalid integer %q: leading zeros not allowed in strict mode", s)
	}
	for _, b := range buf {
		if b < '0' || b > '9' {
			return fmt.Errorf("bencode: invalid integer %q: non-digit character in strict mode", s)
		}
	}
	return nil
}

func validateStrictStringLength(buf []byte) error {
	if len(buf) == 0 {
		return errors.New("bencode: invalid string length: empty value in strict mode")
	}
	s := string(buf)
	if buf[0] == '+' || buf[0] == '-' {
		return fmt.Errorf("bencode: invalid string length %q: sign not allowed in strict mode", s)
	}
	if buf[0] == '0' && len(buf) > 1 {
		return fmt.Errorf("bencode: invalid string length %q: leading zeros not allowed in strict mode", s)
	}
	for _, b := range buf {
		if b < '0' || b > '9' {
			return fmt.Errorf("bencode: invalid string length %q: non-digit character in strict mode", s)
		}
	}
	return nil
}
