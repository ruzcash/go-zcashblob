package zcashblob

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"testing"
)

func emptyTx() *Transaction {
	return &Transaction{Header: OverwinterFlag | Version5, VersionGroupID: VersionGroupIDV5, ConsensusBranchID: 0xc2d6d0b4}
}

func TestRoundTripEmptyV5(t *testing.T) {
	want := emptyTx()
	blob, err := Serialize(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Parse(blob)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Serialize(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(blob, again) {
		t.Fatal("round-trip changed bytes")
	}
}

func TestRoundTripAllPools(t *testing.T) {
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
	blob, err := Serialize(tx)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := Parse(blob)
	if err != nil {
		t.Fatal(err)
	}
	again, err := Serialize(parsed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(blob, again) {
		t.Fatal("round-trip changed bytes")
	}
}

func TestEffectingAndAuthorizingDigestsAreSeparated(t *testing.T) {
	tx := populatedTx()
	txid := tx.TxID()
	auth := tx.AuthDigest()

	tx.Sapling.BindingSig[0] ^= 1
	if got := tx.TxID(); got != txid {
		t.Fatal("authorizing data changed the non-malleable txid")
	}
	if got := tx.AuthDigest(); got == auth {
		t.Fatal("authorizing-data mutation did not change AuthDigest")
	}

	tx.TransparentOutputs[0].Value++
	if got := tx.TxID(); got == txid {
		t.Fatal("effecting-data mutation did not change TxID")
	}
}

func TestPersonalizedBlake2b(t *testing.T) {
	got := personal("ZTxIdHeadersHash")
	want, err := hex.DecodeString("e6f9967555b66ebd3bd806f976a6d2b559dbd87a587e0ab738d1c4d90332e695")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got[:], want) {
		t.Fatalf("got %x", got)
	}
}

func TestRejectsTrailingData(t *testing.T) {
	blob, err := Serialize(emptyTx())
	if err != nil {
		t.Fatal(err)
	}
	blob = append(blob, 0)
	if _, err := Parse(blob); !errors.Is(err, ErrTrailingData) {
		t.Fatalf("got %v", err)
	}
}

func FuzzParse(f *testing.F) {
	empty, err := Serialize(emptyTx())
	if err != nil {
		f.Fatal(err)
	}
	f.Add(empty)
	populated, err := Serialize(populatedTx())
	if err != nil {
		f.Fatal(err)
	}
	f.Add(populated)

	if got := sha256.Sum256(zip244Corpus); hex.EncodeToString(got[:]) != zip244CorpusSHA256 {
		f.Fatal("embedded ZIP-244 corpus checksum mismatch")
	}
	var rows []json.RawMessage
	if err := json.Unmarshal(zip244Corpus, &rows); err != nil {
		f.Fatal(err)
	}
	if len(rows) != zip244VectorCount+2 {
		f.Fatalf("ZIP-244 corpus has %d rows", len(rows))
	}
	for i, rawRow := range rows[2:] {
		var fields []json.RawMessage
		if err := json.Unmarshal(rawRow, &fields); err != nil {
			f.Fatalf("vector %d: %v", i, err)
		}
		if len(fields) != zip244FieldCount {
			f.Fatalf("vector %d has %d fields", i, len(fields))
		}
		f.Add(decodeZIP244HexField(f, fields[0], "tx"))
	}

	f.Fuzz(func(t *testing.T, data []byte) {
		tx, err := Parse(data)
		if err != nil {
			return
		}
		txid, auth := tx.TxID(), tx.AuthDigest()
		out, err := Serialize(tx)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, out) {
			t.Fatal("lossy round-trip")
		}
		reparsed, err := Parse(out)
		if err != nil {
			t.Fatal(err)
		}
		if reparsed.TxID() != txid || reparsed.AuthDigest() != auth {
			t.Fatal("digest changed across round-trip")
		}
	})
}
