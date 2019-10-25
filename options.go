package chunker

import (
	"path/filepath"

	"github.com/damoonazarpazhooh/File-Ingestion/pkg/utils"
	"github.com/kardianos/osext"
	"github.com/palantir/stacktrace"
)

// Option - this method is used to change splitter's configuration after
// crating a new instance
func (s *Multipart) Option(opts ...Option) error {
	var err error
	for _, opt := range opts {
		opt(s)
	}
	s.stateLock.Lock()
	defer s.stateLock.Unlock()
	if len(s.root) != 0 {
		s.root, err = filepath.Abs(s.root)
		if err != nil {
			err = stacktrace.Propagate(err, "[FATAL] Multipart : Error setting up root path (%s) for splitter", s.root)
			return err
		}
	}
	if len(s.root) == 0 {
		path := "tmp"
		selfPath, _ := osext.ExecutableFolder()
		s.root = utils.PathJoin(selfPath, path)
	}
	return nil
}

// Option ...
type Option func(*Multipart)

// LogOps ...
func LogOps() Option {
	return func(s *Multipart) {
		s.stateLock.Lock()
		defer s.stateLock.Unlock()
		s.logOps = true
	}
}

// WithRootPath -
func WithRootPath(arg string) Option {
	return func(s *Multipart) {
		s.stateLock.Lock()
		defer s.stateLock.Unlock()
		s.root = arg
	}
}

// WithMetadataDirectoryPath -
func WithMetadataDirectoryPath(arg string) Option {
	return func(s *Multipart) {
		s.stateLock.Lock()
		defer s.stateLock.Unlock()
		s.rootMetaName = arg
	}
}

// WithChunksDirectoryPath -
func WithChunksDirectoryPath(arg string) Option {
	return func(s *Multipart) {
		s.stateLock.Lock()
		defer s.stateLock.Unlock()
		s.rootChunksDir = arg
	}
}

// WithEncryption -
func WithEncryption(key string) Option {
	return func(s *Multipart) {
		s.stateLock.Lock()
		defer s.stateLock.Unlock()
		s.encryptionKey = key
	}
}

// WithChunkSizeInMegabytes -
func WithChunkSizeInMegabytes(arg int64) Option {
	return func(s *Multipart) {
		s.stateLock.Lock()
		defer s.stateLock.Unlock()
		// 000 is for versioning
		s.chunkSize = int64(arg * 1 << 20)
	}
}

// WithChunkSizeInKilobytes -
func WithChunkSizeInKilobytes(arg int64) Option {
	return func(s *Multipart) {
		s.stateLock.Lock()
		defer s.stateLock.Unlock()
		s.chunkSize = int64(arg * 1 << 10)
	}
}
