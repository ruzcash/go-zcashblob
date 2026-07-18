package zcashblob

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

type decoder struct{ r *bytes.Reader }

func (d *decoder) fixed(p []byte) error { _, err := io.ReadFull(d.r, p); return err }
func (d *decoder) u32(p *uint32) error  { return binary.Read(d.r, binary.LittleEndian, p) }
func (d *decoder) i64(p *int64) error   { return binary.Read(d.r, binary.LittleEndian, p) }
func (d *decoder) count(max uint64) (int, error) {
	v, e := readCompact(d.r)
	if e != nil {
		return 0, e
	}
	if v > max {
		return 0, ErrTooLarge
	}
	return int(v), nil
}
func (d *decoder) countFixed(max, minimumBytes uint64) (int, error) {
	n, err := d.count(max)
	if err != nil {
		return 0, err
	}
	if uint64(n) > uint64(d.r.Len())/minimumBytes {
		return 0, io.ErrUnexpectedEOF
	}
	return n, nil
}
func (d *decoder) varBytes(max uint64) ([]byte, error) {
	n, e := d.count(max)
	if e != nil {
		return nil, e
	}
	if n > d.r.Len() {
		return nil, io.ErrUnexpectedEOF
	}
	p := make([]byte, n)
	return p, d.fixed(p)
}

// Parse decodes exactly one ZIP-225 version 5 transaction. It rejects trailing
// bytes, non-canonical CompactSize values, and inputs above the safety limits.
func Parse(data []byte) (*Transaction, error) {
	if len(data) > MaxTransactionSize {
		return nil, ErrTooLarge
	}
	d := decoder{bytes.NewReader(data)}
	tx := new(Transaction)
	for _, p := range []*uint32{&tx.Header, &tx.VersionGroupID, &tx.ConsensusBranchID, &tx.LockTime, &tx.ExpiryHeight} {
		if e := d.u32(p); e != nil {
			return nil, e
		}
	}
	if tx.Header&OverwinterFlag == 0 || tx.Version() != Version5 || tx.VersionGroupID != VersionGroupIDV5 {
		return nil, ErrUnsupportedVersion
	}
	ni, e := d.countFixed(MaxElements, 41)
	if e != nil {
		return nil, e
	}
	tx.TransparentInputs = make([]TxIn, ni)
	for i := range tx.TransparentInputs {
		x := &tx.TransparentInputs[i]
		if e = d.fixed(x.PreviousOutput.Hash[:]); e != nil {
			return nil, e
		}
		if e = d.u32(&x.PreviousOutput.Index); e != nil {
			return nil, e
		}
		if x.ScriptSig, e = d.varBytes(MaxScriptSize); e != nil {
			return nil, e
		}
		if e = d.u32(&x.Sequence); e != nil {
			return nil, e
		}
	}
	no, e := d.countFixed(MaxElements, 9)
	if e != nil {
		return nil, e
	}
	tx.TransparentOutputs = make([]TxOut, no)
	for i := range tx.TransparentOutputs {
		x := &tx.TransparentOutputs[i]
		if e = d.i64(&x.Value); e != nil {
			return nil, e
		}
		if x.ScriptPubKey, e = d.varBytes(MaxScriptSize); e != nil {
			return nil, e
		}
	}
	ns, e := d.countFixed(MaxElements, 96)
	if e != nil {
		return nil, e
	}
	tx.Sapling.Spends = make([]SaplingSpend, ns)
	for i := range tx.Sapling.Spends {
		s := &tx.Sapling.Spends[i]
		for _, p := range [][]byte{s.CV[:], s.Nullifier[:], s.RK[:]} {
			if e = d.fixed(p); e != nil {
				return nil, e
			}
		}
	}
	nso, e := d.countFixed(MaxElements, 756)
	if e != nil {
		return nil, e
	}
	tx.Sapling.Outputs = make([]SaplingOutput, nso)
	for i := range tx.Sapling.Outputs {
		o := &tx.Sapling.Outputs[i]
		for _, p := range [][]byte{o.CV[:], o.CMU[:], o.EphemeralKey[:], o.EncCiphertext[:], o.OutCiphertext[:]} {
			if e = d.fixed(p); e != nil {
				return nil, e
			}
		}
	}
	if ns+nso > 0 {
		if e = d.i64(&tx.Sapling.ValueBalance); e != nil {
			return nil, e
		}
	}
	if ns > 0 {
		if e = d.fixed(tx.Sapling.Anchor[:]); e != nil {
			return nil, e
		}
	}
	tx.Sapling.SpendProofs = make([][192]byte, ns)
	for i := range tx.Sapling.SpendProofs {
		if e = d.fixed(tx.Sapling.SpendProofs[i][:]); e != nil {
			return nil, e
		}
	}
	tx.Sapling.SpendAuthSigs = make([][64]byte, ns)
	for i := range tx.Sapling.SpendAuthSigs {
		if e = d.fixed(tx.Sapling.SpendAuthSigs[i][:]); e != nil {
			return nil, e
		}
	}
	tx.Sapling.OutputProofs = make([][192]byte, nso)
	for i := range tx.Sapling.OutputProofs {
		if e = d.fixed(tx.Sapling.OutputProofs[i][:]); e != nil {
			return nil, e
		}
	}
	if ns+nso > 0 {
		if e = d.fixed(tx.Sapling.BindingSig[:]); e != nil {
			return nil, e
		}
	}
	na, e := d.countFixed(MaxElements, 820)
	if e != nil {
		return nil, e
	}
	tx.Orchard.Actions = make([]OrchardAction, na)
	for i := range tx.Orchard.Actions {
		a := &tx.Orchard.Actions[i]
		for _, p := range [][]byte{a.CV[:], a.Nullifier[:], a.RK[:], a.CMX[:], a.EphemeralKey[:], a.EncCiphertext[:], a.OutCiphertext[:]} {
			if e = d.fixed(p); e != nil {
				return nil, e
			}
		}
	}
	if na > 0 {
		var f [1]byte
		if e = d.fixed(f[:]); e != nil {
			return nil, e
		}
		tx.Orchard.Flags = f[0]
		if tx.Orchard.Flags&^byte(0x03) != 0 {
			return nil, fmt.Errorf("%w: reserved Orchard flag bits are set", ErrInvalidStructure)
		}
		if e = d.i64(&tx.Orchard.ValueBalance); e != nil {
			return nil, e
		}
		if e = d.fixed(tx.Orchard.Anchor[:]); e != nil {
			return nil, e
		}
		if tx.Orchard.Proofs, e = d.varBytes(MaxProofSize); e != nil {
			return nil, e
		}
		tx.Orchard.SpendAuthSigs = make([][64]byte, na)
		for i := range tx.Orchard.SpendAuthSigs {
			if e = d.fixed(tx.Orchard.SpendAuthSigs[i][:]); e != nil {
				return nil, e
			}
		}
		if e = d.fixed(tx.Orchard.BindingSig[:]); e != nil {
			return nil, e
		}
	}
	if d.r.Len() != 0 {
		return nil, ErrTrailingData
	}
	return tx, nil
}

// ParseFromReader reads one bounded transaction from r and passes it to Parse.
// It buffers the transaction so it can guarantee that no trailing bytes exist.
func ParseFromReader(r io.Reader) (*Transaction, error) {
	if r == nil {
		return nil, ErrNilReader
	}
	data, e := io.ReadAll(io.LimitReader(r, MaxTransactionSize+1))
	if e != nil {
		return nil, e
	}
	if len(data) > MaxTransactionSize {
		return nil, ErrTooLarge
	}
	return Parse(data)
}

// Serialize encodes tx in the ZIP-225 version 5 wire format. It returns the
// same validation errors as Transaction.Validate.
func Serialize(tx *Transaction) ([]byte, error) {
	var b bytes.Buffer
	if e := SerializeToWriter(tx, &b); e != nil {
		return nil, e
	}
	return b.Bytes(), nil
}

type encoder struct {
	w   io.Writer
	err error
}

func (e *encoder) bytes(p []byte) {
	if e.err == nil {
		e.err = writeAll(e.w, p)
	}
}

func (e *encoder) u32(v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	e.bytes(b[:])
}

func (e *encoder) i64(v int64) {
	var b [8]byte
	binary.LittleEndian.PutUint64(b[:], uint64(v))
	e.bytes(b[:])
}

func (e *encoder) compact(v uint64) {
	if e.err == nil {
		e.err = writeCompact(e.w, v)
	}
}

func invalidStructure(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalidStructure, fmt.Sprintf(format, args...))
}

func tooLarge(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrTooLarge, fmt.Sprintf(format, args...))
}

// Validate reports whether tx has a supported v5 header, internally consistent
// authorization-vector lengths, conditional-field presence, allowed Orchard
// flags, and an encoded size within the package safety limits.
//
// Validate only checks the structure needed for safe, unambiguous ZIP-225
// serialization. It does not validate Zcash consensus rules, monetary ranges,
// scripts, expiry, encoded curve points, note commitments, proofs, or
// signatures. A nil receiver is reported as ErrInvalidStructure.
func (tx *Transaction) Validate() error {
	if tx == nil {
		return invalidStructure("nil transaction")
	}
	if tx.Header&OverwinterFlag == 0 || tx.Version() != Version5 || tx.VersionGroupID != VersionGroupIDV5 {
		return ErrUnsupportedVersion
	}
	counts := []struct {
		name string
		n    int
	}{
		{"transparent inputs", len(tx.TransparentInputs)},
		{"transparent outputs", len(tx.TransparentOutputs)},
		{"Sapling spends", len(tx.Sapling.Spends)},
		{"Sapling outputs", len(tx.Sapling.Outputs)},
		{"Orchard actions", len(tx.Orchard.Actions)},
	}
	for _, count := range counts {
		if count.n > MaxElements {
			return tooLarge("too many %s", count.name)
		}
	}
	for _, in := range tx.TransparentInputs {
		if len(in.ScriptSig) > MaxScriptSize {
			return tooLarge("transparent scriptSig exceeds limit")
		}
	}
	for _, out := range tx.TransparentOutputs {
		if len(out.ScriptPubKey) > MaxScriptSize {
			return tooLarge("transparent scriptPubKey exceeds limit")
		}
	}
	if len(tx.Sapling.SpendProofs) != len(tx.Sapling.Spends) ||
		len(tx.Sapling.SpendAuthSigs) != len(tx.Sapling.Spends) ||
		len(tx.Sapling.OutputProofs) != len(tx.Sapling.Outputs) {
		return invalidStructure("Sapling authorization count mismatch")
	}
	if len(tx.Orchard.SpendAuthSigs) != len(tx.Orchard.Actions) {
		return invalidStructure("Orchard authorization count mismatch")
	}
	if len(tx.Orchard.Proofs) > MaxProofSize {
		return tooLarge("Orchard proof exceeds limit")
	}
	if tx.Orchard.Flags&^byte(0x03) != 0 {
		return invalidStructure("reserved Orchard flag bits are set")
	}
	var zero32 [32]byte
	var zero64 [64]byte
	if len(tx.Sapling.Spends)+len(tx.Sapling.Outputs) == 0 {
		if tx.Sapling.ValueBalance != 0 {
			return invalidStructure("Sapling value balance without spends or outputs")
		}
		if tx.Sapling.BindingSig != zero64 {
			return invalidStructure("Sapling binding signature without spends or outputs")
		}
	}
	if len(tx.Sapling.Spends) == 0 && tx.Sapling.Anchor != zero32 {
		return invalidStructure("Sapling anchor without spends")
	}
	if len(tx.Orchard.Actions) == 0 {
		if tx.Orchard.Flags != 0 {
			return invalidStructure("Orchard flags without actions")
		}
		if tx.Orchard.ValueBalance != 0 {
			return invalidStructure("Orchard value balance without actions")
		}
		if tx.Orchard.Anchor != zero32 {
			return invalidStructure("Orchard anchor without actions")
		}
		if len(tx.Orchard.Proofs) != 0 {
			return invalidStructure("Orchard proof without actions")
		}
		if tx.Orchard.BindingSig != zero64 {
			return invalidStructure("Orchard binding signature without actions")
		}
	}
	if encodedSize(tx) > MaxTransactionSize {
		return tooLarge("encoded transaction exceeds limit")
	}
	return nil
}

func encodedSize(tx *Transaction) uint64 {
	size := uint64(20)
	size += uint64(compactSizeLen(uint64(len(tx.TransparentInputs))))
	for _, in := range tx.TransparentInputs {
		n := uint64(len(in.ScriptSig))
		size += 32 + 4 + uint64(compactSizeLen(n)) + n + 4
	}
	size += uint64(compactSizeLen(uint64(len(tx.TransparentOutputs))))
	for _, out := range tx.TransparentOutputs {
		n := uint64(len(out.ScriptPubKey))
		size += 8 + uint64(compactSizeLen(n)) + n
	}
	size += uint64(compactSizeLen(uint64(len(tx.Sapling.Spends)))) + uint64(len(tx.Sapling.Spends))*96
	size += uint64(compactSizeLen(uint64(len(tx.Sapling.Outputs)))) + uint64(len(tx.Sapling.Outputs))*756
	if len(tx.Sapling.Spends)+len(tx.Sapling.Outputs) > 0 {
		size += 8 + 64
	}
	if len(tx.Sapling.Spends) > 0 {
		size += 32
	}
	size += uint64(len(tx.Sapling.Spends)) * (192 + 64)
	size += uint64(len(tx.Sapling.Outputs)) * 192
	size += uint64(compactSizeLen(uint64(len(tx.Orchard.Actions)))) + uint64(len(tx.Orchard.Actions))*820
	if len(tx.Orchard.Actions) > 0 {
		proofLen := uint64(len(tx.Orchard.Proofs))
		size += 1 + 8 + 32 + uint64(compactSizeLen(proofLen)) + proofLen
		size += uint64(len(tx.Orchard.Actions))*64 + 64
	}
	return size
}

// SerializeToWriter writes tx in the ZIP-225 version 5 wire format. It calls
// Transaction.Validate before writing the first byte.
func SerializeToWriter(tx *Transaction, w io.Writer) error {
	if w == nil {
		return ErrNilWriter
	}
	if err := tx.Validate(); err != nil {
		return err
	}
	e := encoder{w: w}
	for _, v := range []uint32{tx.Header, tx.VersionGroupID, tx.ConsensusBranchID, tx.LockTime, tx.ExpiryHeight} {
		e.u32(v)
	}
	e.compact(uint64(len(tx.TransparentInputs)))
	for _, x := range tx.TransparentInputs {
		e.bytes(x.PreviousOutput.Hash[:])
		e.u32(x.PreviousOutput.Index)
		e.compact(uint64(len(x.ScriptSig)))
		e.bytes(x.ScriptSig)
		e.u32(x.Sequence)
	}
	e.compact(uint64(len(tx.TransparentOutputs)))
	for _, x := range tx.TransparentOutputs {
		e.i64(x.Value)
		e.compact(uint64(len(x.ScriptPubKey)))
		e.bytes(x.ScriptPubKey)
	}
	e.compact(uint64(len(tx.Sapling.Spends)))
	for _, s := range tx.Sapling.Spends {
		e.bytes(s.CV[:])
		e.bytes(s.Nullifier[:])
		e.bytes(s.RK[:])
	}
	e.compact(uint64(len(tx.Sapling.Outputs)))
	for _, o := range tx.Sapling.Outputs {
		e.bytes(o.CV[:])
		e.bytes(o.CMU[:])
		e.bytes(o.EphemeralKey[:])
		e.bytes(o.EncCiphertext[:])
		e.bytes(o.OutCiphertext[:])
	}
	if len(tx.Sapling.Spends)+len(tx.Sapling.Outputs) > 0 {
		e.i64(tx.Sapling.ValueBalance)
	}
	if len(tx.Sapling.Spends) > 0 {
		e.bytes(tx.Sapling.Anchor[:])
	}
	for _, p := range tx.Sapling.SpendProofs {
		e.bytes(p[:])
	}
	for _, s := range tx.Sapling.SpendAuthSigs {
		e.bytes(s[:])
	}
	for _, p := range tx.Sapling.OutputProofs {
		e.bytes(p[:])
	}
	if len(tx.Sapling.Spends)+len(tx.Sapling.Outputs) > 0 {
		e.bytes(tx.Sapling.BindingSig[:])
	}
	e.compact(uint64(len(tx.Orchard.Actions)))
	for _, a := range tx.Orchard.Actions {
		e.bytes(a.CV[:])
		e.bytes(a.Nullifier[:])
		e.bytes(a.RK[:])
		e.bytes(a.CMX[:])
		e.bytes(a.EphemeralKey[:])
		e.bytes(a.EncCiphertext[:])
		e.bytes(a.OutCiphertext[:])
	}
	if len(tx.Orchard.Actions) > 0 {
		e.bytes([]byte{tx.Orchard.Flags})
		e.i64(tx.Orchard.ValueBalance)
		e.bytes(tx.Orchard.Anchor[:])
		e.compact(uint64(len(tx.Orchard.Proofs)))
		e.bytes(tx.Orchard.Proofs)
		for _, s := range tx.Orchard.SpendAuthSigs {
			e.bytes(s[:])
		}
		e.bytes(tx.Orchard.BindingSig[:])
	}
	return e.err
}
