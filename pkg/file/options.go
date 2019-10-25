package file

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"sync"

	permitpool "github.com/damoonazarpazhooh/File-Ingestion/internal/permitpool"
	"github.com/palantir/stacktrace"
	"golang.org/x/crypto/hkdf"
)

// Option - options setter method
type Option func(*Storage)

// Storage -
type Storage struct {
	sync.Once
	stateLock   sync.RWMutex
	logOps      bool
	initialized bool
	logCh       chan string
	wg          sync.WaitGroup
	// -----
	path              string
	permitPool        permitpool.PermitPool
	downloadRateLimit int
	uploadRateLimit   int
	numberOfThreads   int
	encryptionKey     []byte
	nonce             []byte
}

// LogOps -
func LogOps() Option {
	return func(e *Storage) {
		e.stateLock.Lock()
		defer e.stateLock.Unlock()
		e.logOps = true
	}
}

// WithPath -
func WithPath(arg string) Option {
	return func(e *Storage) {
		e.stateLock.Lock()
		defer e.stateLock.Unlock()
		e.path = arg
	}
}

// WithUploadRateLimit -
func WithUploadRateLimit(arg int) Option {
	return func(e *Storage) {
		e.stateLock.Lock()
		defer e.stateLock.Unlock()
		e.uploadRateLimit = arg
	}
}

// WithDownloadRateLimit -
func WithDownloadRateLimit(arg int) Option {
	return func(e *Storage) {
		e.stateLock.Lock()
		defer e.stateLock.Unlock()
		e.downloadRateLimit = arg
	}
}

// WithNumberOfThreads -
func WithNumberOfThreads(arg int) Option {
	return func(e *Storage) {
		e.stateLock.Lock()
		defer e.stateLock.Unlock()
		e.numberOfThreads = arg
	}
}

// WithEncryption -
func WithEncryption(arg string) Option {
	return func(e *Storage) {
		e.stateLock.Lock()
		defer e.stateLock.Unlock()
		var (
			key [32]byte
		)

		hx := hex.EncodeToString([]byte(arg))
		masterkey, err := hex.DecodeString(hx)
		if err != nil {
			err = stacktrace.Propagate(err, "[ERROR] Cannot decode hex key")
			log.Fatal(err)
		}
		nonce, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000001")
		if err != nil {
			fmt.Printf("[ERROR] Cannot decode hex key: %v", err)
			log.Fatal(err)
		}
		// _, err = io.ReadFull(rand.Reader, nonce[:])
		// if err != nil {
		// 	err = stacktrace.Propagate(err, "[ERROR] Failed to read random data")
		// 	log.Fatal(err)
		// }
		kdf := hkdf.New(sha256.New, masterkey, nonce[:], nil)
		_, err = io.ReadFull(kdf, key[:])
		if err != nil {
			err = stacktrace.Propagate(err, "ERROR] Failed to derive encryption key")
			log.Fatal(err)
		}
		e.encryptionKey = key[:]
		e.nonce = nonce
	}
}
