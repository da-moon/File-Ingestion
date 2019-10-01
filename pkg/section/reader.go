package section

import (
	"bytes"
	"fmt"
	"io"

	"github.com/mitchellh/hashstructure"
	"github.com/palantir/stacktrace"
)

// Read proxies the underlying SectionReader making this an io.Reader.
func (s *Section) Read(p []byte) (n int, err error) {
	// return s.reader.Read(p)
	return s.SectionReader.Read(p)
}

// Data reads from the embedded io.SectionReader and returns a copy of the
// []byte read.
func (s *Section) Data() ([]byte, error) {
	var buf bytes.Buffer
	var err error

	_, err = io.Copy(&buf, s.SectionReader)
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] could not load data to buffer from section reader ")
		return nil, err
	}
	hash, err := hashstructure.Hash(s, nil)
	if err != nil {
		stacktrace.Propagate(err, "[ERROR] hashstructure could not compute Section's data hash")
		return nil, err
	}
	s.Hash = fmt.Sprintf("%d", hash)
	return buf.Bytes(), nil
}
