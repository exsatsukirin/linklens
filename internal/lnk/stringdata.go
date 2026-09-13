package lnk

import (
	"bytes"
	"encoding/binary"
	"io"
	"unicode/utf16"
)

// StringData holds the optional string fields of a .lnk file.
// Strings are written in the order defined by MS-SHLLINK 2.4:
// Name, RelativePath, WorkingDir, Arguments, IconLocation.
// Each string is prefixed with a uint16 character count (including null terminator),
// then the UTF-16LE encoded string with a 2-byte null terminator (\x00\x00).
//
// A field MUST be present if and only if its corresponding LinkFlags bit is set,
// so absent fields are omitted entirely rather than written as a zero count.
// Writing placeholders for absent fields would shift every following field.
type StringData struct {
	HasWorkingDir   bool
	WorkingDir      string
	HasArguments    bool
	Arguments       string
	HasIconLocation bool
	IconLocation    string
}

// encodeUTF16Le encodes a Go string to UTF-16LE bytes with null terminator.
// Returns (count, data) where count includes the null terminator.
func encodeUTF16Le(s string) (count uint16, data []byte) {
	runes := []rune(s + "\x00")
	encoded := utf16.Encode(runes)
	buf := make([]byte, len(encoded)*2)
	for i, r := range encoded {
		binary.LittleEndian.PutUint16(buf[i*2:], r)
	}
	return uint16(len(encoded)), buf
}

// MarshalBinary serializes the StringData block.
func (sd StringData) MarshalBinary() ([]byte, error) {
	var buf bytes.Buffer

	// Fields are emitted in MS-SHLLINK order and only when their LinkFlags bit
	// is set. Name and RelativePath are not supported and are therefore omitted
	// entirely (never written as zero-count placeholders).

	// WorkingDir
	if sd.HasWorkingDir && sd.WorkingDir != "" {
		count, data := encodeUTF16Le(sd.WorkingDir)
		binary.Write(&buf, binary.LittleEndian, count)
		buf.Write(data)
	}

	// Arguments
	if sd.HasArguments && sd.Arguments != "" {
		count, data := encodeUTF16Le(sd.Arguments)
		binary.Write(&buf, binary.LittleEndian, count)
		buf.Write(data)
	}

	// IconLocation
	if sd.HasIconLocation && sd.IconLocation != "" {
		count, data := encodeUTF16Le(sd.IconLocation)
		binary.Write(&buf, binary.LittleEndian, count)
		buf.Write(data)
	}

	return buf.Bytes(), nil
}

// WriteTo writes the serialized StringData to w.
func (sd StringData) WriteTo(w io.Writer) (int64, error) {
	data, err := sd.MarshalBinary()
	if err != nil {
		return 0, err
	}
	n, err := w.Write(data)
	return int64(n), err
}
