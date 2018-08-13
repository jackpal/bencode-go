package bencode

import (
    "bufio"
    "bytes"
    "errors"
    "strconv"
)

// A relatively fast unmarshaler.
// Adapted from https://github.com/IncSW/go-bencode/blob/master/unmarshaler.go
// License: https://github.com/IncSW/go-bencode/blob/master/LICENSE

// Differences from IncSW are for compatibility with the existing bencode-go API:
// (a) Uses a bufio.Reader rather than a raw []byte
// (b) Strings are returned as golang strings rather than as raw []byte arrays.

func decodeFromReader(r *bufio.Reader) (data interface{}, err error) {
    result, err := unmarshal(r)
    if err != nil {
        return nil, err
    }

    return result, nil
}

func unmarshal(data *bufio.Reader) (interface{}, error) {
    ch, err := data.ReadByte()
    if err != nil {
        return nil, err
    }
    switch ch {
    case 'i':
        integerBuffer, err := optimisticReadBytes(data, 'e')
        if err != nil {
            return nil, err
        }
        integerBuffer = integerBuffer[:len(integerBuffer)-1]

        integer, err := strconv.ParseInt(string(integerBuffer), 10, 64)
        if err != nil {
            return nil, err
        }

        return integer, nil

    case 'l':
        list := []interface{}{}
        for {
            c, err2 := data.ReadByte()
            if err2 == nil {
                if c == 'e' {
                    return list, nil
                } else {
                    data.UnreadByte()
                }
            }

            value, err := unmarshal(data)
            if err != nil {
                return nil, err
            }

            list = append(list, value)
        }

    case 'd':
        dictionary := map[string]interface{}{}
        for {
            c, err2 := data.ReadByte()
            if err2 == nil {
                if c == 'e' {
                    return dictionary, nil
                } else {
                    data.UnreadByte()
                }
            }
            value, err := unmarshal(data)
            if err != nil {
                return nil, err
            }

            key, ok := value.(string)
            if !ok {
                return nil, errors.New("bencode: non-string dictionary key")
            }

            value, err = unmarshal(data)
            if err != nil {
                return nil, err
            }

            dictionary[key] = value
        }

    default:
        data.UnreadByte()
        stringLengthBuffer, err := optimisticReadBytes(data, ':')
        if err != nil {
            return nil, err
        }
        stringLengthBuffer = stringLengthBuffer[:len(stringLengthBuffer)-1]

        stringLength, err := strconv.ParseInt(string(stringLengthBuffer), 10, 64)
        if err != nil {
            return nil, err
        }

        buf := make([]byte, stringLength)

        _, err = readAtLeast(data, buf, int(stringLength))

        return string(buf), err
    }
}

// Reads bytes out of local buffer if possible, which avoids an extra copy.
// The result []byte is only guarenteed to be valid until the next call to a Read method.
func optimisticReadBytes(data *bufio.Reader, delim byte) ([]byte, error) {
    buffered := data.Buffered()
    var buffer []byte
    var err error
    if buffer, err = data.Peek(buffered); err != nil {
        return nil, err
    }

    if i := bytes.IndexByte(buffer, delim); i >= 0 {
        return data.ReadSlice(delim)
    }
    return data.ReadBytes(delim)
}
