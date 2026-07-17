package zcashblob_test

import (
	"encoding/hex"
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
