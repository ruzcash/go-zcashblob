package zcashblob

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"
)

func compactEncoding(t *testing.T, value uint64) []byte {
	t.Helper()
	var encoded bytes.Buffer
	if err := writeCompact(&encoded, value); err != nil {
		t.Fatal(err)
	}
	return encoded.Bytes()
}

func emptyHeader(t *testing.T) []byte {
	t.Helper()
	blob, err := Serialize(emptyTx())
	if err != nil {
		t.Fatal(err)
	}
	return bytes.Clone(blob[:20])
}

func TestParseRejectsUnsupportedHeaders(t *testing.T) {
	valid, err := Serialize(emptyTx())
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		header uint32
		group  uint32
	}{
		{"missing Overwinter flag", Version5, VersionGroupIDV5},
		{"version 4", OverwinterFlag | 4, VersionGroupIDV5},
		{"version 6", OverwinterFlag | 6, VersionGroupIDV5},
		{"wrong version group", OverwinterFlag | Version5, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			blob := bytes.Clone(valid)
			binary.LittleEndian.PutUint32(blob[:4], tc.header)
			binary.LittleEndian.PutUint32(blob[4:8], tc.group)
			if _, err := Parse(blob); !errors.Is(err, ErrUnsupportedVersion) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestParseRejectsHostileDeclaredCounts(t *testing.T) {
	// Each preceding zero is the canonical empty count for an earlier bundle.
	// The oversized count is rejected before any element slice is allocated.
	for priorCounts, name := range []string{
		"transparent inputs",
		"transparent outputs",
		"Sapling spends",
		"Sapling outputs",
		"Orchard actions",
	} {
		t.Run(name, func(t *testing.T) {
			blob := emptyHeader(t)
			blob = append(blob, make([]byte, priorCounts)...)
			blob = append(blob, compactEncoding(t, MaxElements+1)...)
			if _, err := Parse(blob); !errors.Is(err, ErrTooLarge) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestParseRejectsHostileVariableLengths(t *testing.T) {
	t.Run("scriptSig", func(t *testing.T) {
		blob := emptyHeader(t)
		blob = append(blob, 1)                     // one transparent input
		blob = append(blob, make([]byte, 32+4)...) // outpoint
		blob = append(blob, compactEncoding(t, MaxScriptSize+1)...)
		if _, err := Parse(blob); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("scriptPubKey", func(t *testing.T) {
		blob := emptyHeader(t)
		blob = append(blob, 0, 1)               // no inputs, one output
		blob = append(blob, make([]byte, 8)...) // value
		blob = append(blob, compactEncoding(t, MaxScriptSize+1)...)
		if _, err := Parse(blob); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("Orchard proof", func(t *testing.T) {
		tx := emptyTx()
		tx.Orchard.Actions = make([]OrchardAction, 1)
		tx.Orchard.SpendAuthSigs = make([][64]byte, 1)
		blob, err := Serialize(tx)
		if err != nil {
			t.Fatal(err)
		}
		const proofLengthOffset = 20 + 5 + 820 + 1 + 8 + 32
		malformed := bytes.Clone(blob[:proofLengthOffset])
		malformed = append(malformed, compactEncoding(t, MaxProofSize+1)...)
		if _, err := Parse(malformed); !errors.Is(err, ErrTooLarge) {
			t.Fatalf("got %v", err)
		}
	})

	t.Run("non-canonical public path", func(t *testing.T) {
		blob := emptyHeader(t)
		blob = append(blob, 0xfd, 0xfc, 0x00)
		if _, err := Parse(blob); !errors.Is(err, ErrNonCanonical) {
			t.Fatalf("got %v", err)
		}
	})
}
