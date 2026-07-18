package zcashblob

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

func TestCompactSizeTruncation(t *testing.T) {
	for _, format := range []struct {
		prefix  byte
		payload int
	}{
		{0xfd, 2},
		{0xfe, 4},
		{0xff, 8},
	} {
		for n := 0; n < format.payload; n++ {
			encoded := append([]byte{format.prefix}, make([]byte, n)...)
			_, err := readCompact(bytes.NewReader(encoded))
			want := io.EOF
			if n > 0 {
				want = io.ErrUnexpectedEOF
			}
			if !errors.Is(err, want) {
				t.Fatalf("prefix %x with %d payload bytes: got %v, want %v", format.prefix, n, err, want)
			}
		}
	}
}

func TestCompactSizeLen(t *testing.T) {
	for _, test := range []struct {
		value uint64
		want  int
	}{
		{0, 1},
		{252, 1},
		{253, 3},
		{0xffff, 3},
		{0x10000, 5},
		{0xffffffff, 5},
		{0x100000000, 9},
		{^uint64(0), 9},
	} {
		if got := compactSizeLen(test.value); got != test.want {
			t.Fatalf("compactSizeLen(%d) = %d, want %d", test.value, got, test.want)
		}
	}
}

func FuzzCompactSize(f *testing.F) {
	for _, value := range []uint64{0, 252, 253, 0xffff, 0x10000, 0xffffffff, 0x100000000, ^uint64(0)} {
		var encoded bytes.Buffer
		if err := writeCompact(&encoded, value); err != nil {
			f.Fatal(err)
		}
		f.Add(append([]byte(nil), encoded.Bytes()...))
	}
	for _, malformed := range [][]byte{{}, {0xfd}, {0xfe, 0}, {0xff, 0, 0, 0}} {
		f.Add(malformed)
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		r := bytes.NewReader(data)
		value, err := readCompact(r)
		if err != nil {
			return
		}
		consumed := len(data) - r.Len()
		if consumed != compactSizeLen(value) {
			t.Fatalf("decoded %d bytes for %d", consumed, value)
		}
		var canonical bytes.Buffer
		if err := writeCompact(&canonical, value); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data[:consumed], canonical.Bytes()) {
			t.Fatalf("accepted non-canonical encoding %x for %d", data[:consumed], value)
		}
	})
}
