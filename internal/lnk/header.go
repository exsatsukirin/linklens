package lnk

import (
	"encoding/binary"
	"io"
	"time"
)

// ShellLinkHeader represents the first 76 bytes of a .lnk file.
// Reference: https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-shllink/c4399181-2c1b-4c10-ba3f-3e7fbd2c86d3
type ShellLinkHeader struct {
	LinkFlags      uint32
	FileAttributes uint32
	CreationTime   time.Time
	AccessTime     time.Time
	WriteTime      time.Time
	FileSize       uint32
	IconIndex      uint32
	ShowCommand    uint32
	HotKey         uint16
}

// CLSID_ShellLink is the class identifier for shell link objects.
var CLSID_ShellLink = [16]byte{
	0x01, 0x14, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00,
	0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46,
}

// timeToFileTime converts Go time to Windows FILETIME (100-ns intervals since 1601-01-01).
func timeToFileTime(t time.Time) uint64 {
	if t.IsZero() {
		return 0
	}
	// Windows epoch: January 1, 1601 00:00:00 UTC
	// Go epoch: January 1, 1970 00:00:00 UTC
	// Difference: 11644473600 seconds
	const epochDiff = 11644473600
	nano := t.UnixNano()
	if nano == 0 {
		return 0
	}
	ft := uint64(nano/100) + epochDiff*10000000
	return ft
}

// MarshalBinary writes the ShellLinkHeader to a byte slice (76 bytes).
func (h ShellLinkHeader) MarshalBinary() ([]byte, error) {
	buf := make([]byte, 76)
	// HeaderSize (4 bytes) = 0x0000004C
	binary.LittleEndian.PutUint32(buf[0:4], 0x0000004C)
	// LinkCLSID (16 bytes)
	copy(buf[4:20], CLSID_ShellLink[:])
	// LinkFlags (4 bytes)
	binary.LittleEndian.PutUint32(buf[20:24], h.LinkFlags)
	// FileAttributes (4 bytes)
	binary.LittleEndian.PutUint32(buf[24:28], h.FileAttributes)
	// CreationTime (8 bytes)
	binary.LittleEndian.PutUint64(buf[28:36], timeToFileTime(h.CreationTime))
	// AccessTime (8 bytes)
	binary.LittleEndian.PutUint64(buf[36:44], timeToFileTime(h.AccessTime))
	// WriteTime (8 bytes)
	binary.LittleEndian.PutUint64(buf[44:52], timeToFileTime(h.WriteTime))
	// FileSize (4 bytes)
	binary.LittleEndian.PutUint32(buf[52:56], h.FileSize)
	// IconIndex (4 bytes)
	binary.LittleEndian.PutUint32(buf[56:60], h.IconIndex)
	// ShowCommand (4 bytes)
	binary.LittleEndian.PutUint32(buf[60:64], h.ShowCommand)
	// HotKey (2 bytes)
	binary.LittleEndian.PutUint16(buf[64:66], h.HotKey)
	// Reserved (10 bytes) - already 0
	return buf, nil
}

// WriteTo writes the serialized header to w.
func (h ShellLinkHeader) WriteTo(w io.Writer) (int64, error) {
	data, err := h.MarshalBinary()
	if err != nil {
		return 0, err
	}
	n, err := w.Write(data)
	return int64(n), err
}
