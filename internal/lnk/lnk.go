package lnk

import (
	"encoding/binary"
	"errors"
	"io"
	"os"
	"time"
	"unicode/utf16"
)

// LnkFile represents a Windows shortcut (.lnk) file being built.
type LnkFile struct {
	Target  string // Required: target executable or document
	WorkDir string // Optional: working directory
	Args    string // Optional: command-line arguments
	Icon    string // Optional: icon file path
}

// WriteTo serializes the complete .lnk file and writes it to w.
func (l *LnkFile) WriteTo(w io.Writer) (int64, error) {
	if l.Target == "" {
		return 0, errors.New("target path is required")
	}

	// Build LinkFlags
	linkFlags := uint32(0)
	linkFlags |= 0x00000001 // HasLinkTargetIDList
	linkFlags |= 0x00000002 // HasLinkInfo
	linkFlags |= 0x00000080 // IsUnicode
	if l.WorkDir != "" {
		linkFlags |= 0x00000010 // HasWorkingDir
	}
	if l.Args != "" {
		linkFlags |= 0x00000020 // HasArguments
	}
	if l.Icon != "" {
		linkFlags |= 0x00000040 // HasIconLocation
	}

	now := time.Now()

	// Get target file size for header and IDList
	var fileSize uint32
	if info, err := os.Stat(l.Target); err == nil {
		fileSize = uint32(info.Size())
	}

	header := ShellLinkHeader{
		LinkFlags:      linkFlags,
		FileAttributes: 0x00000020, // FILE_ATTRIBUTE_ARCHIVE
		CreationTime:   now,
		AccessTime:     now,
		WriteTime:      now,
		FileSize:       fileSize,
		IconIndex:      0,
		ShowCommand:    1, // SW_SHOWNORMAL
		HotKey:         0,
	}

	stringData := StringData{
		HasWorkingDir:   l.WorkDir != "",
		WorkingDir:      l.WorkDir,
		HasArguments:    l.Args != "",
		Arguments:       l.Args,
		HasIconLocation: l.Icon != "",
		IconLocation:    l.Icon,
	}

	// Write header
	totalWritten, err := header.WriteTo(w)
	if err != nil {
		return totalWritten, err
	}

	// Write IDList
	idList := IDListProxy{Target: l.Target, FileSize: fileSize}
	n, err := idList.WriteTo(w)
	totalWritten += n
	if err != nil {
		return totalWritten, err
	}

	// Write LinkInfo
	linkInfo := LinkInfo{Target: l.Target}
	n, err = linkInfo.WriteTo(w)
	totalWritten += n
	if err != nil {
		return totalWritten, err
	}

	// Write StringData
	n, err = stringData.WriteTo(w)
	totalWritten += n
	if err != nil {
		return totalWritten, err
	}

	// Write PropertyStoreDataBlock (ExtraData)
	psBlock := buildPropertyStoreDataBlock(l.Target)
	n2, err := w.Write(psBlock)
	totalWritten += int64(n2)
	if err != nil {
		return totalWritten, err
	}

	// Write TerminalBlock (ExtraData terminator, 4 bytes of 0x00000000)
	terminalBlock := []byte{0x00, 0x00, 0x00, 0x00}
	n3, err := w.Write(terminalBlock)
	totalWritten += int64(n3)
	if err != nil {
		return totalWritten, err
	}

	return totalWritten, nil
}

// buildPropertyStoreDataBlock creates an ExtraData PropertyStoreDataBlock
// containing the "Name" property (display name of the shortcut).
func buildPropertyStoreDataBlock(target string) []byte {
	// Extract display name from target path (filename without extension)
	displayName := target
	for i := len(target) - 1; i >= 0; i-- {
		if target[i] == '\\' || target[i] == '/' {
			displayName = target[i+1:]
			break
		}
	}
	// Remove extension
	for i := len(displayName) - 1; i >= 0; i-- {
		if displayName[i] == '.' {
			displayName = displayName[:i]
			break
		}
	}

	// Encode display name as UTF-16LE with null terminator
	encoded := utf16.Encode([]rune(displayName + "\x00"))
	nameData := make([]byte, len(encoded)*2)
	for i, r := range encoded {
		binary.LittleEndian.PutUint16(nameData[i*2:], r)
	}

	// PropertyStore = header(8) + FormatID(16) + Name property(4+4+len(nameData))
	propertyStoreSize := 8 + 16 + 4 + 4 + len(nameData)

	// PropertyStoreDataBlock = size(4) + signature(4) + PropertyStore
	totalSize := 4 + 4 + propertyStoreSize

	buf := make([]byte, totalSize)

	// ExtraData block header
	binary.LittleEndian.PutUint32(buf[0:4], uint32(totalSize))
	binary.LittleEndian.PutUint32(buf[4:8], 0xA0000009) // PropertyStoreDataBlock signature

	// PropertyStore header
	off := 8
	binary.LittleEndian.PutUint32(buf[off:off+4], uint32(propertyStoreSize))
	binary.LittleEndian.PutUint32(buf[off+4:off+8], 0x0534) // wVersion
	off += 8

	// FormatID: {D5CDD505-2E9C-101B-9397-08002B2CF9AE}
	copy(buf[off:off+16], []byte{
		0x05, 0xD5, 0xCD, 0xD5, 0x9C, 0x2E, 0x1B, 0x10,
		0x93, 0x97, 0x08, 0x00, 0x2B, 0x2C, 0xF9, 0xAE,
	})
	off += 16

	// Name property value: {B725F130-47EF-101A-A5F1-02608C9EEBAC}, PID=10
	// Type: VT_LPWSTR (0x001F) + data (UTF-16LE string)
	binary.LittleEndian.PutUint32(buf[off:off+4], 0x001F) // VT_LPWSTR
	copy(buf[off+4:off+4+len(nameData)], nameData)

	return buf
}
