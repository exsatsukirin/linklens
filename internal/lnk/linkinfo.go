package lnk

import (
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// LinkInfo contains information about the target's location, including
// volume information and local file path.
// Reference: MS-SHLLINK section 2.3 (LinkInfo)
type LinkInfo struct {
	Target string // Full target path, e.g. "C:\Program Files\MyApp\app.exe"
}

// MarshalBinary serializes the LinkInfo block.
func (li LinkInfo) MarshalBinary() ([]byte, error) {
	if li.Target == "" {
		return nil, fmt.Errorf("empty target path for LinkInfo")
	}

	// Normalize path
	target := filepath.ToSlash(li.Target)
	target = strings.ReplaceAll(target, "/", "\\")

	// Extract drive letter and path
	if len(target) < 2 || target[1] != ':' {
		return nil, fmt.Errorf("target must be absolute Windows path: %s", target)
	}

	// ANSI local base path (null-terminated byte string)
	ansiLocalBasePath := []byte(target + "\x00")

	// VolumeID (21 bytes: 16 bytes of fixed fields + "Data\0" ANSI label).
	// Per MS-SHLLINK 2.3.1 each field before the label is a 4-byte unsigned
	// integer, so DriveType must occupy a full uint32. Encoding it as 2 bytes
	// shifts the serial number and label offset and corrupts the block.
	volumeIDData := []byte{
		0x15, 0x00, 0x00, 0x00, // VolumeIDSize = 21
		0x03, 0x00, 0x00, 0x00, // DriveType = DRIVE_FIXED
		0x00, 0x00, 0x00, 0x00, // VolumeSerialNumber = 0 (unknown)
		0x10, 0x00, 0x00, 0x00, // VolumeLabelOffset = 16
		0x44, 0x61, 0x74, 0x61, 0x00, // VolumeLabel = "Data" + null
	}

	const linkInfoHeaderSize = 28
	const linkInfoFlags = 0x00000001 // VolumeIDAndLocalBasePath (ANSI)

	// VolumeID starts right after the header (already 4-byte aligned)
	volumeIDOffset := linkInfoHeaderSize

	// LocalBasePath (ANSI) follows VolumeID
	localBasePathOffset := volumeIDOffset + len(volumeIDData)

	// CommonPathSuffix follows ANSI path (empty null-terminated string)
	commonPathSuffixOffset := localBasePathOffset + len(ansiLocalBasePath)

	// Total size
	totalSize := commonPathSuffixOffset + 1 // +1 for null terminator of empty suffix

	buf := make([]byte, totalSize)

	// LinkInfo header
	binary.LittleEndian.PutUint32(buf[0:4], uint32(totalSize))
	binary.LittleEndian.PutUint32(buf[4:8], linkInfoHeaderSize)
	binary.LittleEndian.PutUint32(buf[8:12], linkInfoFlags)
	binary.LittleEndian.PutUint32(buf[12:16], uint32(volumeIDOffset))
	binary.LittleEndian.PutUint32(buf[16:20], uint32(localBasePathOffset))
	binary.LittleEndian.PutUint32(buf[20:24], 0) // CommonNetworkRelativeLinkOffset
	binary.LittleEndian.PutUint32(buf[24:28], uint32(commonPathSuffixOffset))

	// VolumeID
	copy(buf[volumeIDOffset:], volumeIDData)

	// LocalBasePath (ANSI)
	copy(buf[localBasePathOffset:], ansiLocalBasePath)

	// CommonPathSuffix (empty null-terminated string)
	buf[commonPathSuffixOffset] = 0

	return buf, nil
}

// WriteTo writes the serialized LinkInfo to w.
func (li LinkInfo) WriteTo(w io.Writer) (int64, error) {
	data, err := li.MarshalBinary()
	if err != nil {
		return 0, err
	}
	n, err := w.Write(data)
	return int64(n), err
}
