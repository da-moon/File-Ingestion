package section

import (
	"io"
)

// Section ...
type Section struct {
	Start         int64             `json:"start" mapstructure:"start"`
	End           int64             `json:"end" mapstructure:"end"`
	Size          int64             `json:"size" mapstructure:"size"`
	Number        int               `json:"number" mapstructure:"number"`
	Hash          string            `json:"hash" mapstructure:"hash"`
	SectionReader *io.SectionReader `json:"-" mapstructure:"-"`
	SectionWriter io.WriterAt       `json:"-" mapstructure:"-"`
}

// New ...
func New(start, size int64, number int, reader io.ReaderAt, writer io.WriterAt) *Section {
	result := &Section{
		Start:         start,
		End:           start + size,
		Size:          size,
		Number:        number,
		SectionReader: io.NewSectionReader(reader, start, size),
		SectionWriter: writer,
	}
	return result
}
