package mf

import "errors"

const (
	// SenderHashLength is the required length in bytes for a sender hash.
	SenderHashLength = 16

	// MaxMessageSize is the maximum text size allowed in a message to fit within Reticulum MTU.
	// Calculated as MTU (500) - MaxHeaderSize (64) - SenderHashLength (16).
	MaxMessageSize = 420

	// testHashHex is a test hash used in unit tests.
	testHashHex = "0123456789abcdef0123456789abcdef"

	// errFmtExpected is a common error format string.
	errFmtExpected = "%w: expected %d, got %d"
)

var (
	// ErrInvalidHashLength is returned when a hash has an incorrect length.
	ErrInvalidHashLength = errors.New("invalid hash length")

	// ErrMessageTooShort is returned when message data is too short to be unpacked.
	ErrMessageTooShort = errors.New("message too short")

	// ErrMessageTooLong is returned when message text exceeds MaxMessageSize.
	ErrMessageTooLong = errors.New("message text too long")
)


