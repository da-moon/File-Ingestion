package jsonutil

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/palantir/stacktrace"
)

// EncodeJSON - Encodes/Marshals the given object into JSON
func EncodeJSON(in interface{}) ([]byte, error) {
	if in == nil {
		return nil, stacktrace.NewError("input for encoding is nil")
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	if err := enc.Encode(in); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeJSONWithIndentation - Encodes/Marshals the given object into JSON
func EncodeJSONWithIndentation(in interface{}) ([]byte, error) {
	// if in == nil {
	// 	return nil, stacktrace.NewError("input for encoding is nil")
	// }
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)

	enc.SetIndent("", "    ")
	err := enc.Encode(in)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// EncodeJSONWithoutErr - Encodes/Marshals the given object into JSON but does not return an err
func EncodeJSONWithoutErr(in interface{}) []byte {
	res, _ := EncodeJSON(in)
	return res
}

// DecodeJSON tries to decompress the given data. The call to decompress, fails
// if the content was not compressed in the first place, which is identified by
// a canary byte before the compressed data. If the data is not compressed, it
// is JSON decoded directly. Otherwise the decompressed data will be JSON
// decoded.
func DecodeJSON(data []byte, out interface{}) error {
	if len(data) == 0 {
		return stacktrace.NewError("'data' being decoded is nil")
	}
	if out == nil {
		return stacktrace.NewError("output parameter 'out' is nil")
	}

	return DecodeJSONFromReader(bytes.NewReader(data), out)
}

// DecodeJSONFromReader - Decodes/Unmarshals the given io.Reader pointing to a JSON, into a desired object
func DecodeJSONFromReader(r io.Reader, out interface{}) error {
	if r == nil {
		return stacktrace.NewError("'io.Reader' being decoded is nil")
	}
	if out == nil {
		return stacktrace.NewError("output parameter 'out' is nil")
	}

	dec := json.NewDecoder(r)

	// While decoding JSON values, interpret the integer values as `json.Number`s instead of `float64`.
	dec.UseNumber()

	// Since 'out' is an interface representing a pointer, pass it to the decoder without an '&'
	return dec.Decode(out)
}
