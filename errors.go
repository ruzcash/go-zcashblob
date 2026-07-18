package zcashblob

import "errors"

var (
	// ErrNonCanonical indicates a CompactSize value used a longer encoding.
	ErrNonCanonical = errors.New("non-canonical compactSize")
	// ErrTooLarge indicates input data or a transaction field exceeded a safety
	// limit.
	ErrTooLarge = errors.New("declared size exceeds limit")
	// ErrTrailingData indicates bytes remained after a complete transaction.
	ErrTrailingData = errors.New("trailing data after transaction")
	// ErrUnsupportedVersion indicates a non-v5 or non-Overwintered header.
	ErrUnsupportedVersion = errors.New("only overwintered Zcash v5 transactions are supported")
	// ErrInvalidStructure indicates inconsistent transaction fields.
	ErrInvalidStructure = errors.New("invalid transaction structure")
	// ErrNilReader indicates a nil reader was passed to ParseFromReader.
	ErrNilReader = errors.New("nil reader")
	// ErrNilWriter indicates a nil writer was passed to SerializeToWriter.
	ErrNilWriter = errors.New("nil writer")
)
