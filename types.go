// Package zcashblob parses, serializes, and hashes Zcash v5 transaction blobs.
package zcashblob

const (
	// OverwinterFlag marks post-Overwinter transaction headers.
	OverwinterFlag uint32 = 1 << 31
	// Version5 is the transaction version introduced by NU5.
	Version5 uint32 = 5
	// VersionGroupIDV5 identifies the ZIP-225 transaction layout.
	VersionGroupIDV5 uint32 = 0x26A7270A
	// MaxElements bounds every attacker-controlled transaction vector.
	MaxElements = (1 << 16) - 1
	// MaxScriptSize bounds an individual transparent script.
	MaxScriptSize = 10 << 20
	// MaxProofSize bounds the encoded Orchard proof.
	MaxProofSize = 10 << 20
	// MaxTransactionSize bounds input accepted by Parse and ParseFromReader.
	MaxTransactionSize = 16 << 20
)

// OutPoint identifies a transparent output of an earlier transaction. Hash is
// stored in the same byte order used by the wire encoding.
type OutPoint struct {
	Hash  [32]byte
	Index uint32
}

// TxIn is a transparent transaction input.
type TxIn struct {
	PreviousOutput OutPoint
	ScriptSig      []byte
	Sequence       uint32
}

// TxOut is a transparent transaction output. Value is measured in zatoshis.
type TxOut struct {
	Value        int64
	ScriptPubKey []byte
}

// SaplingSpend contains the effecting data of a v5 Sapling spend description.
type SaplingSpend struct{ CV, Nullifier, RK [32]byte }

// SaplingOutput contains the effecting data of a v5 Sapling output description.
type SaplingOutput struct {
	CV, CMU, EphemeralKey [32]byte
	EncCiphertext         [580]byte
	OutCiphertext         [80]byte
}

// SaplingBundle contains all Sapling effecting and authorizing data.
type SaplingBundle struct {
	Spends        []SaplingSpend
	Outputs       []SaplingOutput
	ValueBalance  int64
	Anchor        [32]byte
	SpendProofs   [][192]byte
	SpendAuthSigs [][64]byte
	OutputProofs  [][192]byte
	BindingSig    [64]byte
}

// OrchardAction contains the effecting data of a v5 Orchard action.
type OrchardAction struct {
	CV, Nullifier, RK, CMX, EphemeralKey [32]byte
	EncCiphertext                        [580]byte
	OutCiphertext                        [80]byte
}

// OrchardBundle contains all Orchard effecting and authorizing data.
type OrchardBundle struct {
	Actions       []OrchardAction
	Flags         byte
	ValueBalance  int64
	Anchor        [32]byte
	Proofs        []byte
	SpendAuthSigs [][64]byte
	BindingSig    [64]byte
}

// Transaction is a parsed ZIP-225 version 5 transaction.
type Transaction struct {
	Header, VersionGroupID, ConsensusBranchID, LockTime, ExpiryHeight uint32
	TransparentInputs                                                 []TxIn
	TransparentOutputs                                                []TxOut
	Sapling                                                           SaplingBundle
	Orchard                                                           OrchardBundle
}

// Version returns the transaction version with the Overwinter flag removed.
func (tx *Transaction) Version() uint32 { return tx.Header &^ OverwinterFlag }
