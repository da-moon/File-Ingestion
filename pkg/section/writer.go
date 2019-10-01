package section

import (
	"bytes"

	"github.com/palantir/stacktrace"
)

// Merge uses write at to add bytes to a file section
// it decompresses and decrypts bytes if needed
func (s *Section) Merge(p []byte) (int, error) {

	buf := bytes.NewBuffer(p)
	n, err := s.SectionWriter.WriteAt(buf.Bytes(), s.Start)
	if err != nil {
		err = stacktrace.Propagate(err, "[ERROR] failed to write chunk data")
		return 0, err
	}

	return n, nil
}
