package zcashblob

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

var errTestReader = errors.New("test reader failure")

type errorReader struct {
	prefix []byte
}

func (r *errorReader) Read(p []byte) (int, error) {
	if len(r.prefix) == 0 {
		return 0, errTestReader
	}
	n := copy(p, r.prefix)
	r.prefix = r.prefix[n:]
	return n, nil
}

type invalidCountWriter struct {
	overreport bool
}

func (w invalidCountWriter) Write(p []byte) (int, error) {
	if w.overreport {
		return len(p) + 1, nil
	}
	return -1, nil
}

type chunkWriter struct {
	buf   bytes.Buffer
	limit int
}

func (w *chunkWriter) Write(p []byte) (int, error) {
	if len(p) > w.limit {
		p = p[:w.limit]
	}
	return w.buf.Write(p)
}

func TestParseFromReader(t *testing.T) {
	blob, err := Serialize(populatedTx())
	if err != nil {
		t.Fatal(err)
	}

	t.Run("valid", func(t *testing.T) {
		tx, err := ParseFromReader(bytes.NewReader(blob))
		if err != nil {
			t.Fatal(err)
		}
		got, err := Serialize(tx)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, blob) {
			t.Fatal("reader round-trip changed bytes")
		}
	})

	t.Run("read error", func(t *testing.T) {
		r := &errorReader{prefix: append([]byte(nil), blob[:7]...)}
		if _, err := ParseFromReader(r); !errors.Is(err, errTestReader) {
			t.Fatalf("got %v", err)
		}
	})
}

func TestSerializeToWriterHonorsWriterContract(t *testing.T) {
	tx := populatedTx()
	want, err := Serialize(tx)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name string
		w    io.Writer
	}{
		{"negative count", invalidCountWriter{}},
		{"excessive count", invalidCountWriter{overreport: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := SerializeToWriter(tx, test.w); !errors.Is(err, io.ErrShortWrite) {
				t.Fatalf("got %v", err)
			}
		})
	}

	t.Run("partial writes without error", func(t *testing.T) {
		w := &chunkWriter{limit: 3}
		if err := SerializeToWriter(tx, w); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(w.buf.Bytes(), want) {
			t.Fatal("partial writer changed serialization")
		}
	})
}
