package file

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/binary"
	"io"

	"github.com/palantir/stacktrace"
)

// encryptor ...
type encryptor struct {
	reader         io.Reader
	key            []byte
	rand           io.Reader
	buffer         Buffer
	offset         int
	lastByte       byte
	firstRead      bool
	cipherID       byte
	cipher         cipher.AEAD
	randVal        []byte
	sequenceNumber uint32
	finalized      bool
}

// New returns an io.Reader that encrypts everything it reads.
func (s *Storage) newEncryptor(reader io.Reader) (*encryptor, error) {
	var err error
	result := &encryptor{
		key:       s.encryptionKey,
		reader:    reader,
		buffer:    make(Buffer, MaxBufferSize),
		firstRead: true,
	}
	if len(result.key) != KeySize {
		err := stacktrace.NewError("[ERROR] encryptor cannot be initialized due to invalid key size")
		return nil, err
	}
	result.rand = rand.Reader
	result.cipherID = []byte{AES256GCM}[0]
	aes256, err := aes.NewCipher(result.key)
	if err != nil {
		return nil, err
	}
	result.cipher, err = cipher.NewGCM(aes256)
	if err != nil {
		return nil, err
	}
	var randVal [12]byte
	_, err = io.ReadFull(result.rand, randVal[:])
	if err != nil {
		return nil, err
	}
	result.randVal = randVal[:]
	return result, nil
}

// Read ...
func (e *encryptor) Read(p []byte) (int, error) {
	var (
		count int
		err   error
	)
	if e.firstRead {
		e.firstRead = false
		_, err = io.ReadFull(e.reader, e.buffer[HeaderSize:HeaderSize+1])
		if err != nil && err != io.EOF {
			return 0, err
		}
		if err == io.EOF {
			e.finalized = true
			return 0, io.EOF
		}
		e.lastByte = e.buffer[HeaderSize]
	}

	if e.offset > 0 {
		remaining := e.buffer.GetLength() - e.offset
		if len(p) < remaining {
			e.offset += copy(p, e.buffer[e.offset:e.offset+len(p)])
			return len(p), nil
		}
		count = copy(p, e.buffer[e.offset:e.offset+remaining])
		p = p[remaining:]
		e.offset = 0
	}
	if e.finalized {
		return count, io.EOF
	}
	finalize := false

	for len(p) >= MaxBufferSize {
		e.buffer[HeaderSize] = e.lastByte
		nn, err := io.ReadFull(
			e.reader,
			e.buffer[HeaderSize+1:HeaderSize+1+MaxPayloadSize],
		)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			err = stacktrace.Propagate(err, "[ERROR] encryptor failed to read maximum payload from reader")
			return count, err
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			finalize = true
			e.seal(p, e.buffer[HeaderSize:HeaderSize+1+nn], finalize)
			return count + HeaderSize + TagSize + 1 + nn, io.EOF
		}
		e.lastByte = e.buffer[HeaderSize+MaxPayloadSize]
		e.seal(p, e.buffer[HeaderSize:HeaderSize+MaxPayloadSize], finalize)
		p = p[MaxBufferSize:]
		count += MaxBufferSize
	}
	if len(p) > 0 {
		e.buffer[HeaderSize] = e.lastByte
		nn, err := io.ReadFull(
			e.reader,
			e.buffer[HeaderSize+1:HeaderSize+1+MaxPayloadSize],
		)
		if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
			err = stacktrace.Propagate(err, "[ERROR] encryptor failed to read from reader")
			return count, err
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			finalize = true
			e.seal(e.buffer, e.buffer[HeaderSize:HeaderSize+1+nn], finalize)
			if len(p) > e.buffer.GetLength() {
				count += copy(p, e.buffer[:e.buffer.GetLength()])
				return count, io.EOF
			}
		} else {
			e.lastByte = e.buffer[HeaderSize+MaxPayloadSize]
			e.seal(e.buffer, e.buffer[HeaderSize:HeaderSize+MaxPayloadSize], finalize)
		}
		e.offset = copy(p, e.buffer[:len(p)])
		count += e.offset
	}
	return count, nil
}

func (e *encryptor) seal(dst, src []byte, finalize bool) {
	if e.finalized {
		err := stacktrace.NewError("[ERROR] sealing byte bursts after Close is not permitted")
		panic(err)
	}
	e.finalized = finalize
	header := Header(dst[:HeaderSize])
	header.SetLength(len(src))
	header.SetRand(e.randVal, finalize)
	var nonce [StandardNonceSize]byte
	copy(nonce[:], header.Nonce())
	binary.LittleEndian.PutUint32(
		nonce[8:],
		binary.LittleEndian.Uint32(nonce[8:])^e.sequenceNumber,
	)
	e.cipher.Seal(dst[HeaderSize:HeaderSize], nonce[:], src, header.AddData())
	e.sequenceNumber++
}

// decryptor ...
type decryptor struct {
	rand           io.Reader
	key            []byte
	reader         io.Reader
	buffer         Buffer
	header         Header
	finalized      bool
	sequenceNumber uint32
	cipher         cipher.AEAD
	offset         int
}

// newDecryptor returns an io.Reader decrypts everything it reads.
func (s *Storage) newDecryptor(reader io.Reader) (*decryptor, error) {
	result := &decryptor{
		key:    s.encryptionKey,
		reader: reader,
		buffer: make(Buffer, MaxBufferSize),
	}

	if len(result.key) != KeySize {
		err := stacktrace.NewError("[ERROR] Encryptor cannot be initialized due to invalid key size")
		return nil, err
	}
	result.rand = rand.Reader
	aes256, err := aes.NewCipher(result.key)
	if err != nil {
		return nil, err
	}
	result.cipher, err = cipher.NewGCM(aes256)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Read ...
func (d *decryptor) Read(p []byte) (n int, err error) {
	if d.offset > 0 {
		remaining := len(d.buffer.Data()) - d.offset
		if len(p) < remaining {
			n = copy(p, d.buffer.Data()[d.offset:d.offset+len(p)])
			d.offset += n
			return n, nil
		}
		n = copy(p, d.buffer.Data()[d.offset:])
		p = p[remaining:]
		d.offset = 0
	}
	for len(p) >= MaxPayloadSize {
		nn, err := io.ReadFull(d.reader, d.buffer)
		if err == io.EOF && !d.finalized {
			err = ErrUnexpectedEOF
			err = stacktrace.Propagate(err, "[ERROR] decryptor Read failed because it reached EOF without getting final data burst")
			return n, err
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			// err = stacktrace.Propagate(err, "[ERROR] decryptor Read failed because it reached EOF or reading from reader failed")

			return n, err
		}
		err = d.metadata(p, d.buffer[:nn])
		if err != nil {
			err = stacktrace.Propagate(err, "[ERROR] decryptor Read failed because it could not initialize metadata for the sequence")

			return n, err
		}
		p = p[len(d.buffer.Data()):]
		n += len(d.buffer.Data())
	}
	if len(p) > 0 {
		nn, err := io.ReadFull(d.reader, d.buffer)
		if err == io.EOF && !d.finalized {
			err = ErrUnexpectedEOF
			err = stacktrace.Propagate(err, "[ERROR] decryptor Read failed because it reached EOF without getting final data burst")
			return n, err
		}
		if err != nil && err != io.ErrUnexpectedEOF {
			// err = stacktrace.Propagate(err, "[ERROR] decryptor Read failed because it reached EOF or reading from reader failed")
			return n, err
		}
		err = d.metadata(d.buffer[HeaderSize:], d.buffer[:nn])
		if err != nil {
			err = stacktrace.Propagate(err, "[ERROR] decryptor Read failed because it could not initialize metadata for the sequence")
			return n, err
		}
		payload := d.buffer.Data()
		if len(p) < len(payload) {
			d.offset = copy(p, payload[:len(p)])
			n += d.offset
		} else {
			n += copy(p, payload)
		}
	}
	return n, nil
}

// metadata ...
func (d *decryptor) metadata(dst, src []byte) error {
	if d.finalized {

		return ErrUnexpectedData
	}
	if len(src) <= HeaderSize+TagSize {
		err := ErrInvalidPayloadSize
		err = stacktrace.Propagate(err, "[ERROR] Could not generate metadata for decryptor because current source length (%v) is lower than or equal to sum of header size constant (%v) and Tag size constant (%v)", len(src), HeaderSize, TagSize)
		return err
	}

	header := Buffer(src).Header()
	if d.header == nil {
		d.header = make([]byte, HeaderSize)
		copy(d.header, header)
	}
	if len(src) != HeaderSize+TagSize+header.GetLength() {
		err := ErrInvalidPayloadSize
		err = stacktrace.Propagate(err, "[ERROR] Could not generate metadata for decryptor because current source length (%v) is not equal to the sum of header size constant (%v) and Tag size constant (%v) and header size (%v)", len(src), HeaderSize, TagSize, header.GetLength())
		return err
	}
	if !header.IsFinal() && header.GetLength() != MaxPayloadSize {
		err := ErrInvalidPayloadSize
		err = stacktrace.Propagate(err, "[ERROR] Could not generate metadata for decryptor because unfinalized header length (%v) is not equal to the sum of max payload size constant (%v)", header.GetLength(), MaxPayloadSize)
		return err
	}
	refNonce := d.header.Nonce()
	if header.IsFinal() {
		d.finalized = true
		refNonce[0] |= HeaderFinalFlag
	}
	if subtle.ConstantTimeCompare(header.Nonce(), refNonce[:]) != 1 {
		return ErrNonceMismatch
	}
	var nonce [StandardNonceSize]byte
	copy(nonce[:], header.Nonce())
	binary.LittleEndian.PutUint32(
		nonce[8:],
		binary.LittleEndian.Uint32(nonce[8:])^d.sequenceNumber,
	)
	cipher := d.cipher
	ciphertext := src[HeaderSize : HeaderSize+header.GetLength()+TagSize]
	_, err := cipher.Open(
		dst[:0],
		nonce[:],
		ciphertext,
		header.AddData(),
	)
	if err != nil {
		err = stacktrace.Propagate(err, ErrAuthentication.Error())
		err = stacktrace.Propagate(err, "[ERROR] Could not generate metadata for decryptor becuase cipher failed with decrypting and authenticating ciphertext")
		return err
	}
	d.sequenceNumber++
	return nil
}
