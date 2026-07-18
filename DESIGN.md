# Design

`go-zcashblob` has one job: represent the complete ZIP-225 version 5 wire
format without hiding bytes or pretending to perform Zcash consensus
validation. The package stays flat because it exposes one cohesive Go API and
has no internal subpackages or commands.

## Layers

The implementation has five narrow layers:

1. `types.go`, `errors.go`, and `doc.go` define the public model, constructor,
   limits, validation contract, and stable error categories.
2. `codec.go` parses, structurally validates, sizes, and serializes the ZIP-225
   layout. Parsing is bounded before allocation; serialization validates the
   complete transaction before writing any bytes.
3. `compactsize.go` owns the canonical variable-length integer encoding and
   complete-write loop used by the codec and digest field encodings.
4. `hash.go` builds the ZIP-244 digest trees. Effecting data contributes to
   `TxID`; authorizing data contributes to `AuthDigest`.
5. `blake2b.go` contains the package-local BLAKE2b-256 primitive used by those
   digest trees.

The public `Validate` method deliberately stops at the encoding boundary. It
checks the v5 header, resource limits, conditional-field presence, reserved
Orchard flags, and the one-to-one relationships between descriptions and
authorization fields. It does not execute scripts, constrain values or expiry
against chain state, or verify points, commitments, proofs, or signatures.
Those are consensus-node responsibilities.

## Why BLAKE2b is local

ZIP-244 requires BLAKE2b-256 personalization as part of the BLAKE2 parameter
block. It is not equivalent to prefixing a personalization string to the
message. Keeping the small primitive local preserves the module's
zero-dependency property and makes that parameter-block behavior explicit at
the call site.

This cryptographic code is intentionally not a general-purpose public API. Its
only consumer is the fixed ZIP-244 digest construction. Confidence comes from
official full-transaction vectors, an independent empty-message personalized
hash oracle, block-boundary tests, and differential digest checks described
below.

## In-memory invariants

- `Header` must contain `OverwinterFlag | Version5`, and `VersionGroupID` must
  equal `VersionGroupIDV5`.
- Sapling spend proofs and spend authorization signatures correspond by index
  to `Sapling.Spends`; output proofs correspond by index to
  `Sapling.Outputs`.
- Orchard spend authorization signatures correspond by index to
  `Orchard.Actions`; only flag bits 0 and 1 are defined.
- Conditional bundle fields follow ZIP-225 presence rules during encoding.
- Parsed byte slices own their storage. Serialization preserves their order and
  exact contents.
- `TxID`, `TxIDString`, `Hash`, and `AuthDigest` require a transaction that
  passes `Validate`. They avoid a second full validation pass so callers that
  hash repeatedly can validate once at the boundary.

`NewTransactionV5` establishes the mandatory header fields. It does not select
a consensus branch for the caller, because that choice depends on the target
network and epoch.

## Test oracles

The tests use several independent oracles instead of relying only on
parse-then-serialize symmetry:

- the complete pinned ZIP-244 corpus verifies parsing through both entry
  points, byte-identical serialization, transaction IDs, and authorizing-data
  digests;
- a non-zero transaction covering every field has a fixed wire SHA-256 oracle
  and a semantic equality check;
- a field mutation matrix verifies whether each field commits through `TxID`,
  `AuthDigest`, or both;
- every-byte mutations, truncation cases, hostile declarations, writer fault
  injection, and fuzz targets exercise rejection and failure paths;
- personalized BLAKE2b and compression block boundaries are tested separately
  from the transaction codec.

Fixture provenance and licensing are recorded in `testdata/README.md`.

## Extension rules

Support for another transaction version should be added as an explicit format,
not as optional branches that weaken v5 invariants. New public fields must state
their wire presence condition, correspondence rules, and whether they affect
the transaction-ID or authorizing-data digest. Any codec change should add an
independent wire or digest oracle before relying on round-trip tests.

The normative format and digest definitions are
[ZIP-225](https://zips.z.cash/zip-0225) and
[ZIP-244](https://zips.z.cash/zip-0244).
