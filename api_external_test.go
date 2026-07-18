package zcashblob_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"

	"github.com/ruzcash/go-zcashblob"
)

const nu5BranchID = 0xc2d6d0b4

func TestNewTransactionV5AndValidate(t *testing.T) {
	tx := zcashblob.NewTransactionV5(nu5BranchID)
	if tx.Header != zcashblob.OverwinterFlag|zcashblob.Version5 {
		t.Fatalf("header = %#x", tx.Header)
	}
	if tx.Version() != zcashblob.Version5 {
		t.Fatalf("version = %d", tx.Version())
	}
	if tx.VersionGroupID != zcashblob.VersionGroupIDV5 {
		t.Fatalf("version group = %#x", tx.VersionGroupID)
	}
	if tx.ConsensusBranchID != nu5BranchID {
		t.Fatalf("branch ID = %#x", tx.ConsensusBranchID)
	}
	if err := tx.Validate(); err != nil {
		t.Fatalf("empty v5 transaction: %v", err)
	}

	tx.Sapling.Spends = make([]zcashblob.SaplingSpend, 1)
	if err := tx.Validate(); !errors.Is(err, zcashblob.ErrInvalidStructure) {
		t.Fatalf("mismatched Sapling authorization vectors: %v", err)
	}

	var nilTx *zcashblob.Transaction
	if err := nilTx.Validate(); !errors.Is(err, zcashblob.ErrInvalidStructure) {
		t.Fatalf("nil transaction: %v", err)
	}
}

func TestValidateRejectsUnencodedConditionalFields(t *testing.T) {
	nonzero32 := [32]byte{1}
	nonzero64 := [64]byte{1}

	tests := []struct {
		name   string
		change func(*zcashblob.Transaction)
	}{
		{"Sapling value balance without bundle", func(tx *zcashblob.Transaction) {
			tx.Sapling.ValueBalance = 1
		}},
		{"Sapling binding signature without bundle", func(tx *zcashblob.Transaction) {
			tx.Sapling.BindingSig = nonzero64
		}},
		{"Sapling anchor without spends", func(tx *zcashblob.Transaction) {
			tx.Sapling.Outputs = make([]zcashblob.SaplingOutput, 1)
			tx.Sapling.OutputProofs = make([][192]byte, 1)
			tx.Sapling.Anchor = nonzero32
		}},
		{"Orchard flags without actions", func(tx *zcashblob.Transaction) {
			tx.Orchard.Flags = 1
		}},
		{"Orchard value balance without actions", func(tx *zcashblob.Transaction) {
			tx.Orchard.ValueBalance = 1
		}},
		{"Orchard anchor without actions", func(tx *zcashblob.Transaction) {
			tx.Orchard.Anchor = nonzero32
		}},
		{"Orchard proof without actions", func(tx *zcashblob.Transaction) {
			tx.Orchard.Proofs = []byte{1}
		}},
		{"Orchard binding signature without actions", func(tx *zcashblob.Transaction) {
			tx.Orchard.BindingSig = nonzero64
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tx := zcashblob.NewTransactionV5(nu5BranchID)
			test.change(tx)
			if err := tx.Validate(); !errors.Is(err, zcashblob.ErrInvalidStructure) {
				t.Fatalf("Validate() = %v", err)
			}
		})
	}
}

func TestReaderWriterAPI(t *testing.T) {
	want := zcashblob.NewTransactionV5(nu5BranchID)
	want.TransparentOutputs = []zcashblob.TxOut{{
		Value:        12345,
		ScriptPubKey: []byte{0x51},
	}}

	var wire bytes.Buffer
	if err := zcashblob.SerializeToWriter(want, &wire); err != nil {
		t.Fatal(err)
	}
	wireBytes := bytes.Clone(wire.Bytes())

	got, err := zcashblob.ParseFromReader(&wire)
	if err != nil {
		t.Fatal(err)
	}
	if err := got.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(got.TransparentOutputs) != 1 ||
		got.TransparentOutputs[0].Value != 12345 ||
		!bytes.Equal(got.TransparentOutputs[0].ScriptPubKey, []byte{0x51}) {
		t.Fatalf("unexpected transparent output: %+v", got.TransparentOutputs)
	}

	again, err := zcashblob.Serialize(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, wireBytes) {
		t.Fatal("reader/writer round trip changed the wire encoding")
	}
}

func TestTxIDStringDisplayOrder(t *testing.T) {
	tx := zcashblob.NewTransactionV5(nu5BranchID)
	const want = "8e6b6d721fc653ef162daa85b32bff85144b9245add517cf710d5155cf5876df"
	if got := tx.TxIDString(); got != want {
		t.Fatalf("TxIDString() = %s", got)
	}

	digest := tx.TxID()
	display, err := hex.DecodeString(want)
	if err != nil {
		t.Fatal(err)
	}
	for i := range digest {
		if digest[i] != display[len(display)-1-i] {
			t.Fatalf("display byte %d is not reversed", i)
		}
	}
}
