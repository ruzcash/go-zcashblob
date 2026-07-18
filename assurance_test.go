package zcashblob

import (
	"bytes"
	"errors"
	"io"
	"testing"
)

var errAssuranceRead = errors.New("assurance reader failure")

type assuranceErrorReader struct {
	prefix []byte
}

func (r *assuranceErrorReader) Read(p []byte) (int, error) {
	if len(r.prefix) == 0 {
		return 0, errAssuranceRead
	}
	n := copy(p, r.prefix)
	r.prefix = r.prefix[n:]
	return n, nil
}

type assuranceInvalidCountWriter struct {
	overreport bool
}

func (w assuranceInvalidCountWriter) Write(p []byte) (int, error) {
	if w.overreport {
		return len(p) + 1, nil
	}
	return -1, nil
}

type assuranceChunkWriter struct {
	buf   bytes.Buffer
	limit int
}

func (w *assuranceChunkWriter) Write(p []byte) (int, error) {
	if len(p) > w.limit {
		p = p[:w.limit]
	}
	return w.buf.Write(p)
}

func TestParseFromReaderAssurance(t *testing.T) {
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
		r := &assuranceErrorReader{prefix: append([]byte(nil), blob[:7]...)}
		if _, err := ParseFromReader(r); !errors.Is(err, errAssuranceRead) {
			t.Fatalf("got %v", err)
		}
	})
}

func TestSerializeErrorsAssurance(t *testing.T) {
	if blob, err := Serialize(nil); !errors.Is(err, ErrInvalidStructure) || blob != nil {
		t.Fatalf("nil transaction: blob=%x err=%v", blob, err)
	}
	if blob, err := Serialize(&Transaction{}); !errors.Is(err, ErrUnsupportedVersion) || blob != nil {
		t.Fatalf("unsupported transaction: blob=%x err=%v", blob, err)
	}
}

func TestSerializationValidationAssurance(t *testing.T) {
	// One shared allocation is enough to exercise every per-field size limit and
	// the aggregate transaction limit without retaining several 10 MiB buffers.
	large := make([]byte, MaxScriptSize+1)
	aggregatePart := large[:8<<20]

	tests := []struct {
		name string
		make func() *Transaction
		want error
	}{
		{
			name: "missing Overwinter flag",
			make: func() *Transaction {
				tx := emptyTx()
				tx.Header = Version5
				return tx
			},
			want: ErrUnsupportedVersion,
		},
		{
			name: "wrong version",
			make: func() *Transaction {
				tx := emptyTx()
				tx.Header = OverwinterFlag | 4
				return tx
			},
			want: ErrUnsupportedVersion,
		},
		{
			name: "wrong version group",
			make: func() *Transaction {
				tx := emptyTx()
				tx.VersionGroupID = 0
				return tx
			},
			want: ErrUnsupportedVersion,
		},
		{
			name: "too many elements",
			make: func() *Transaction {
				tx := emptyTx()
				tx.TransparentOutputs = make([]TxOut, MaxElements+1)
				return tx
			},
			want: ErrInvalidStructure,
		},
		{
			name: "oversized scriptSig",
			make: func() *Transaction {
				tx := emptyTx()
				tx.TransparentInputs = []TxIn{{ScriptSig: large}}
				return tx
			},
			want: ErrInvalidStructure,
		},
		{
			name: "oversized scriptPubKey",
			make: func() *Transaction {
				tx := emptyTx()
				tx.TransparentOutputs = []TxOut{{ScriptPubKey: large}}
				return tx
			},
			want: ErrInvalidStructure,
		},
		{
			name: "Sapling spend proof mismatch",
			make: func() *Transaction {
				tx := emptyTx()
				tx.Sapling.Spends = make([]SaplingSpend, 1)
				tx.Sapling.SpendAuthSigs = make([][64]byte, 1)
				return tx
			},
			want: ErrInvalidStructure,
		},
		{
			name: "Sapling spend auth mismatch",
			make: func() *Transaction {
				tx := emptyTx()
				tx.Sapling.Spends = make([]SaplingSpend, 1)
				tx.Sapling.SpendProofs = make([][192]byte, 1)
				return tx
			},
			want: ErrInvalidStructure,
		},
		{
			name: "Sapling output proof mismatch",
			make: func() *Transaction {
				tx := emptyTx()
				tx.Sapling.Outputs = make([]SaplingOutput, 1)
				return tx
			},
			want: ErrInvalidStructure,
		},
		{
			name: "Orchard auth mismatch",
			make: func() *Transaction {
				tx := emptyTx()
				tx.Orchard.Actions = make([]OrchardAction, 1)
				return tx
			},
			want: ErrInvalidStructure,
		},
		{
			name: "oversized Orchard proof",
			make: func() *Transaction {
				tx := emptyTx()
				tx.Orchard.Proofs = large
				return tx
			},
			want: ErrInvalidStructure,
		},
		{
			name: "reserved Orchard flags",
			make: func() *Transaction {
				tx := emptyTx()
				tx.Orchard.Flags = 0x04
				return tx
			},
			want: ErrInvalidStructure,
		},
		{
			name: "aggregate transaction size",
			make: func() *Transaction {
				tx := emptyTx()
				tx.TransparentInputs = []TxIn{{ScriptSig: aggregatePart}}
				tx.TransparentOutputs = []TxOut{{ScriptPubKey: aggregatePart}}
				return tx
			},
			want: ErrTooLarge,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var dst bytes.Buffer
			err := SerializeToWriter(test.make(), &dst)
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v, want %v", err, test.want)
			}
			if dst.Len() != 0 {
				t.Fatalf("wrote %d bytes before validation failed", dst.Len())
			}
		})
	}
}

func TestWriterContractAssurance(t *testing.T) {
	tx := populatedTx()
	want, err := Serialize(tx)
	if err != nil {
		t.Fatal(err)
	}

	for _, test := range []struct {
		name string
		w    io.Writer
	}{
		{"negative count", assuranceInvalidCountWriter{}},
		{"excessive count", assuranceInvalidCountWriter{overreport: true}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := SerializeToWriter(tx, test.w); !errors.Is(err, io.ErrShortWrite) {
				t.Fatalf("got %v", err)
			}
		})
	}

	t.Run("partial writes without error", func(t *testing.T) {
		w := &assuranceChunkWriter{limit: 3}
		if err := SerializeToWriter(tx, w); err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(w.buf.Bytes(), want) {
			t.Fatal("partial writer changed serialization")
		}
	})
}

func TestCompactSizeTruncationAssurance(t *testing.T) {
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

func TestCompactSizeLenAssurance(t *testing.T) {
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

func TestTransactionShapeAssurance(t *testing.T) {
	spendOnly := func() *Transaction {
		tx := emptyTx()
		tx.Sapling.Spends = make([]SaplingSpend, 1)
		tx.Sapling.SpendProofs = make([][192]byte, 1)
		tx.Sapling.SpendAuthSigs = make([][64]byte, 1)
		return tx
	}
	outputOnly := func() *Transaction {
		tx := emptyTx()
		tx.Sapling.Outputs = make([]SaplingOutput, 1)
		tx.Sapling.OutputProofs = make([][192]byte, 1)
		return tx
	}
	orchardOnly := func() *Transaction {
		tx := emptyTx()
		tx.Orchard.Actions = make([]OrchardAction, 1)
		tx.Orchard.Proofs = make([]byte, 253)
		tx.Orchard.SpendAuthSigs = make([][64]byte, 1)
		return tx
	}
	transparentOnly := func() *Transaction {
		tx := emptyTx()
		tx.TransparentInputs = []TxIn{{ScriptSig: make([]byte, 253)}}
		tx.TransparentOutputs = []TxOut{{ScriptPubKey: make([]byte, 0x10000)}}
		return tx
	}

	for _, test := range []struct {
		name string
		tx   *Transaction
	}{
		{"empty", emptyTx()},
		{"transparent only", transparentOnly()},
		{"Sapling spend only", spendOnly()},
		{"Sapling output only", outputOnly()},
		{"Orchard only", orchardOnly()},
		{"all pools", populatedTx()},
	} {
		t.Run(test.name, func(t *testing.T) {
			blob, err := Serialize(test.tx)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := encodedSize(test.tx), uint64(len(blob)); got != want {
				t.Fatalf("encodedSize = %d, serialized length = %d", got, want)
			}
			parsed, err := Parse(blob)
			if err != nil {
				t.Fatal(err)
			}
			if got, want := parsed.Hash(), parsed.TxID(); got != want {
				t.Fatalf("Hash = %x, TxID = %x", got, want)
			}
		})
	}
}

func TestBlake2bChunkSplitsAssurance(t *testing.T) {
	var personalization [16]byte
	copy(personalization[:], "AssuranceHash___")
	message := make([]byte, 3*128+1)
	for i := range message {
		message[i] = byte(i % 251)
	}
	want := blake2bPersonal(personalization, message)
	for split := 0; split <= len(message); split++ {
		got := blake2bPersonal(personalization, message[:split], nil, message[split:])
		if got != want {
			t.Fatalf("split %d: got %x, want %x", split, got, want)
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
