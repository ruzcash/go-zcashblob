package zcashblob

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"testing"
)

// allFieldsTx uses distinct, non-zero data in every field. This makes the
// semantic round trip sensitive to same-sized fields being reordered or
// assigned to the wrong destination.
func allFieldsTx() *Transaction {
	tx := emptyTx()
	tx.LockTime = 0x10203040
	tx.ExpiryHeight = 0x50607080

	nextTag := byte(1)
	fill := func(dst []byte) {
		for i := range dst {
			dst[i] = nextTag + byte(i*29)
		}
		nextTag += 17
	}

	tx.TransparentInputs = make([]TxIn, 2)
	for i := range tx.TransparentInputs {
		in := &tx.TransparentInputs[i]
		fill(in.PreviousOutput.Hash[:])
		in.PreviousOutput.Index = uint32(0x11223344 + i)
		in.ScriptSig = make([]byte, 3+i*250)
		fill(in.ScriptSig)
		in.Sequence = uint32(0xa0b0c0d0 + i)
	}
	tx.TransparentOutputs = make([]TxOut, 2)
	for i := range tx.TransparentOutputs {
		out := &tx.TransparentOutputs[i]
		out.Value = []int64{123456789, -987654321}[i]
		out.ScriptPubKey = make([]byte, 5+i*248)
		fill(out.ScriptPubKey)
	}

	tx.Sapling.Spends = make([]SaplingSpend, 2)
	tx.Sapling.SpendProofs = make([][192]byte, 2)
	tx.Sapling.SpendAuthSigs = make([][64]byte, 2)
	for i := range tx.Sapling.Spends {
		spend := &tx.Sapling.Spends[i]
		fill(spend.CV[:])
		fill(spend.Nullifier[:])
		fill(spend.RK[:])
		fill(tx.Sapling.SpendProofs[i][:])
		fill(tx.Sapling.SpendAuthSigs[i][:])
	}
	tx.Sapling.Outputs = make([]SaplingOutput, 2)
	tx.Sapling.OutputProofs = make([][192]byte, 2)
	for i := range tx.Sapling.Outputs {
		out := &tx.Sapling.Outputs[i]
		fill(out.CV[:])
		fill(out.CMU[:])
		fill(out.EphemeralKey[:])
		fill(out.EncCiphertext[:])
		fill(out.OutCiphertext[:])
		fill(tx.Sapling.OutputProofs[i][:])
	}
	tx.Sapling.ValueBalance = -1122334455
	fill(tx.Sapling.Anchor[:])
	fill(tx.Sapling.BindingSig[:])

	tx.Orchard.Actions = make([]OrchardAction, 2)
	tx.Orchard.SpendAuthSigs = make([][64]byte, 2)
	for i := range tx.Orchard.Actions {
		action := &tx.Orchard.Actions[i]
		fill(action.CV[:])
		fill(action.Nullifier[:])
		fill(action.RK[:])
		fill(action.CMX[:])
		fill(action.EphemeralKey[:])
		fill(action.EncCiphertext[:])
		fill(action.OutCiphertext[:])
		fill(tx.Orchard.SpendAuthSigs[i][:])
	}
	tx.Orchard.Flags = 3
	tx.Orchard.ValueBalance = 9988776655
	fill(tx.Orchard.Anchor[:])
	tx.Orchard.Proofs = make([]byte, 253)
	fill(tx.Orchard.Proofs)
	fill(tx.Orchard.BindingSig[:])
	return tx
}

func TestAllFieldsSemanticRoundTrip(t *testing.T) {
	want := allFieldsTx()
	blob, err := Serialize(want)
	if err != nil {
		t.Fatal(err)
	}

	const wantWireSHA256 = "b2486af20e5dd54760f9d3b4441201a429a50fe76ca22a085d55dcf5d825aa3e"
	gotWireSHA256 := sha256.Sum256(blob)
	if hex.EncodeToString(gotWireSHA256[:]) != wantWireSHA256 {
		t.Fatalf("wire-format oracle changed: got %x", gotWireSHA256)
	}

	got, err := Parse(blob)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatal("semantic round trip changed transaction fields")
	}
	again, err := Serialize(got)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(again, blob) {
		t.Fatal("second serialization changed wire bytes")
	}
}

func cloneTransaction(t *testing.T, tx *Transaction) *Transaction {
	t.Helper()
	blob, err := Serialize(tx)
	if err != nil {
		t.Fatal(err)
	}
	clone, err := Parse(blob)
	if err != nil {
		t.Fatal(err)
	}
	return clone
}

func TestZIP244CommitmentFieldMatrix(t *testing.T) {
	base := allFieldsTx()
	baseTxID, baseAuth := base.TxID(), base.AuthDigest()
	type mutation struct {
		name       string
		change     func(*Transaction)
		txid, auth bool
	}
	flip := func(p *byte) { *p ^= 1 }
	cases := []mutation{
		{"header", func(tx *Transaction) { tx.Header ^= 1 << 8 }, true, false},
		{"version group", func(tx *Transaction) { tx.VersionGroupID ^= 1 }, true, false},
		{"consensus branch", func(tx *Transaction) { tx.ConsensusBranchID ^= 1 }, true, true},
		{"lock time", func(tx *Transaction) { tx.LockTime ^= 1 }, true, false},
		{"expiry height", func(tx *Transaction) { tx.ExpiryHeight ^= 1 }, true, false},
		{"transparent prevout hash", func(tx *Transaction) { flip(&tx.TransparentInputs[0].PreviousOutput.Hash[0]) }, true, false},
		{"transparent prevout index", func(tx *Transaction) { tx.TransparentInputs[0].PreviousOutput.Index ^= 1 }, true, false},
		{"transparent scriptSig", func(tx *Transaction) { flip(&tx.TransparentInputs[0].ScriptSig[0]) }, false, true},
		{"transparent sequence", func(tx *Transaction) { tx.TransparentInputs[0].Sequence ^= 1 }, true, false},
		{"transparent value", func(tx *Transaction) { tx.TransparentOutputs[0].Value ^= 1 }, true, false},
		{"transparent scriptPubKey", func(tx *Transaction) { flip(&tx.TransparentOutputs[0].ScriptPubKey[0]) }, true, false},
		{"Sapling spend cv", func(tx *Transaction) { flip(&tx.Sapling.Spends[0].CV[0]) }, true, false},
		{"Sapling spend nullifier", func(tx *Transaction) { flip(&tx.Sapling.Spends[0].Nullifier[0]) }, true, false},
		{"Sapling spend rk", func(tx *Transaction) { flip(&tx.Sapling.Spends[0].RK[0]) }, true, false},
		{"Sapling value balance", func(tx *Transaction) { tx.Sapling.ValueBalance ^= 1 }, true, false},
		{"Sapling anchor", func(tx *Transaction) { flip(&tx.Sapling.Anchor[0]) }, true, false},
		{"Sapling spend proof", func(tx *Transaction) { flip(&tx.Sapling.SpendProofs[0][0]) }, false, true},
		{"Sapling spend auth sig", func(tx *Transaction) { flip(&tx.Sapling.SpendAuthSigs[0][0]) }, false, true},
		{"Sapling output cv", func(tx *Transaction) { flip(&tx.Sapling.Outputs[0].CV[0]) }, true, false},
		{"Sapling output cmu", func(tx *Transaction) { flip(&tx.Sapling.Outputs[0].CMU[0]) }, true, false},
		{"Sapling output ephemeral key", func(tx *Transaction) { flip(&tx.Sapling.Outputs[0].EphemeralKey[0]) }, true, false},
		{"Sapling ciphertext compact boundary 51", func(tx *Transaction) { flip(&tx.Sapling.Outputs[0].EncCiphertext[51]) }, true, false},
		{"Sapling ciphertext memo boundary 52", func(tx *Transaction) { flip(&tx.Sapling.Outputs[0].EncCiphertext[52]) }, true, false},
		{"Sapling ciphertext memo boundary 563", func(tx *Transaction) { flip(&tx.Sapling.Outputs[0].EncCiphertext[563]) }, true, false},
		{"Sapling ciphertext noncompact boundary 564", func(tx *Transaction) { flip(&tx.Sapling.Outputs[0].EncCiphertext[564]) }, true, false},
		{"Sapling out ciphertext", func(tx *Transaction) { flip(&tx.Sapling.Outputs[0].OutCiphertext[0]) }, true, false},
		{"Sapling output proof", func(tx *Transaction) { flip(&tx.Sapling.OutputProofs[0][0]) }, false, true},
		{"Sapling binding sig", func(tx *Transaction) { flip(&tx.Sapling.BindingSig[0]) }, false, true},
		{"Orchard action cv", func(tx *Transaction) { flip(&tx.Orchard.Actions[0].CV[0]) }, true, false},
		{"Orchard action nullifier", func(tx *Transaction) { flip(&tx.Orchard.Actions[0].Nullifier[0]) }, true, false},
		{"Orchard action rk", func(tx *Transaction) { flip(&tx.Orchard.Actions[0].RK[0]) }, true, false},
		{"Orchard action cmx", func(tx *Transaction) { flip(&tx.Orchard.Actions[0].CMX[0]) }, true, false},
		{"Orchard action ephemeral key", func(tx *Transaction) { flip(&tx.Orchard.Actions[0].EphemeralKey[0]) }, true, false},
		{"Orchard ciphertext compact boundary 51", func(tx *Transaction) { flip(&tx.Orchard.Actions[0].EncCiphertext[51]) }, true, false},
		{"Orchard ciphertext memo boundary 52", func(tx *Transaction) { flip(&tx.Orchard.Actions[0].EncCiphertext[52]) }, true, false},
		{"Orchard ciphertext memo boundary 563", func(tx *Transaction) { flip(&tx.Orchard.Actions[0].EncCiphertext[563]) }, true, false},
		{"Orchard ciphertext noncompact boundary 564", func(tx *Transaction) { flip(&tx.Orchard.Actions[0].EncCiphertext[564]) }, true, false},
		{"Orchard out ciphertext", func(tx *Transaction) { flip(&tx.Orchard.Actions[0].OutCiphertext[0]) }, true, false},
		{"Orchard flags", func(tx *Transaction) { tx.Orchard.Flags ^= 1 }, true, false},
		{"Orchard value balance", func(tx *Transaction) { tx.Orchard.ValueBalance ^= 1 }, true, false},
		{"Orchard anchor", func(tx *Transaction) { flip(&tx.Orchard.Anchor[0]) }, true, false},
		{"Orchard proof", func(tx *Transaction) { flip(&tx.Orchard.Proofs[0]) }, false, true},
		{"Orchard spend auth sig", func(tx *Transaction) { flip(&tx.Orchard.SpendAuthSigs[0][0]) }, false, true},
		{"Orchard binding sig", func(tx *Transaction) { flip(&tx.Orchard.BindingSig[0]) }, false, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tx := cloneTransaction(t, base)
			tc.change(tx)
			if changed := tx.TxID() != baseTxID; changed != tc.txid {
				t.Fatalf("TxID changed=%v, want %v", changed, tc.txid)
			}
			if changed := tx.AuthDigest() != baseAuth; changed != tc.auth {
				t.Fatalf("AuthDigest changed=%v, want %v", changed, tc.auth)
			}
		})
	}
}

func TestZIP244DigestPairCommitsToEveryAcceptedByte(t *testing.T) {
	baseBlob, err := Serialize(allFieldsTx())
	if err != nil {
		t.Fatal(err)
	}
	base, err := Parse(baseBlob)
	if err != nil {
		t.Fatal(err)
	}
	baseTxID, baseAuth := base.TxID(), base.AuthDigest()

	accepted := 0
	for i := range baseBlob {
		mutated := bytes.Clone(baseBlob)
		mutated[i] ^= 1
		tx, err := Parse(mutated)
		if err != nil {
			continue
		}
		accepted++
		if tx.TxID() == baseTxID && tx.AuthDigest() == baseAuth {
			t.Fatalf("accepted byte mutation at offset %d changed neither digest", i)
		}
	}
	if accepted == 0 {
		t.Fatal("test did not produce any accepted byte mutations")
	}
}
