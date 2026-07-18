package zcashblob

import (
	"encoding/binary"
	"io"
)

func readCompact(r io.Reader) (uint64, error) {
	var b [9]byte
	if _, err := io.ReadFull(r, b[:1]); err != nil {
		return 0, err
	}
	switch b[0] {
	case 0xfd:
		if _, err := io.ReadFull(r, b[:2]); err != nil {
			return 0, err
		}
		v := uint64(binary.LittleEndian.Uint16(b[:2]))
		if v < 253 {
			return 0, ErrNonCanonical
		}
		return v, nil
	case 0xfe:
		if _, err := io.ReadFull(r, b[:4]); err != nil {
			return 0, err
		}
		v := uint64(binary.LittleEndian.Uint32(b[:4]))
		if v <= 0xffff {
			return 0, ErrNonCanonical
		}
		return v, nil
	case 0xff:
		if _, err := io.ReadFull(r, b[:8]); err != nil {
			return 0, err
		}
		v := binary.LittleEndian.Uint64(b[:8])
		if v <= 0xffffffff {
			return 0, ErrNonCanonical
		}
		return v, nil
	default:
		return uint64(b[0]), nil
	}
}

func writeCompact(w io.Writer, v uint64) error {
	var b [9]byte
	var out []byte
	switch {
	case v < 253:
		b[0] = byte(v)
		out = b[:1]
	case v <= 0xffff:
		b[0] = 0xfd
		binary.LittleEndian.PutUint16(b[1:], uint16(v))
		out = b[:3]
	case v <= 0xffffffff:
		b[0] = 0xfe
		binary.LittleEndian.PutUint32(b[1:], uint32(v))
		out = b[:5]
	default:
		b[0] = 0xff
		binary.LittleEndian.PutUint64(b[1:], v)
		out = b[:9]
	}
	return writeAll(w, out)
}

func writeAll(w io.Writer, p []byte) error {
	for len(p) > 0 {
		n, err := w.Write(p)
		if n < 0 || n > len(p) {
			return io.ErrShortWrite
		}
		p = p[n:]
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
	}
	return nil
}

func compactSizeLen(v uint64) int {
	switch {
	case v < 253:
		return 1
	case v <= 0xffff:
		return 3
	case v <= 0xffffffff:
		return 5
	default:
		return 9
	}
}
