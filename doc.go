// Package zcashblob parses, serializes, and hashes Zcash ZIP-225 version 5
// transaction blobs.
//
// Parse and ParseFromReader accept one complete wire encoding and reject
// trailing bytes, non-canonical CompactSize integers, unsupported transaction
// versions, and inputs outside the package safety limits. Serialize and
// SerializeToWriter perform the inverse operation after structural validation.
//
// NewTransactionV5 initializes the mandatory v5 header fields for callers that
// construct a transaction. Validate checks that a Transaction can be encoded
// without discarding fields and under this package's structural and resource
// limits. It is not a Zcash consensus validator: scripts, amounts, expiry
// rules, keys, commitments, proofs, and signatures are not cryptographically
// validated.
//
// TxID and AuthDigest implement the personalized BLAKE2b-256 digest trees from
// ZIP-244. A Transaction should pass Validate before either digest is computed.
package zcashblob
