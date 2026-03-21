// Package mimetype is a stub replacement for gabriel-vasile/mimetype.
// Only the symbols used by go-playground/validator are implemented.
package mimetype

import (
	"io"
)

// MIME represents a MIME type.
type MIME struct {
	mime string
}

func (m *MIME) String() string { return m.mime }
func (m *MIME) Is(mime string) bool { return m.mime == mime }

// DetectReader detects the MIME type from an io.Reader.
func DetectReader(r io.Reader) (*MIME, error) {
	return &MIME{mime: "application/octet-stream"}, nil
}

// Detect detects the MIME type of a byte slice.
func Detect(raw []byte) *MIME {
	return &MIME{mime: "application/octet-stream"}
}

// DetectFile detects the MIME type from a file path.
func DetectFile(path string) (*MIME, error) {
	return &MIME{mime: "application/octet-stream"}, nil
}
