# bencode-go

A Go language binding for encoding and decoding data in the bencode format that
is used by the BitTorrent peer-to-peer file sharing protocol.

## Quick Start

### Get the package
```bash
go get -u github.com/jackpal/bencode-go
```

### Import the package
```go
import bencode "github.com/jackpal/bencode-go"
```

### Unmarshal a bencode stream into an object
```go
data := myAwesomeObject{}
err := bencode.Unmarshal(reader, &data)
```

### Decode a bencode stream
```go
data, err := bencode.Decode(reader)
```

### Decode with configurable limits or strict validation
```go
dec := bencode.NewDecoder(reader).
	SetMaxStringLength(10 * 1024 * 1024). // 10 MiB
	SetMaxDepth(50).
	SetStrict(true) // Enforces BEP 3 rules (no leading zeros, sorted keys, etc.)

data, err := dec.Decode()
// or: err := dec.Unmarshal(&myObject)

// If the stream contains trailing non-bencode data, access unread buffered bytes:
leftover := dec.Buffered()
```

### Encode an object into a bencode stream
```go
err := bencode.Marshal(writer, data)
```

## Complete documentation

http://godoc.org/github.com/jackpal/bencode-go

## License

This project is licensed under the Go Authors standard license. (See the LICENSE
file for details.)

## Version History

| tag    | Notes                                                                           |
| ------ | ------------------------------------------------------------------------------- |
| v1.1.0 | Added Decoder with configurable resource limits, strict BEP 3 validation, and `any` types. |
| v1.0.2 | Added go module.                                                                |
| v1.0.1 | Removed architecture specific test that was failing on ARM.                      |
| v1.0.0 | First version.                                                                  |
