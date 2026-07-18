package zcashblob

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

const (
	zip244CorpusSHA256 = "3d20892f19cec18afba2ef2907bb8426192d0ea4eac2c0060c2f31ff78ad93a3"
	zip244VectorCount  = 10
	zip244FieldCount   = 13
	zip244Schema       = "tx, txid, auth_digest, amounts, script_pubkeys, transparent_input, sighash_shielded, sighash_all, sighash_none, sighash_single, sighash_all_anyone, sighash_none_anyone, sighash_single_anyone"
)

//go:embed testdata/zip_0244.json
var zip244Corpus []byte

func TestZIP244PinnedOfficialCorpus(t *testing.T) {
	digest := sha256.Sum256(zip244Corpus)
	if got := hex.EncodeToString(digest[:]); got != zip244CorpusSHA256 {
		t.Fatalf("corpus SHA-256: got %s want %s", got, zip244CorpusSHA256)
	}

	var rows []json.RawMessage
	if err := json.Unmarshal(zip244Corpus, &rows); err != nil {
		t.Fatalf("decode corpus: %v", err)
	}
	if got, want := len(rows), zip244VectorCount+2; got != want {
		t.Fatalf("corpus rows: got %d want %d", got, want)
	}

	var schemaRow []string
	if err := json.Unmarshal(rows[1], &schemaRow); err != nil {
		t.Fatalf("decode schema row: %v", err)
	}
	if len(schemaRow) != 1 || schemaRow[0] != zip244Schema {
		t.Fatalf("unexpected corpus schema: %q", schemaRow)
	}
	if got := len(strings.Split(schemaRow[0], ", ")); got != zip244FieldCount {
		t.Fatalf("schema field count: got %d want %d", got, zip244FieldCount)
	}

	for i, rawRow := range rows[2:] {
		t.Run(fmt.Sprintf("vector_%02d", i), func(t *testing.T) {
			var fields []json.RawMessage
			if err := json.Unmarshal(rawRow, &fields); err != nil {
				t.Fatalf("decode vector: %v", err)
			}
			if got := len(fields); got != zip244FieldCount {
				t.Fatalf("vector field count: got %d want %d", got, zip244FieldCount)
			}

			blob := decodeZIP244HexField(t, fields[0], "tx")
			wantTxID := decodeZIP244DigestField(t, fields[1], "txid")
			wantAuth := decodeZIP244DigestField(t, fields[2], "auth_digest")
			var displayTxID [32]byte
			for i := range wantTxID {
				displayTxID[len(displayTxID)-1-i] = wantTxID[i]
			}
			wantTxIDString := hex.EncodeToString(displayTxID[:])

			parsed, err := Parse(blob)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			streamed, err := ParseFromReader(bytes.NewReader(blob))
			if err != nil {
				t.Fatalf("ParseFromReader: %v", err)
			}

			parsers := []struct {
				name string
				tx   *Transaction
			}{
				{name: "slice", tx: parsed},
				{name: "reader", tx: streamed},
			}
			for _, parser := range parsers {
				t.Run(parser.name, func(t *testing.T) {
					tx := parser.tx
					if got := tx.TxID(); got != wantTxID {
						t.Fatalf("TxID: got %x want %x", got, wantTxID)
					}
					if got := tx.Hash(); got != wantTxID {
						t.Fatalf("Hash: got %x want %x", got, wantTxID)
					}
					if got := tx.TxIDString(); got != wantTxIDString {
						t.Fatalf("TxIDString: got %s want %s", got, wantTxIDString)
					}
					if got := tx.AuthDigest(); got != wantAuth {
						t.Fatalf("AuthDigest: got %x want %x", got, wantAuth)
					}
					rebuilt, err := Serialize(tx)
					if err != nil {
						t.Fatalf("Serialize: %v", err)
					}
					if !bytes.Equal(rebuilt, blob) {
						t.Fatal("Serialize did not reproduce the exact transaction bytes")
					}
				})
			}
		})
	}
}

func decodeZIP244HexField(t testing.TB, raw json.RawMessage, name string) []byte {
	t.Helper()
	var encoded string
	if err := json.Unmarshal(raw, &encoded); err != nil {
		t.Fatalf("decode %s field: %v", name, err)
	}
	decoded, err := hex.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode %s hex: %v", name, err)
	}
	return decoded
}

func decodeZIP244DigestField(t testing.TB, raw json.RawMessage, name string) [32]byte {
	t.Helper()
	decoded := decodeZIP244HexField(t, raw, name)
	if len(decoded) != 32 {
		t.Fatalf("%s length: got %d want 32", name, len(decoded))
	}
	var digest [32]byte
	copy(digest[:], decoded)
	return digest
}
