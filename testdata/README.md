# ZIP-244 test vectors

`zip_0244.json` is an unmodified copy of the native-order ZIP-244 test-vector
corpus published by the Zcash project.

- Source repository: https://github.com/zcash/zcash-test-vectors
- Source path: `test-vectors/json/zip_0244.json`
- Pinned revision: `78321beacb0e0477e33cd002b56585a107c2708c`
- SHA-256: `3d20892f19cec18afba2ef2907bb8426192d0ea4eac2c0060c2f31ff78ad93a3`
- Retrieved: 2026-07-18

The file contains one provenance row, one schema row, and ten test vectors.
Each vector has thirteen fields. The tests currently consume the serialized
transaction, transaction ID, and authorizing-data digest; retaining the exact
upstream file keeps the corpus independently verifiable.

The corpus is used under the MIT license at the recipient's option. The
applicable license text is in `LICENSE-MIT-zcash-test-vectors`.

Do not edit the fixture in place. Update it only by importing a complete
upstream revision and intentionally updating the pinned revision, checksum,
and test expectations together.
