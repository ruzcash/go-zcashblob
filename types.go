package zcashblob

const (
	// OverwinterFlag is bit 31 of a post-Overwinter transaction header. ZIP-225
	// requires this bit to be set for version 5 transactions.
	OverwinterFlag uint32 = 1 << 31
	// Version5 is the transaction version introduced by NU5 and specified by
	// ZIP-225.
	Version5 uint32 = 5
	// VersionGroupIDV5 identifies the ZIP-225 version 5 transaction layout.
	VersionGroupIDV5 uint32 = 0x26A7270A
	// MaxElements is the package safety limit for every attacker-controlled
	// transaction vector.
	MaxElements = (1 << 16) - 1
	// MaxScriptSize is the package safety limit for one transparent script.
	MaxScriptSize = 10 << 20
	// MaxProofSize is the package safety limit for the encoded Orchard proof.
	MaxProofSize = 10 << 20
	// MaxTransactionSize is the package safety limit for input accepted by
	// Parse and ParseFromReader and output accepted by Validate.
	MaxTransactionSize = 16 << 20
)

// OutPoint identifies a transparent output of an earlier transaction.
type OutPoint struct {
	// Hash is the previous transaction identifier in ZIP-225 wire byte order.
	Hash [32]byte
	// Index selects an output of the previous transaction.
	Index uint32
}

// TxIn is a transparent input, encoded as in Bitcoin by ZIP-225.
type TxIn struct {
	// PreviousOutput identifies the transparent output being spent.
	PreviousOutput OutPoint
	// ScriptSig is the unlocking script, without its CompactSize length prefix.
	ScriptSig []byte
	// Sequence is the input's nSequence value.
	Sequence uint32
}

// TxOut is a transparent output, encoded as in Bitcoin by ZIP-225.
type TxOut struct {
	// Value is the output amount in zatoshis. Validate does not check monetary
	// range or consensus rules.
	Value int64
	// ScriptPubKey is the locking script, without its CompactSize length prefix.
	ScriptPubKey []byte
}

// SaplingSpend contains the effecting data of a v5 Sapling spend description.
// Its proof and authorization signature occupy the corresponding indices in
// SaplingBundle.SpendProofs and SaplingBundle.SpendAuthSigs.
type SaplingSpend struct {
	// CV is the value commitment to the input note's value.
	CV [32]byte
	// Nullifier identifies the input note without revealing it.
	Nullifier [32]byte
	// RK is the randomized validating key for the corresponding spend
	// authorization signature.
	RK [32]byte
}

// SaplingOutput contains the effecting data of a v5 Sapling output description.
type SaplingOutput struct {
	// CV is the value commitment to the output note's value.
	CV [32]byte
	// CMU is the u-coordinate of the output note commitment.
	CMU [32]byte
	// EphemeralKey is an encoded ephemeral Jubjub public key.
	EphemeralKey [32]byte
	// EncCiphertext contains the encrypted note plaintext.
	EncCiphertext [580]byte
	// OutCiphertext encrypts the outgoing recovery data.
	OutCiphertext [80]byte
}

// SaplingBundle contains the Sapling effecting and authorizing data of a v5
// transaction. Proof and signature slices have one-to-one, index-preserving
// correspondence with Spends or Outputs; Validate enforces those lengths.
type SaplingBundle struct {
	// Spends contains the effecting portion of each Sapling spend.
	Spends []SaplingSpend
	// Outputs contains the effecting portion of each Sapling output.
	Outputs []SaplingOutput
	// ValueBalance is the net value of Sapling spends minus outputs in
	// zatoshis. It is encoded only when the bundle has a spend or output.
	ValueBalance int64
	// Anchor is a root of the Sapling note commitment tree. It is shared by
	// every spend and encoded only when Spends is non-empty.
	Anchor [32]byte
	// SpendProofs contains one zk-SNARK proof per element of Spends.
	SpendProofs [][192]byte
	// SpendAuthSigs contains one authorization signature per element of Spends.
	SpendAuthSigs [][64]byte
	// OutputProofs contains one zk-SNARK proof per element of Outputs.
	OutputProofs [][192]byte
	// BindingSig is the Sapling binding signature. It is encoded only when the
	// bundle has a spend or output.
	BindingSig [64]byte
}

// OrchardAction contains the effecting data of a v5 Orchard action.
type OrchardAction struct {
	// CV is the value commitment to the input note's value minus the output
	// note's value.
	CV [32]byte
	// Nullifier identifies the input note without revealing it.
	Nullifier [32]byte
	// RK is the randomized validating key for the corresponding spend
	// authorization signature.
	RK [32]byte
	// CMX is the x-coordinate of the output note commitment.
	CMX [32]byte
	// EphemeralKey is an encoded ephemeral Pallas public key.
	EphemeralKey [32]byte
	// EncCiphertext contains the encrypted note plaintext.
	EncCiphertext [580]byte
	// OutCiphertext encrypts the outgoing recovery data.
	OutCiphertext [80]byte
}

// OrchardBundle contains the Orchard effecting and authorizing data of a v5
// transaction. All fields after Actions are encoded only when Actions is
// non-empty.
type OrchardBundle struct {
	// Actions contains Orchard action descriptions.
	Actions []OrchardAction
	// Flags contains enableSpendsOrchard in bit 0 and enableOutputsOrchard in
	// bit 1. All other bits are reserved and rejected by Parse and Validate.
	Flags byte
	// ValueBalance is the net value of Orchard spends minus outputs in
	// zatoshis.
	ValueBalance int64
	// Anchor is a root of the Orchard note commitment tree.
	Anchor [32]byte
	// Proofs is the encoded aggregated zk-SNARK proof for Actions, without its
	// CompactSize length prefix. Its exact length is preserved.
	Proofs []byte
	// SpendAuthSigs contains one authorization signature per element of Actions.
	SpendAuthSigs [][64]byte
	// BindingSig is the Orchard binding signature.
	BindingSig [64]byte
}

// Transaction is a ZIP-225 version 5 transaction. The zero value is not a
// valid v5 header; use NewTransactionV5 when constructing a transaction.
type Transaction struct {
	// Header contains the Overwinter flag in bit 31 and the transaction version
	// in bits 30 through 0.
	Header uint32
	// VersionGroupID identifies the transaction layout. Version 5 uses
	// VersionGroupIDV5.
	VersionGroupID uint32
	// ConsensusBranchID identifies the target consensus branch and domain
	// separates ZIP-244 top-level digests.
	ConsensusBranchID uint32
	// LockTime is a Unix time or block height encoded as in Bitcoin.
	LockTime uint32
	// ExpiryHeight is the height after which the transaction expires. ZIP-225
	// permits 1 through 499999999, or zero to disable expiry; Validate does not
	// enforce this consensus range.
	ExpiryHeight uint32
	// TransparentInputs contains the transparent inputs in wire order.
	TransparentInputs []TxIn
	// TransparentOutputs contains the transparent outputs in wire order.
	TransparentOutputs []TxOut
	// Sapling contains the Sapling bundle.
	Sapling SaplingBundle
	// Orchard contains the Orchard bundle.
	Orchard OrchardBundle
}

// NewTransactionV5 returns an empty structurally valid ZIP-225 transaction for
// consensusBranchID. LockTime and ExpiryHeight are zero and all pool bundles
// are empty. The caller is responsible for choosing a branch ID suitable for
// the target network and epoch.
func NewTransactionV5(consensusBranchID uint32) *Transaction {
	return &Transaction{
		Header:            OverwinterFlag | Version5,
		VersionGroupID:    VersionGroupIDV5,
		ConsensusBranchID: consensusBranchID,
	}
}

// Version returns the transaction version with the Overwinter flag removed.
func (tx *Transaction) Version() uint32 { return tx.Header &^ OverwinterFlag }
