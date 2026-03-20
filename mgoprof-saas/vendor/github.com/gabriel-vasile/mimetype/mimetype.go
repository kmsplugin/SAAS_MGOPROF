// Package mimetype provides MIME type detection.
// This is a stub implementation used when the actual package is unavailable.
package mimetype

import (
	"io"
)

// MIME holds the detected MIME type.
type MIME struct {
	mime string
}

// String returns the MIME type string.
func (m *MIME) String() string {
	if m == nil {
		return ""
	}
	return m.mime
}

// Is reports whether the MIME type matches the given string.
func (m *MIME) Is(expectedMIME string) bool {
	return m.String() == expectedMIME
}

// DetectReader detects the MIME type from an io.Reader.
func DetectReader(r io.Reader) (*MIME, error) {
	return &MIME{mime: "application/octet-stream"}, nil
}

// Detect detects the MIME type from a byte slice.
func Detect(in []byte) *MIME {
	return &MIME{mime: "application/octet-stream"}
}

// DetectFile detects the MIME type of a file.
func DetectFile(path string) (*MIME, error) {
	return &MIME{mime: "application/octet-stream"}, nil
}
