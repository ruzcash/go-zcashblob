package zcashblob

import "testing"

func TestEncodedSizeAcrossTransactionShapes(t *testing.T) {
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
