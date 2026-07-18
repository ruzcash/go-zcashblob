package zcashblob

import (
	"bytes"
	"errors"
	"testing"
)

func TestSerializeRejectsInvalidTransactions(t *testing.T) {
	if blob, err := Serialize(nil); !errors.Is(err, ErrInvalidStructure) || blob != nil {
		t.Fatalf("nil transaction: blob=%x err=%v", blob, err)
	}
	if blob, err := Serialize(&Transaction{}); !errors.Is(err, ErrUnsupportedVersion) || blob != nil {
		t.Fatalf("unsupported transaction: blob=%x err=%v", blob, err)
	}
}

func TestSerializationStructureLimits(t *testing.T) {
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
			want: ErrTooLarge,
		},
		{
			name: "oversized scriptSig",
			make: func() *Transaction {
				tx := emptyTx()
				tx.TransparentInputs = []TxIn{{ScriptSig: large}}
				return tx
			},
			want: ErrTooLarge,
		},
		{
			name: "oversized scriptPubKey",
			make: func() *Transaction {
				tx := emptyTx()
				tx.TransparentOutputs = []TxOut{{ScriptPubKey: large}}
				return tx
			},
			want: ErrTooLarge,
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
			want: ErrTooLarge,
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
