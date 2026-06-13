package lnk

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"unicode/utf16"
)

// BuildIDList constructs an ITEMIDLIST (LinkTargetIDList) from a Windows path.
// The format is:
//   uint16 totalSize (excluding this field)
//   ITEMIDLIST entries...
//   uint16 terminalID (0x0000)
func BuildIDList(target string) ([]byte, error) {
	if target == "" {
		return nil, errors.New("empty target path")
	}

	// Normalize to backslashes
	target = filepath.ToSlash(target)
	target = strings.ReplaceAll(target, "/", "\\")

	// Parse drive letter
	if len(target) < 2 || target[1] != ':' {
		return nil, fmt.Errorf("target must be an absolute Windows path with drive letter: %s", target)
	}
	drive := strings.ToUpper(target[:2]) // e.g., "C:"
	pathParts := strings.Split(strings.Trim(target[2:], "\\"), "\\")
	fileName := pathParts[len(pathParts)-1]
	dirParts := pathParts[:len(pathParts)-1]

	var buf bytes.Buffer

	// Item 1: Root folder (CLSID_MyComputer: {20D04FE0-3AEA-1069-A2D8-08002B30309D})
	rootItem := makeRootItem()
	binary.Write(&buf, binary.LittleEndian, uint16(len(rootItem)+2))
	buf.Write(rootItem)

	// Item 2: Drive volume (e.g., C:\)
	driveItem := makeDriveItem(drive)
	binary.Write(&buf, binary.LittleEndian, uint16(len(driveItem)+2))
	buf.Write(driveItem)

	// Items for each directory in the path
	for _, part := range dirParts {
		if part == "" {
			continue
		}
		dirItem := makeFileSystemItem(part, true)
		binary.Write(&buf, binary.LittleEndian, uint16(len(dirItem)+2))
		buf.Write(dirItem)
	}

	// Final item: the file
	fileItem := makeFileSystemItem(fileName, false)
	binary.Write(&buf, binary.LittleEndian, uint16(len(fileItem)+2))
	buf.Write(fileItem)

	// Terminal ID (0x0000)
	binary.Write(&buf, binary.LittleEndian, uint16(0))

	// Total size prefix
	totalData := buf.Bytes()
	totalSize := uint16(len(totalData))
	result := make([]byte, 2+len(totalData))
	binary.LittleEndian.PutUint16(result[0:2], totalSize)
	copy(result[2:], totalData)

	return result, nil
}

// makeRootItem creates an ItemID for the MyComputer (This PC) root folder.
func makeRootItem() []byte {
	// CLSID_MyComputer: {20D04FE0-3AEA-1069-A2D8-08002B30309D}
	clsid := []byte{
		0xE0, 0x4F, 0xD0, 0x20, 0xEA, 0x3A, 0x69, 0x10,
		0xA2, 0xD8, 0x08, 0x00, 0x2B, 0x30, 0x30, 0x9D,
	}
	// Total root item data: 16 bytes CLSID + 4 extra = 20 bytes
	buf := make([]byte, 20)
	copy(buf[0:16], clsid)
	buf[16] = 0x1F // type (extension block)
	// buf[17:20] = 0 (extension field)
	return buf
}

// makeDriveItem creates an ItemID for a drive letter (e.g., C:\).
func makeDriveItem(drive string) []byte {
	driveName := drive + "\\"
	runes := []rune(driveName + "\x00")
	encoded := utf16.Encode(runes)
	buf := make([]byte, 2+len(encoded)*2)
	buf[0] = 0x2F // extension block type (drive)
	buf[1] = 0x00
	for i, r := range encoded {
		binary.LittleEndian.PutUint16(buf[2+i*2:], r)
	}
	return buf
}

// makeFileSystemItem creates an ItemID for a file system entry.
func makeFileSystemItem(name string, isDir bool) []byte {
	runes := []rune(name + "\x00")
	encoded := utf16.Encode(runes)

	buf := make([]byte, 2+len(encoded)*2)
	buf[0] = 0x32 // extension block type (file system entry)
	buf[1] = 0x00 // unused
	for i, r := range encoded {
		binary.LittleEndian.PutUint16(buf[2+i*2:], r)
	}
	return buf
}

// IDListProxy is a helper to write the full LinkTargetIDList to a writer.
type IDListProxy struct {
	Target string
}

// WriteTo writes the serialized IDList to w.
func (p IDListProxy) WriteTo(w io.Writer) (int64, error) {
	data, err := BuildIDList(p.Target)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(data)
	return int64(n), err
}
