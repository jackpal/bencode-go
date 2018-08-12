// Copyright (c) 2017 Aleksey Lin <aleksey@incsw.in> (https://incsw.in)
// MIT licence, see https://github.com/IncSW/go-bencode/LICENSE
// Adapted from https://github.com/IncSW/go-bencode

package bencode

import (
"testing"
"bytes"
)

var marshalTestData = map[string]interface{}{
    "announce": []byte("udp://tracker.publicbt.com:80/announce"),
    "announce-list": []interface{}{
        []interface{}{[]byte("udp://tracker.publicbt.com:80/announce")},
        []interface{}{[]byte("udp://tracker.openbittorrent.com:80/announce")},
    },
    "comment": []byte("Debian CD from cdimage.debian.org"),
    "info": map[string]interface{}{
        "name":         []byte("debian-8.8.0-arm64-netinst.iso"),
        "length":       170917888,
        "piece length": 262144,
    },
}

var unmarshalTestData = []byte("d4:infod6:lengthi170917888e12:piece lengthi262144e4:name30:debian-8.8.0-arm64-netinst.isoe8:announce38:udp://tracker.publicbt.com:80/announce13:announce-listll38:udp://tracker.publicbt.com:80/announceel44:udp://tracker.openbittorrent.com:80/announceee7:comment33:Debian CD from cdimage.debian.orge")

func BenchmarkJackpalBencodeMarshal(b *testing.B) {
    b.ReportAllocs()
    for n := 0; n < b.N; n++ {
        Marshal(bytes.NewBuffer(nil), marshalTestData)
    }
}

func BenchmarkJackpalBencodeUnmarshal(b *testing.B) {
    b.ReportAllocs()
    for n := 0; n < b.N; n++ {
        Decode(bytes.NewReader(unmarshalTestData))
    }
}
