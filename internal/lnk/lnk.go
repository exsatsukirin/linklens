package lnk

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// isASCII reports whether s consists solely of ASCII characters.
func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > 0x7f {
			return false
		}
	}
	return true
}

// LnkFile represents a Windows shortcut (.lnk) file being built.
type LnkFile struct {
	Target  string // Required: target executable or document
	WorkDir string // Optional: working directory
	Args    string // Optional: command-line arguments
	Icon    string // Optional: icon file path, optionally with a ",index" suffix
}

// splitIconLocation splits an "icon path,index" argument into the icon path and
// the icon index. Windows stores the path in the ICON_LOCATION string and the
// index in ShellLinkHeader.IconIndex (MS-SHLLINK 2.1.1); keeping the index in
// the path makes the Shell look for a file literally named "app.exe,0".
func splitIconLocation(icon string) (string, uint32) {
	i := strings.LastIndex(icon, ",")
	if i < 0 {
		return icon, 0
	}
	n, err := strconv.ParseUint(strings.TrimSpace(icon[i+1:]), 10, 32)
	if err != nil {
		return icon, 0
	}
	return strings.TrimSpace(icon[:i]), uint32(n)
}

// WriteTo serializes the complete .lnk file and writes it to w.
func (l *LnkFile) WriteTo(w io.Writer) (int64, error) {
	if l.Target == "" {
		return 0, errors.New("target path is required")
	}

	// Only ASCII is supported for now. The .lnk path fields are stored in ANSI
	// (system codepage) form, which cannot represent non-ASCII text portably,
	// so reject it up front instead of silently writing a shortcut that Windows
	// cannot resolve. Non-ASCII support may return once a target codepage can
	// be selected.
	for _, field := range []struct{ name, value string }{
		{"target", l.Target},
		{"workdir", l.WorkDir},
		{"args", l.Args},
		{"icon", l.Icon},
	} {
		if !isASCII(field.value) {
			return 0, fmt.Errorf("%s contains non-ASCII characters, which are not supported yet: %q", field.name, field.value)
		}
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

	// The icon index belongs in the header, not in the icon path string.
	iconPath, iconIndex := splitIconLocation(l.Icon)

	header := ShellLinkHeader{
		LinkFlags:      linkFlags,
		FileAttributes: 0x00000020, // FILE_ATTRIBUTE_ARCHIVE
		CreationTime:   now,
		AccessTime:     now,
		WriteTime:      now,
		FileSize:       fileSize,
		IconIndex:      iconIndex,
		ShowCommand:    1, // SW_SHOWNORMAL
		HotKey:         0,
	}

	stringData := StringData{
		HasWorkingDir:   l.WorkDir != "",
		WorkingDir:      l.WorkDir,
		HasArguments:    l.Args != "",
		Arguments:       l.Args,
		HasIconLocation: l.Icon != "",
		IconLocation:    iconPath,
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

	// ExtraData is optional. Only the mandatory 4-byte TerminalBlock is
	// written: the former PropertyStoreDataBlock was malformed (wrong store
	// version and a missing SerializedPropertyValue size field), and strict
	// parsers rejected the file because of it.
	terminalBlock := []byte{0x00, 0x00, 0x00, 0x00}
	n2, err := w.Write(terminalBlock)
	totalWritten += int64(n2)
	if err != nil {
		return totalWritten, err
	}

	return totalWritten, nil
}
