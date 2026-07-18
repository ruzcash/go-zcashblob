package zcashblob

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
)

func personal(name string, data ...[]byte) [32]byte {
	var p [16]byte
	copy(p[:], name)
	return blake2bPersonal(p, data...)
}

// blake2bPersonal implements BLAKE2b-256 personalization. Personalization is
// part of the parameter block (it is not a prefix to the hashed message).
func blake2bPersonal(person [16]byte, chunks ...[]byte) [32]byte {
	return sumPersonal(person, chunks...)
}

func le32(v uint32) []byte { var b [4]byte; binary.LittleEndian.PutUint32(b[:], v); return b[:] }
func le64(v int64) []byte  { var b [8]byte; binary.LittleEndian.PutUint64(b[:], uint64(v)); return b[:] }

// TxID computes the non-malleable ZIP-244 transaction identifier. The returned
// bytes are in digest order; RPCs and explorers conventionally display them in
// reverse order. TxID assumes that tx passes Validate; it does not repeat
// structural validation and does not perform consensus validation.
func (tx *Transaction) TxID() [32]byte {
	h := personal("ZTxIdHeadersHash", le32(tx.Header), le32(tx.VersionGroupID), le32(tx.ConsensusBranchID), le32(tx.LockTime), le32(tx.ExpiryHeight))
	prev, seq, out := []byte{}, []byte{}, []byte{}
	for _, x := range tx.TransparentInputs {
		prev = append(prev, x.PreviousOutput.Hash[:]...)
		prev = append(prev, le32(x.PreviousOutput.Index)...)
		seq = append(seq, le32(x.Sequence)...)
	}
	for _, x := range tx.TransparentOutputs {
		out = append(out, le64(x.Value)...)
		var b bytes.Buffer
		writeCompact(&b, uint64(len(x.ScriptPubKey)))
		out = append(out, b.Bytes()...)
		out = append(out, x.ScriptPubKey...)
	}
	pt := personal("ZTxIdPrevoutHash", prev)
	sq := personal("ZTxIdSequencHash", seq)
	ot := personal("ZTxIdOutputsHash", out)
	td := personal("ZTxIdTranspaHash", pt[:], sq[:], ot[:])
	if len(tx.TransparentInputs)+len(tx.TransparentOutputs) == 0 {
		td = personal("ZTxIdTranspaHash")
	}
	sd := tx.saplingDigest()
	od := tx.orchardDigest()
	var top [16]byte
	copy(top[:], "ZcashTxHash_")
	binary.LittleEndian.PutUint32(top[12:], tx.ConsensusBranchID)
	return blake2bPersonal(top, h[:], td[:], sd[:], od[:])
}

// TxIDString returns the conventional lowercase hexadecimal transaction ID
// used by Zcash RPCs and block explorers. It reverses the digest-order bytes
// returned by TxID before encoding them.
//
// TxIDString has the same precondition as TxID: tx should pass Validate.
func (tx *Transaction) TxIDString() string {
	digest := tx.TxID()
	var display [32]byte
	for i := range digest {
		display[len(display)-1-i] = digest[i]
	}
	return hex.EncodeToString(display[:])
}

// Hash is an alias for TxID and has the same Validate precondition.
func (tx *Transaction) Hash() [32]byte { return tx.TxID() }

// AuthDigest computes the ZIP-244 commitment to transaction authorizing data.
// Together, TxID and AuthDigest commit to every byte of a v5 transaction.
// AuthDigest assumes that tx passes Validate; it does not repeat structural
// validation and does not perform consensus validation.
func (tx *Transaction) AuthDigest() [32]byte {
	transparent := make([]byte, 0)
	for _, in := range tx.TransparentInputs {
		var size bytes.Buffer
		_ = writeCompact(&size, uint64(len(in.ScriptSig)))
		transparent = append(transparent, size.Bytes()...)
		transparent = append(transparent, in.ScriptSig...)
	}
	transparentDigest := personal("ZTxAuthTransHash", transparent)

	saplingDigest := personal("ZTxAuthSapliHash")
	if len(tx.Sapling.Spends)+len(tx.Sapling.Outputs) > 0 {
		parts := make([][]byte, 0, len(tx.Sapling.SpendProofs)+len(tx.Sapling.SpendAuthSigs)+len(tx.Sapling.OutputProofs)+1)
		for i := range tx.Sapling.SpendProofs {
			parts = append(parts, tx.Sapling.SpendProofs[i][:])
		}
		for i := range tx.Sapling.SpendAuthSigs {
			parts = append(parts, tx.Sapling.SpendAuthSigs[i][:])
		}
		for i := range tx.Sapling.OutputProofs {
			parts = append(parts, tx.Sapling.OutputProofs[i][:])
		}
		parts = append(parts, tx.Sapling.BindingSig[:])
		saplingDigest = personal("ZTxAuthSapliHash", parts...)
	}

	orchardDigest := personal("ZTxAuthOrchaHash")
	if len(tx.Orchard.Actions) > 0 {
		parts := make([][]byte, 0, len(tx.Orchard.SpendAuthSigs)+2)
		parts = append(parts, tx.Orchard.Proofs)
		for i := range tx.Orchard.SpendAuthSigs {
			parts = append(parts, tx.Orchard.SpendAuthSigs[i][:])
		}
		parts = append(parts, tx.Orchard.BindingSig[:])
		orchardDigest = personal("ZTxAuthOrchaHash", parts...)
	}

	var top [16]byte
	copy(top[:], "ZTxAuthHash_")
	binary.LittleEndian.PutUint32(top[12:], tx.ConsensusBranchID)
	return blake2bPersonal(top, transparentDigest[:], saplingDigest[:], orchardDigest[:])
}

func (tx *Transaction) saplingDigest() [32]byte {
	if len(tx.Sapling.Spends)+len(tx.Sapling.Outputs) == 0 {
		return personal("ZTxIdSaplingHash")
	}
	c, n := []byte{}, []byte{}
	for _, s := range tx.Sapling.Spends {
		c = append(c, s.Nullifier[:]...)
		n = append(n, s.CV[:]...)
		n = append(n, tx.Sapling.Anchor[:]...)
		n = append(n, s.RK[:]...)
	}
	ch := personal("ZTxIdSSpendCHash", c)
	nh := personal("ZTxIdSSpendNHash", n)
	sp := personal("ZTxIdSSpendsHash", ch[:], nh[:])
	if len(tx.Sapling.Spends) == 0 {
		sp = personal("ZTxIdSSpendsHash")
	}
	oc, om, on := []byte{}, []byte{}, []byte{}
	for _, o := range tx.Sapling.Outputs {
		oc = append(oc, o.CMU[:]...)
		oc = append(oc, o.EphemeralKey[:]...)
		oc = append(oc, o.EncCiphertext[:52]...)
		om = append(om, o.EncCiphertext[52:564]...)
		on = append(on, o.CV[:]...)
		on = append(on, o.EncCiphertext[564:]...)
		on = append(on, o.OutCiphertext[:]...)
	}
	och := personal("ZTxIdSOutC__Hash", oc)
	omh := personal("ZTxIdSOutM__Hash", om)
	onh := personal("ZTxIdSOutN__Hash", on)
	op := personal("ZTxIdSOutputHash", och[:], omh[:], onh[:])
	if len(tx.Sapling.Outputs) == 0 {
		op = personal("ZTxIdSOutputHash")
	}
	return personal("ZTxIdSaplingHash", sp[:], op[:], le64(tx.Sapling.ValueBalance))
}

func (tx *Transaction) orchardDigest() [32]byte {
	if len(tx.Orchard.Actions) == 0 {
		return personal("ZTxIdOrchardHash")
	}
	c, m, n := []byte{}, []byte{}, []byte{}
	for _, a := range tx.Orchard.Actions {
		c = append(c, a.Nullifier[:]...)
		c = append(c, a.CMX[:]...)
		c = append(c, a.EphemeralKey[:]...)
		c = append(c, a.EncCiphertext[:52]...)
		m = append(m, a.EncCiphertext[52:564]...)
		n = append(n, a.CV[:]...)
		n = append(n, a.RK[:]...)
		n = append(n, a.EncCiphertext[564:]...)
		n = append(n, a.OutCiphertext[:]...)
	}
	ch := personal("ZTxIdOrcActCHash", c)
	mh := personal("ZTxIdOrcActMHash", m)
	nh := personal("ZTxIdOrcActNHash", n)
	return personal("ZTxIdOrchardHash", ch[:], mh[:], nh[:], []byte{tx.Orchard.Flags}, le64(tx.Orchard.ValueBalance), tx.Orchard.Anchor[:])
}
