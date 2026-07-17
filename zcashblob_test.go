package zcashblob

import (
	"bytes"
	"encoding/hex"
	"errors"
	"testing"
)

func emptyTx() *Transaction {
	return &Transaction{Header: OverwinterFlag | Version5, VersionGroupID: VersionGroupIDV5, ConsensusBranchID: 0xc2d6d0b4}
}

func TestRoundTripEmptyV5(t *testing.T) {
	want := emptyTx()
	blob, e := Serialize(want)
	if e != nil {
		t.Fatal(e)
	}
	got, e := Parse(blob)
	if e != nil {
		t.Fatal(e)
	}
	again, e := Serialize(got)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(blob, again) {
		t.Fatal("round-trip changed bytes")
	}
}

func TestRoundTripAllPools(t *testing.T) {
	tx := emptyTx()
	tx.TransparentInputs = []TxIn{{ScriptSig: []byte{1, 2, 3}, Sequence: 42}}
	tx.TransparentOutputs = []TxOut{{Value: 123, ScriptPubKey: []byte{0x51}}}
	tx.Sapling.Spends = make([]SaplingSpend, 1)
	tx.Sapling.SpendProofs = make([][192]byte, 1)
	tx.Sapling.SpendAuthSigs = make([][64]byte, 1)
	tx.Sapling.Outputs = make([]SaplingOutput, 1)
	tx.Sapling.OutputProofs = make([][192]byte, 1)
	tx.Orchard.Actions = make([]OrchardAction, 1)
	tx.Orchard.Proofs = []byte{4, 5}
	tx.Orchard.SpendAuthSigs = make([][64]byte, 1)
	blob, e := Serialize(tx)
	if e != nil {
		t.Fatal(e)
	}
	parsed, e := Parse(blob)
	if e != nil {
		t.Fatal(e)
	}
	again, e := Serialize(parsed)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Equal(blob, again) {
		t.Fatal("round-trip changed bytes")
	}
}

func TestEffectingAndAuthorizingDigestsAreSeparated(t *testing.T) {
	tx := populatedTx()
	txid := tx.TxID()
	auth := tx.AuthDigest()

	tx.Sapling.BindingSig[0] ^= 1
	if got := tx.TxID(); got != txid {
		t.Fatal("authorizing data changed the non-malleable txid")
	}
	if got := tx.AuthDigest(); got == auth {
		t.Fatal("authorizing-data mutation did not change AuthDigest")
	}

	tx.TransparentOutputs[0].Value++
	if got := tx.TxID(); got == txid {
		t.Fatal("effecting-data mutation did not change TxID")
	}
}

func TestPersonalizedBlake2b(t *testing.T) {
	got := personal("ZTxIdHeadersHash")
	want, _ := hex.DecodeString("e6f9967555b66ebd3bd806f976a6d2b559dbd87a587e0ab738d1c4d90332e695")
	if !bytes.Equal(got[:], want) {
		t.Fatalf("got %x", got)
	}
}

// Published ZIP-244 conformance vectors cover transparent and shielded
// transaction components.
func TestZIP244OfficialVectors(t *testing.T) {
	vectors := []struct {
		blob string
		txid string
		auth string
	}{
		{"050000800a27a726b4d0d6c2c2eb518f68984d02010000000000000000000000000000000000000000000000000000000000000000ffffffff060468984d0200ffffffff00000000", "28d16c3cd78a6b7a50b11a1b8417764c4e63c6e3d8aa289f7e87e39845872764", "332155b1cc23a2571e86e49e06010cd25321dcfcca34ae14e8b3f4f00270d287"},
		{"050000800a27a726b4d0d6c25e3dbaf7ae12670d010000000000000000000000000000000000000000000000000000000000000000ffffffff0604ae12670d00ffffffff01516cf4adec75070003656500000000", "6bf4efe77af69b7219475f60a0f792db0263e4e12fa1d9ee1a1b9a68540590da", "993bfca6149975a4013797ead55839a13a0fb152f68372bb0e0fd9499477f903"},
		{"050000800a27a726b4d0d6c223e119f635ef1d05024b216b7023fadc2d25949c90037e71e3e550726d210a2c688342e52440635e9cc14afe100665515151ac53782e9e4a5fa87f0a956f5b85509960285c22627c59483a5a4c28cce4b156e551406a7ee8355656a20043e38ce103bd9a274e288d020000aafe033252c7030005516a63656338eb8b41ca5104000653516365acac000000", "90d2886cb628813371c7d1bd02031b6ca66b42d1db4e118d65f31b2dccb63235", "9c0532e6788fe9e28b3b67f571989be77ae761dfd375c74bbf5db3cafaa1f9a5"},
		{"050000800a27a726b4d0d6c21fc998c31f4dd208010000000000000000000000000000000000000000000000000000000000000000ffffffff06041f4dd20800ffffffff015058e5754c2104000753ac51530051520001e5849f96bae6f2056f33ab1e6989d7d264adc97855a990103b4d1e6350d5c31a39c3caf69459e462f141be8b39037ffa255ce27e4ad7b566a29620a9f011ab08fb2ad3050652b3f65b8e34526a2a15fc2ddc5b5113e4882c7cca0dd5577be067ba7a175dae4bbe3ef4863d53708915090f47a068e227433f9e49d3aa09e356d8d66d0c0121e91a3c4aa3f27fa1b63396e2b41db908fdab8b18cc7304e94e970568f9421c0dbbbaf84598d972b0534f48a5e52670436aaa776ed2482ad703430201e53443c36dcfd34a0cb6637876105e79bf3bd58ec148cb64970e3223a91f71dfcfd5a04b667fbaf3d4b3b908b9828820dfecdd753750b5f9d2216e56c615272f854464c0ca4b1e85aedd038292c4e1a57744ebba010b9ebfbb011bd6f0b78805025d27f3c17746bae116c15d9f471f0f6288a150647b2afe9df7cccf01f5cde5f04680bbfed87f6cf429fb27ad6babe791766611cf5bc20e48bef119259b9b8a0e39c3df28cb9582ea338601cdc481b32fb82adeebb3dade25d1a3df20c37e712506b5d996c49a9f0f30ddcb91fe9004e1e83294a6c9203d94e8dc2cbb449de4155032604e47997016b304fd437d8235045e255a19b743a0a9f2e336b44cae307bb3987bd3e4e777fbb34c0ab8cc3d67466c0a88dd4ccad18a07a8d1068df5b629e5718d0f6df5c957cf71bb00a5178f175caca944e635c5159f738e2402a2d21aa081e10e456afb00b9f62416c8b9c0f7228f510729e0be3f305313d77f7379dc2af24869c6c74ee4471498861d192f0ff0f508285dab6b6a36ccf7d12256cc76b95503720ac672d08268d2cf7773b6ba2a5f664847bf707f2fc10c98f2f006ec22ccb5a8c8b7c40c7c2d49a6639b9f2ce33c25c04bc461e744dfa536b00d94baddf4f4d14044c695a33881477df124f0fcf206a9fb2e65e304cdbf0c4d2390170c130ab849c2f22b5cdd3921640c8cf1976ae1010b0dfd9cb2543e45f99749cc4d61f2e8aabfe98bd905fa39951b33ea769c45ab9531c57209862ad12fd76ba4807e65417b6cd12fa8ec916f013ebb8706a9a556c762f88500006effeda06c4be24b04846392e9d1e6930eae01fa21fbd700583fb598b92c8f4eb8a61aa6235db60f2841cf3a1c6ab54c67066844711d091eb931a1bd6281aedf2a0e8fab18817202a9be06402ed9cc720c16bfe881e4df4255e87afb7fc62f38116bbe03cd8a3cb11a27d568414782f47b1a44c97c680467694bc9709d32916c97e8006cbb07ba0e4180a3738038c374c4cce8f32959afb25f303f5815c4533124acf9d18940e77522ac5dc4b9570aae8f47b7f57fd8767bea1a24ae7bed65b409e1dd26b8dddd68858d6f5161f073d90636860a9aaee18629b06330a8ee30591debfcef56a026bb28c3b06ec2cfaf5b79ab72694d1d012a7594dd80ae7dfa0c00", "a3cbadd7a58d80a4c2f61809c24a2f086c58ceecaf7af9414c38bdbdc4e46e98", "ad64580ed3a28a3ba41e2d320b5ff2a07fa19db074afc455e92e0f326be08a6a"},
	}
	for i, v := range vectors {
		blob, err := hex.DecodeString(v.blob)
		if err != nil {
			t.Fatal(err)
		}
		tx, err := Parse(blob)
		if err != nil {
			t.Fatalf("vector %d parse: %v", i, err)
		}
		wantTxID, _ := hex.DecodeString(v.txid)
		gotTxID := tx.TxID()
		if !bytes.Equal(gotTxID[:], wantTxID) {
			t.Fatalf("vector %d txid: got %x want %x", i, gotTxID, wantTxID)
		}
		wantAuth, _ := hex.DecodeString(v.auth)
		gotAuth := tx.AuthDigest()
		if !bytes.Equal(gotAuth[:], wantAuth) {
			t.Fatalf("vector %d auth digest: got %x want %x", i, gotAuth, wantAuth)
		}
		rebuilt, err := Serialize(tx)
		if err != nil {
			t.Fatalf("vector %d serialize: %v", i, err)
		}
		if !bytes.Equal(rebuilt, blob) {
			t.Fatalf("vector %d round-trip mismatch", i)
		}
	}
}

func TestRejectsTrailingData(t *testing.T) {
	b, _ := Serialize(emptyTx())
	b = append(b, 0)
	_, e := Parse(b)
	if !errors.Is(e, ErrTrailingData) {
		t.Fatalf("got %v", e)
	}
}

func FuzzParse(f *testing.F) {
	b, _ := Serialize(emptyTx())
	f.Add(b)
	b, _ = Serialize(populatedTx())
	f.Add(b)
	f.Fuzz(func(t *testing.T, data []byte) {
		tx, e := Parse(data)
		if e != nil {
			return
		}
		out, e := Serialize(tx)
		if e != nil {
			t.Fatal(e)
		}
		if !bytes.Equal(data, out) {
			t.Fatal("lossy round-trip")
		}
	})
}
