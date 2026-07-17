package zcashblob

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io"
	"testing"
)

var errInjected = errors.New("injected writer failure")

type failAfterWriter struct{ remaining int }

func (w *failAfterWriter) Write(p []byte) (int, error) {
	if w.remaining == 0 {
		return 0, errInjected
	}
	if len(p) > w.remaining {
		n := w.remaining
		w.remaining = 0
		return n, errInjected
	}
	w.remaining -= len(p)
	return len(p), nil
}

type zeroWriter struct{}

func (zeroWriter) Write([]byte) (int, error) { return 0, nil }

func populatedTx() *Transaction {
	tx := emptyTx()
	tx.TransparentInputs = []TxIn{{ScriptSig: []byte{1, 2, 3}, Sequence: 42}}
	tx.TransparentOutputs = []TxOut{{Value: 123, ScriptPubKey: []byte{0x51}}}
	tx.Sapling.Spends = make([]SaplingSpend, 1)
	tx.Sapling.SpendProofs = make([][192]byte, 1)
	tx.Sapling.SpendAuthSigs = make([][64]byte, 1)
	tx.Sapling.Outputs = make([]SaplingOutput, 1)
	tx.Sapling.OutputProofs = make([][192]byte, 1)
	tx.Orchard.Actions = make([]OrchardAction, 1)
	tx.Orchard.Proofs = []byte{4, 5}
	tx.Orchard.SpendAuthSigs = make([][64]byte, 1)
	return tx
}

func TestSerializePropagatesEveryWriterFailure(t *testing.T) {
	tx := populatedTx()
	blob, err := Serialize(tx)
	if err != nil {
		t.Fatal(err)
	}
	for offset := 0; offset < len(blob); offset++ {
		w := &failAfterWriter{remaining: offset}
		if err := SerializeToWriter(tx, w); !errors.Is(err, errInjected) {
			t.Fatalf("offset %d: got %v", offset, err)
		}
	}
	if err := SerializeToWriter(tx, zeroWriter{}); !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("zero writer: got %v", err)
	}
}

func TestSerializeValidatesBeforeWriting(t *testing.T) {
	tx := populatedTx()
	tx.Sapling.SpendProofs = nil
	var dst bytes.Buffer
	err := SerializeToWriter(tx, &dst)
	if !errors.Is(err, ErrInvalidStructure) {
		t.Fatalf("got %v", err)
	}
	if dst.Len() != 0 {
		t.Fatalf("wrote %d bytes before validation failed", dst.Len())
	}

	tx = emptyTx()
	tx.TransparentOutputs = []TxOut{{ScriptPubKey: make([]byte, MaxScriptSize+1)}}
	err = SerializeToWriter(tx, &dst)
	if !errors.Is(err, ErrInvalidStructure) || dst.Len() != 0 {
		t.Fatalf("oversized script: err=%v bytes=%d", err, dst.Len())
	}
}

func TestNilReaderAndWriter(t *testing.T) {
	if _, err := ParseFromReader(nil); !errors.Is(err, ErrNilReader) {
		t.Fatalf("nil reader: %v", err)
	}
	if err := SerializeToWriter(emptyTx(), nil); !errors.Is(err, ErrNilWriter) {
		t.Fatalf("nil writer: %v", err)
	}
}

func TestReaderAndSliceSizeLimits(t *testing.T) {
	over := make([]byte, MaxTransactionSize+1)
	if _, err := Parse(over); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("Parse: %v", err)
	}
	if _, err := ParseFromReader(bytes.NewReader(over)); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("ParseFromReader: %v", err)
	}
}

func TestDeclaredFixedVectorsArePreflighted(t *testing.T) {
	blob, err := Serialize(emptyTx())
	if err != nil {
		t.Fatal(err)
	}
	// Replace the Sapling output count with 65,535 and provide no elements.
	malformed := append([]byte(nil), blob[:23]...)
	malformed = append(malformed, 0xfd, 0xff, 0xff)
	if _, err := Parse(malformed); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Sapling count: %v", err)
	}
	// Reach the Orchard count with all preceding vectors empty.
	malformed = append([]byte(nil), blob[:24]...)
	malformed = append(malformed, 0xfd, 0xff, 0xff)
	if _, err := Parse(malformed); !errors.Is(err, io.ErrUnexpectedEOF) {
		t.Fatalf("Orchard count: %v", err)
	}
}

func TestRejectsReservedOrchardFlags(t *testing.T) {
	tx := emptyTx()
	tx.Orchard.Actions = make([]OrchardAction, 1)
	tx.Orchard.SpendAuthSigs = make([][64]byte, 1)
	blob, err := Serialize(tx)
	if err != nil {
		t.Fatal(err)
	}
	const flagsOffset = 20 + 5 + 820
	blob[flagsOffset] = 0x80
	if _, err := Parse(blob); !errors.Is(err, ErrInvalidStructure) {
		t.Fatalf("got %v", err)
	}
}

func TestEveryTruncationIsRejected(t *testing.T) {
	blob, err := Serialize(populatedTx())
	if err != nil {
		t.Fatal(err)
	}
	for n := range blob {
		if _, err := Parse(blob[:n]); err == nil {
			t.Fatalf("accepted truncation at %d of %d", n, len(blob))
		}
	}
}

func TestCompactSizeCanonicalBoundaries(t *testing.T) {
	values := []uint64{0, 252, 253, 0xffff, 0x10000, 0xffffffff, 0x100000000}
	for _, want := range values {
		var b bytes.Buffer
		if err := writeCompact(&b, want); err != nil {
			t.Fatal(err)
		}
		got, err := readCompact(&b)
		if err != nil || got != want {
			t.Fatalf("%d: got %d, %v", want, got, err)
		}
	}
	bad := [][]byte{{0xfd, 0xfc, 0x00}, {0xfe, 0xff, 0xff, 0x00, 0x00}, {0xff, 0xff, 0xff, 0xff, 0xff, 0, 0, 0, 0}}
	for _, encoded := range bad {
		if _, err := readCompact(bytes.NewReader(encoded)); !errors.Is(err, ErrNonCanonical) {
			t.Fatalf("%x: got %v", encoded, err)
		}
	}
}

func TestPersonalizedBlake2bBlockBoundaries(t *testing.T) {
	vectors := map[int]string{
		0:   "e6f9967555b66ebd3bd806f976a6d2b559dbd87a587e0ab738d1c4d90332e695",
		1:   "eaae7fabca668a8de609ed77329ae198c341fcfa18a40fcd0a6c601afb2dc67c",
		127: "4d5b4e98ad6479410078c6f502ffa2b778af8080081ecb761942ef93290ca953",
		128: "017eeab5695d72917c5352a5d179242630293f7d740697eda0694592c8c035cc",
		129: "d17e65c54832cd4a5a8429fced29c74ffccb3e0636b695268151058da21dd688",
		257: "f4ccc349b49883b7cd8afce325b9be59853037b6a915d17f1462cda92d7ad1e5",
	}
	for n, wantHex := range vectors {
		msg := make([]byte, n)
		for i := range msg {
			msg[i] = byte(i % 251)
		}
		want, _ := hex.DecodeString(wantHex)
		got := personal("ZTxIdHeadersHash", msg)
		if !bytes.Equal(got[:], want) {
			t.Fatalf("length %d: got %x", n, got)
		}
	}
}
