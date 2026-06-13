package lnk

import (
	"encoding/binary"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf16"
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
	localBasePath := target

	// Convert to UTF-16LE
	drive := strings.ToUpper(target[:2]) // "C:"
	_ = drive

	pathUtf16 := utf16.Encode([]rune(localBasePath + "\x00"))

	// Build VolumeID
	var volumeIDBuf []byte
	volumeIDHeaderSize := uint32(4 + 2 + 4 + 4) // 14 bytes
	volumeIDBuf = append(volumeIDBuf, byte(volumeIDHeaderSize), byte(volumeIDHeaderSize>>8), byte(volumeIDHeaderSize>>16), byte(volumeIDHeaderSize>>24))
	volumeIDBuf = append(volumeIDBuf, 0x03, 0x00)                     // DriveType 3 = DRIVE_FIXED
	volumeIDBuf = append(volumeIDBuf, 0, 0, 0, 0)                    // VolumeSerialNumber (0 = unknown)
	volumeIDBuf = append(volumeIDBuf, 0, 0, 0, 0)                    // VolumeLabelOffset (0 = no label)
	volumeIDData := volumeIDBuf

	// Build LinkInfo:
	//   uint32 LinkInfoSize
	//   uint32 LinkInfoHeaderSize (always 0x0000001C = 28)
	//   uint32 LinkInfoFlags (0 = VolumeIDAndLocalBasePath)
	//   uint32 VolumeIDOffset (relative to start of LinkInfo)
	//   uint32 LocalBasePathOffset (relative to start of LinkInfo)
	//   uint32 CommonNetworkRelativeLinkOffset (0 = none)
	//   uint32 CommonPathSuffixOffset (0 = none)
	//   [padding to 4-byte alignment for VolumeIDOffset]
	//   VolumeIDBlock (variable)
	//   LocalBasePath (UTF-16LE null-terminated)
	//   [padding to 4-byte alignment]

	const linkInfoHeaderSize = 28

	// Align VolumeIDOffset to 4-byte boundary
	volumeIDOffset := align4(linkInfoHeaderSize)

	// LocalBasePathOffset = after VolumeID
	localBasePathOffset := align4(volumeIDOffset + len(volumeIDData))

	// Total size = after LocalBasePath
	localBasePathBytes := make([]byte, len(pathUtf16)*2)
	for i, r := range pathUtf16 {
		binary.LittleEndian.PutUint16(localBasePathBytes[i*2:], r)
	}
	totalSize := align4(localBasePathOffset + len(localBasePathBytes))

	buf := make([]byte, totalSize)

	// LinkInfoSize
	binary.LittleEndian.PutUint32(buf[0:4], uint32(totalSize))
	// LinkInfoHeaderSize
	binary.LittleEndian.PutUint32(buf[4:8], linkInfoHeaderSize)
	// LinkInfoFlags
	binary.LittleEndian.PutUint32(buf[8:12], 0)
	// VolumeIDOffset
	binary.LittleEndian.PutUint32(buf[12:16], uint32(volumeIDOffset))
	// LocalBasePathOffset
	binary.LittleEndian.PutUint32(buf[16:20], uint32(localBasePathOffset))
	// CommonNetworkRelativeLinkOffset
	binary.LittleEndian.PutUint32(buf[20:24], 0)
	// CommonPathSuffixOffset
	binary.LittleEndian.PutUint32(buf[24:28], 0)

	// VolumeID
	copy(buf[volumeIDOffset:volumeIDOffset+len(volumeIDData)], volumeIDData)

	// LocalBasePath
	copy(buf[localBasePathOffset:localBasePathOffset+len(localBasePathBytes)], localBasePathBytes)

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

func align4(n int) int {
	return (n + 3) & ^3
}
