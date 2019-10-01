package file

import (
	"encoding/binary"

	"github.com/palantir/stacktrace"
)

// Errors
var (
	// ErrInvalidPayloadSize ...
	ErrInvalidPayloadSize = stacktrace.NewError("invalid payload size")
	// ErrAuthentication ...
	ErrAuthentication = stacktrace.NewError("authentication failed")
	// ErrLargerSizeThanExpected ...
	ErrLargerSizeThanExpected = stacktrace.NewError("data size is too large")
	// ErrNonceMismatch ...
	ErrNonceMismatch = stacktrace.NewError("header nonce mismatch")
	// ErrUnexpectedEOF ...
	ErrUnexpectedEOF = stacktrace.NewError("unexpected end of file (EOF)")
	// ErrUnexpectedData ...
	ErrUnexpectedData = stacktrace.NewError("unexpected data after final burst of data")
	// ErrInvalidDecryptedSize ...
	ErrInvalidDecryptedSize = stacktrace.NewError("size is not valid")
)

// Consts
const (
	// AES256GCM ...
	AES256GCM byte = iota
	// TagSize ...
	TagSize = 16
	// HeaderSize ...
	HeaderSize = 16
	// GCM standard nounce size
	StandardNonceSize = 12
	// 1000 0000
	HeaderFinalFlag = 0x80
	// KeySize ...
	KeySize = 32
	// MaxPayloadSize ...
	MaxPayloadSize = 1 << 16
	// MaxBufferSize ...
	MaxBufferSize = HeaderSize + MaxPayloadSize + TagSize
	// MaxDecryptedSize ...
	MaxDecryptedSize = 1 << 48
	// MaxEncryptedSize ...
	MaxEncryptedSize = MaxDecryptedSize + ((HeaderSize + TagSize) * 1 << 32)
)

// Buffer ...
type Buffer []byte

// Header ...
func (b Buffer) Header() Header {
	return Header(b[:HeaderSize])
}

// Data ...
func (b Buffer) Data() []byte {
	return b[HeaderSize : HeaderSize+b.Header().GetLength()]
}

// Ciphertext ...
func (b Buffer) Ciphertext() []byte {
	return b[HeaderSize:b.GetLength()]
}

// GetLength ...
func (b Buffer) GetLength() int {
	return HeaderSize + TagSize + b.Header().GetLength()
}

// Header ...
type Header []byte

// GetLength ...
func (h Header) GetLength() int {
	return int(binary.LittleEndian.Uint32(h[0:HeaderSize-StandardNonceSize])) + 1
}

// SetLength ...
func (h Header) SetLength(length int) {
	binary.LittleEndian.PutUint32(h[0:HeaderSize-StandardNonceSize], uint32(length-1))
}

// IsFinal ...
func (h Header) IsFinal() bool {
	return h[HeaderSize-StandardNonceSize]&HeaderFinalFlag == HeaderFinalFlag
}

// Nonce ...
func (h Header) Nonce() []byte {
	return h[HeaderSize-StandardNonceSize : HeaderSize]
}

// AddData ...
func (h Header) AddData() []byte {
	return h[:HeaderSize-StandardNonceSize]
}

// SetRand ...
func (h Header) SetRand(randVal []byte, final bool) {
	copy(h[HeaderSize-StandardNonceSize:], randVal)
	if final {
		//  h[HeaderSize - StandardNonceSize] | 1000 0000
		h[HeaderSize-StandardNonceSize] |= HeaderFinalFlag
	} else {
		//  h[HeaderSize - StandardNonceSize] | 0111 1111
		h[HeaderSize-StandardNonceSize] &= 0x7F
	}
}
