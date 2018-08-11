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

### Encode an object into a bencode stream
```go
err := bencode.Marshal(writer, data)
```

## Complete documentation

http://godoc.org/github.com/jackpal/bencode-go

## License

This project is licensed under the Go Authors standard license. (See the LICENSE
file for details.)
