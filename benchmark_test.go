package zcashblob

import "testing"

var (
	benchmarkBytes  []byte
	benchmarkDigest [32]byte
	benchmarkTx     *Transaction
)

func benchmarkTransaction(b *testing.B) (*Transaction, []byte) {
	b.Helper()
	tx := populatedTx()
	blob, err := Serialize(tx)
	if err != nil {
		b.Fatal(err)
	}
	return tx, blob
}

func BenchmarkParse(b *testing.B) {
	_, blob := benchmarkTransaction(b)
	b.ReportAllocs()
	b.SetBytes(int64(len(blob)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var err error
		benchmarkTx, err = Parse(blob)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSerialize(b *testing.B) {
	tx, blob := benchmarkTransaction(b)
	b.ReportAllocs()
	b.SetBytes(int64(len(blob)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var err error
		benchmarkBytes, err = Serialize(tx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkTxID(b *testing.B) {
	tx, blob := benchmarkTransaction(b)
	b.ReportAllocs()
	b.SetBytes(int64(len(blob)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkDigest = tx.TxID()
	}
}

func BenchmarkAuthDigest(b *testing.B) {
	tx, blob := benchmarkTransaction(b)
	b.ReportAllocs()
	b.SetBytes(int64(len(blob)))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		benchmarkDigest = tx.AuthDigest()
	}
}
