# go-zcashblob

[![CI](https://github.com/ruzcash/go-zcashblob/actions/workflows/ci.yml/badge.svg)](https://github.com/ruzcash/go-zcashblob/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/ruzcash/go-zcashblob.svg)](https://pkg.go.dev/github.com/ruzcash/go-zcashblob)

`go-zcashblob` is a dependency-free Go library for parsing, serializing, and
hashing Zcash version 5 raw transactions.

A **blob** is an opaque sequence of binary bytes. A Zcash transaction blob is
the exact byte representation accepted and returned by node RPC interfaces.
This package turns those bytes into typed transparent, Sapling, and Orchard
fields and can rebuild the original bytes without loss.

## Features

- ZIP-225 version 5 transaction parsing and serialization
- transparent inputs and outputs
- Sapling spends, outputs, proofs, and authorizing data
- Orchard actions, aggregated proofs, and authorizing data
- ZIP-244 non-malleable transaction IDs
- ZIP-244 authorizing-data commitments
- canonical CompactSize encoding
- byte-for-byte parse/serialize round trips
- bounded allocations and complete writer-error propagation
- no external dependencies

## Installation

```sh
go get github.com/ruzcash/go-zcashblob
```

## Example

```go
package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/ruzcash/go-zcashblob"
)

func main() {
	const rawHex = "050000800a27a726b4d0d6c2c2eb518f68984d02010000000000000000000000000000000000000000000000000000000000000000ffffffff060468984d0200ffffffff00000000"

	blob, err := hex.DecodeString(rawHex)
	if err != nil {
		log.Fatal(err)
	}
	tx, err := zcashblob.Parse(blob)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("transparent inputs: %d\n", len(tx.TransparentInputs))
	fmt.Printf("Sapling outputs: %d\n", len(tx.Sapling.Outputs))
	fmt.Printf("Orchard actions: %d\n", len(tx.Orchard.Actions))
	fmt.Printf("ZIP-244 txid digest: %x\n", tx.TxID())
	fmt.Printf("authorization digest: %x\n", tx.AuthDigest())

	rebuilt, err := zcashblob.Serialize(tx)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("round trip preserved %d bytes\n", len(rebuilt))
}
```

`TxID` returns the 32 digest bytes in hash-function order. RPCs and block
explorers conventionally display a transaction ID with those bytes reversed.

## Scope and safety

The parser validates the v5 wire structure. It rejects non-canonical lengths,
reserved Orchard flag bits, oversized input, truncated fields, inconsistent
authorization-vector lengths during serialization, and trailing bytes.

This package is **not a consensus validator**. It does not execute transparent
scripts or validate monetary ranges, expiry rules, curve points, proofs, or
signatures. Only a consensus node can determine whether a transaction is valid
for a particular network and chain state.

Versions 1 through 4 are intentionally rejected because they use different
pool layouts and transaction-ID rules. `MaxTransactionSize`, `MaxScriptSize`,
`MaxProofSize`, and `MaxElements` are defensive library policies.

The Orchard proof length is preserved exactly instead of being forced to the
original ZIP-225 formula. This maintains compatibility across consensus
branches that apply different historical proof-shape rules.

## Verification

The test suite includes official ZIP-244 transaction-ID and authorization
vectors, complete writer fault injection, every-byte truncation checks,
CompactSize boundary cases, adversarial length declarations, fuzzing seeds,
and race-safe round trips.

```sh
go test ./...
go test -race ./...
go vet ./...
go test -fuzz=FuzzParse -fuzztime=30s
```

## Specifications

- [ZIP 225 — Version 5 Transaction Format](https://zips.z.cash/zip-0225)
- [ZIP 244 — Transaction Identifier Non-Malleability](https://zips.z.cash/zip-0244)

Licensed under the MIT License.
