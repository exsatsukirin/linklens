package lnk

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf16"
)

// BuildIDList constructs an ITEMIDLIST (LinkTargetIDList) from a Windows path.
// The format is:
//
//	uint16 totalSize (excluding this field)
//	ITEMIDLIST entries...
//	uint16 terminalID (0x0000)
func BuildIDList(target string, fileSize uint32) ([]byte, error) {
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

	now := time.Now()

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
		dirItem := makeDirItem(part, now)
		binary.Write(&buf, binary.LittleEndian, uint16(len(dirItem)+2))
		buf.Write(dirItem)
	}

	// Final item: the file
	fileItem := makeFileItem(fileName, fileSize, now)
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
// Returns data WITHOUT the 2-byte cb (size) prefix; BuildIDList adds it.
// Format: 0x1F (type) + 16-byte CLSID = 18 bytes of data.
func makeRootItem() []byte {
	// CLSID_MyComputer: {20D04FE0-3AEA-1069-A2D8-08002B30309D}
	clsid := []byte{
		0xE0, 0x4F, 0xD0, 0x20, 0xEA, 0x3A, 0x69, 0x10,
		0xA2, 0xD8, 0x08, 0x00, 0x2B, 0x30, 0x30, 0x9D,
	}
	buf := make([]byte, 18)
	buf[0] = 0x1F // type (shell folder indicator)
	buf[1] = 0x50 // SortIndex for MyComputer (This PC)
	copy(buf[2:18], clsid)
	return buf
}

// makeDriveItem creates an ItemID for a drive letter (e.g., C:\).
// Returns data WITHOUT the 2-byte cb prefix; BuildIDList adds it.
// Format: 0x2F (type) + ASCII drive path "X:\" + zero-padding = 23 bytes data.
func makeDriveItem(drive string) []byte {
	// Fixed 23-byte data: 0x2F + "X:\" (3 ASCII bytes) + 19 zero-padding
	buf := make([]byte, 23)
	buf[0] = 0x2F // drive/volume type indicator
	copy(buf[1:], []byte(drive+"\\"))
	// remaining bytes are zero-padded
	return buf
}

// makeDirItem creates an ItemID for a directory with BEEF0004 extension block.
func makeDirItem(name string, t time.Time) []byte {
	longNameBytes := make([]byte, (len([]rune(name))+1)*2)
	putUTF16LE(longNameBytes, name)
	// BEEF0004: header(8) + timestamps(16) + fixedPad(20) + longName + extCount(2) + end(1)
	beeDataSize := 8 + 16 + 20 + len(longNameBytes) + 3
	// Item: header(12) + name + null + BEEF0004(sizeField(2) + data)
	itemDataSize := 12 + len(name) + 1 + 2 + beeDataSize
	buf := make([]byte, itemDataSize)
	buf[0] = 0x31
	buf[10] = 0x10
	copy(buf[12:], []byte(name))
	buf[12+len(name)] = 0x00

	beeStart := 12 + len(name) + 1 // BEEF0004 size field starts here
	ft := timeToFileTime(t)
	binary.LittleEndian.PutUint16(buf[beeStart:beeStart+2], uint16(2+beeDataSize))
	bee := beeStart + 2 // BEEF0004 data starts here
	binary.LittleEndian.PutUint16(buf[bee:bee+2], 9)
	binary.LittleEndian.PutUint32(buf[bee+4:bee+8], 0xBEEF0004)
	binary.LittleEndian.PutUint64(buf[bee+8:bee+16], ft)
	binary.LittleEndian.PutUint64(buf[bee+16:bee+24], ft)
	// 20 bytes fixed padding at bee+24
	copy(buf[bee+44:], longNameBytes)
	lnEnd := bee + 44 + len(longNameBytes)
	binary.LittleEndian.PutUint16(buf[lnEnd:lnEnd+2], 1)
	buf[lnEnd+2] = 0x00
	return buf
}

// makeFileItem creates an ItemID for a file with BEEF0004 extension block.
func makeFileItem(name string, fileSize uint32, t time.Time) []byte {
	longNameBytes := make([]byte, (len([]rune(name))+1)*2)
	putUTF16LE(longNameBytes, name)
	beeDataSize := 8 + 16 + 20 + len(longNameBytes) + 3
	itemDataSize := 12 + len(name) + 1 + 2 + beeDataSize
	buf := make([]byte, itemDataSize)
	buf[0] = 0x32
	binary.LittleEndian.PutUint32(buf[2:6], fileSize)
	binary.LittleEndian.PutUint16(buf[10:12], 0x0020)
	copy(buf[12:], []byte(name))
	buf[12+len(name)] = 0x00

	beeStart := 12 + len(name) + 1
	ft := timeToFileTime(t)
	binary.LittleEndian.PutUint16(buf[beeStart:beeStart+2], uint16(2+beeDataSize))
	bee := beeStart + 2
	binary.LittleEndian.PutUint16(buf[bee:bee+2], 9)
	binary.LittleEndian.PutUint32(buf[bee+4:bee+8], 0xBEEF0004)
	binary.LittleEndian.PutUint64(buf[bee+8:bee+16], ft)
	binary.LittleEndian.PutUint64(buf[bee+16:bee+24], ft)
	copy(buf[bee+44:], longNameBytes)
	lnEnd := bee + 44 + len(longNameBytes)
	binary.LittleEndian.PutUint16(buf[lnEnd:lnEnd+2], 1)
	buf[lnEnd+2] = 0x00
	return buf
}

// putUTF16LE encodes s as UTF-16LE with null terminator into buf.
func putUTF16LE(buf []byte, s string) {
	encoded := utf16.Encode([]rune(s + "\x00"))
	for i, r := range encoded {
		binary.LittleEndian.PutUint16(buf[i*2:], r)
	}
}

// IDListProxy is a helper to write the full LinkTargetIDList to a writer.
type IDListProxy struct {
	Target   string
	FileSize uint32
}

// WriteTo writes the serialized IDList to w.
func (p IDListProxy) WriteTo(w io.Writer) (int64, error) {
	data, err := BuildIDList(p.Target, p.FileSize)
	if err != nil {
		return 0, err
	}
	n, err := w.Write(data)
	return int64(n), err
}
