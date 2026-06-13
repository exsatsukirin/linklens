package lnk

import (
	"errors"
	"io"
	"time"
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

	header := ShellLinkHeader{
		LinkFlags:      linkFlags,
		FileAttributes: 0x00000020, // FILE_ATTRIBUTE_ARCHIVE
		CreationTime:   now,
		AccessTime:     now,
		WriteTime:      now,
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
	idList := IDListProxy{Target: l.Target}
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

	return totalWritten, nil
}
