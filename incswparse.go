package bencode

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"strconv"
)

// A relatively fast unmarshaler.
// Adapted from https://github.com/IncSW/go-bencode/blob/master/unmarshaler.go
// License: https://github.com/IncSW/go-bencode/blob/master/LICENSE

func (s *decodeState) decode(data *bufio.Reader) (any, error) {
	return s.unmarshal(data)
}

func (s *decodeState) unmarshal(data *bufio.Reader) (any, error) {
	if err := s.incElement(); err != nil {
		return nil, err
	}
	ch, err := data.ReadByte()
	if err != nil {
		return nil, err
	}
	switch ch {
	case 'i':
		integerBuffer, err := readNumBytes(data, 'e')
		if err != nil {
			return nil, err
		}
		if s.opts.Strict {
			if err := validateStrictInteger(integerBuffer); err != nil {
				return nil, err
			}
		}
		integer, err := strconv.ParseInt(string(integerBuffer), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bencode: invalid integer %q: %w", string(integerBuffer), err)
		}
		return integer, nil

	case 'l':
		if err := s.incDepth(); err != nil {
			return nil, err
		}
		defer s.decDepth()

		list := []any{}
		for {
			c, err2 := data.ReadByte()
			if err2 != nil {
				return nil, err2
			}
			if c == 'e' {
				return list, nil
			}
			data.UnreadByte()

			value, err := s.unmarshal(data)
			if err != nil {
				return nil, err
			}

			list = append(list, value)
		}

	case 'd':
		if err := s.incDepth(); err != nil {
			return nil, err
		}
		defer s.decDepth()

		dictionary := map[string]any{}
		var lastKey string
		var hasLastKey bool

		for {
			c, err2 := data.ReadByte()
			if err2 != nil {
				return nil, err2
			}
			if c == 'e' {
				return dictionary, nil
			}
			data.UnreadByte()

			keyVal, err := s.unmarshal(data)
			if err != nil {
				return nil, err
			}

			key, ok := keyVal.(string)
			if !ok {
				return nil, errors.New("bencode: non-string dictionary key")
			}

			if s.opts.Strict {
				if hasLastKey {
					if key == lastKey {
						return nil, fmt.Errorf("bencode: duplicate dictionary key %q", key)
					}
					if key < lastKey {
						return nil, fmt.Errorf("bencode: dictionary keys not in ascending order: %q followed by %q", lastKey, key)
					}
				}
				lastKey = key
				hasLastKey = true
			}

			value, err := s.unmarshal(data)
			if err != nil {
				return nil, err
			}

			dictionary[key] = value
		}

	default:
		data.UnreadByte()
		lenBuf, err := readNumBytes(data, ':')
		if err != nil {
			return nil, err
		}
		if s.opts.Strict {
			if err := validateStrictStringLength(lenBuf); err != nil {
				return nil, err
			}
		}
		stringLength, err := strconv.ParseInt(string(lenBuf), 10, 64)
		if err != nil {
			return nil, fmt.Errorf("bencode: invalid string length %q: %w", string(lenBuf), err)
		}
		if err := s.checkStringLength(stringLength); err != nil {
			return nil, err
		}

		if peekBuf, peekErr := data.Peek(int(stringLength)); peekErr == nil {
			str := string(peekBuf)
			_, err = data.Discard(int(stringLength))
			return str, err
		}

		var buf = make([]byte, stringLength)
		if _, err = io.ReadFull(data, buf); err != nil {
			return nil, err
		}

		return string(buf), nil
	}
}
