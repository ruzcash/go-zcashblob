package zcashblob_test

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"log"

	"github.com/ruzcash/go-zcashblob"
)

func ExampleParse() {
	const rawHex = "050000800a27a726b4d0d6c2c2eb518f68984d02010000000000000000000000000000000000000000000000000000000000000000ffffffff060468984d0200ffffffff00000000"
	blob, err := hex.DecodeString(rawHex)
	if err != nil {
		log.Fatal(err)
	}
	tx, err := zcashblob.Parse(blob)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("version=%d inputs=%d outputs=%d\n", tx.Version(), len(tx.TransparentInputs), len(tx.TransparentOutputs))

	// Output:
	// version=5 inputs=1 outputs=0
}

func ExampleNewTransactionV5() {
	tx := zcashblob.NewTransactionV5(0xc2d6d0b4)
	fmt.Printf("version=%d group=%08x valid=%t\n",
		tx.Version(), tx.VersionGroupID, tx.Validate() == nil)

	// Output:
	// version=5 group=26a7270a valid=true
}

func ExampleTransaction_Validate() {
	tx := zcashblob.NewTransactionV5(0xc2d6d0b4)
	tx.Orchard.Actions = make([]zcashblob.OrchardAction, 1)

	err := tx.Validate()
	fmt.Println(errors.Is(err, zcashblob.ErrInvalidStructure))

	// Output:
	// true
}

func ExampleSerializeToWriter() {
	tx := zcashblob.NewTransactionV5(0xc2d6d0b4)
	tx.TransparentOutputs = []zcashblob.TxOut{{Value: 1, ScriptPubKey: []byte{0x51}}}

	var wire bytes.Buffer
	if err := zcashblob.SerializeToWriter(tx, &wire); err != nil {
		log.Fatal(err)
	}
	parsed, err := zcashblob.ParseFromReader(&wire)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("outputs=%d value=%d\n",
		len(parsed.TransparentOutputs), parsed.TransparentOutputs[0].Value)

	// Output:
	// outputs=1 value=1
}

func ExampleTransaction_TxIDString() {
	tx := zcashblob.NewTransactionV5(0xc2d6d0b4)
	fmt.Println(tx.TxIDString())

	// Output:
	// 8e6b6d721fc653ef162daa85b32bff85144b9245add517cf710d5155cf5876df
}
