package lnk

import (
	"bytes"
	"encoding/binary"
	"strings"
	"testing"
)

func TestLnkFile_Write(t *testing.T) {
	lnk := LnkFile{
		Target:  `C:\Program Files\MyApp\app.exe`,
		WorkDir: `C:\Program Files\MyApp`,
		Args:    "--verbose",
		Icon:    `C:\Program Files\MyApp\app.exe,0`,
	}

	var buf bytes.Buffer
	n, err := lnk.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}
	if n <= 0 {
		t.Fatalf("WriteTo() wrote %d bytes", n)
	}
	if buf.Len() <= 76 {
		t.Fatalf("output too short: %d bytes, expected > 76", buf.Len())
	}

	// Verify it starts with the shell link magic
	data := buf.Bytes()
	if data[0] != 0x4C || data[1] != 0x00 || data[2] != 0x00 || data[3] != 0x00 {
		t.Errorf("data doesn't start with header size magic: %x", data[:4])
	}

	// Verify CLSID at offset 4
	expectedClsid := []byte{
		0x01, 0x14, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46,
	}
	if !bytes.Equal(data[4:20], expectedClsid) {
		t.Errorf("bad CLSID: got %x", data[4:20])
	}

	// Verify LinkFlags include our expected flags
	linkFlags := binary.LittleEndian.Uint32(data[20:24])
	if linkFlags&0x00000001 == 0 {
		t.Error("HasLinkTargetIDList flag not set")
	}
	if linkFlags&0x00000002 == 0 {
		t.Error("HasLinkInfo flag not set")
	}
	if linkFlags&0x00000080 == 0 {
		t.Error("IsUnicode flag not set")
	}
}

func TestLnkFile_WriteMinimal(t *testing.T) {
	lnk := LnkFile{
		Target: `C:\file.txt`,
	}

	var buf bytes.Buffer
	n, err := lnk.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}
	if n <= 76 {
		t.Fatalf("output too short: %d bytes", n)
	}
}

func TestLnkFile_WriteEmptyTarget(t *testing.T) {
	lnk := LnkFile{}
	var buf bytes.Buffer
	_, err := lnk.WriteTo(&buf)
	if err == nil {
		t.Error("expected error for empty target")
	}
}

// TestLnkFile_RejectsNonASCII documents the current limitation: non-ASCII input
// is refused up front rather than silently producing a shortcut whose ANSI path
// fields cannot be resolved by Windows.
func TestLnkFile_RejectsNonASCII(t *testing.T) {
	tests := []struct {
		name string
		lnk  LnkFile
	}{
		{"target", LnkFile{Target: `C:\用户\文档\报告.docx`}},
		{"workdir", LnkFile{Target: `C:\app.exe`, WorkDir: `C:\用户`}},
		{"args", LnkFile{Target: `C:\app.exe`, Args: `--输入=你好`}},
		{"icon", LnkFile{Target: `C:\app.exe`, Icon: `C:\软件\app.exe,0`}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if _, err := tt.lnk.WriteTo(&buf); err == nil {
				t.Errorf("WriteTo() with non-ASCII %s: expected error, got nil", tt.name)
			} else if !strings.Contains(err.Error(), "non-ASCII") {
				t.Errorf("error should mention non-ASCII, got: %v", err)
			}
			if buf.Len() != 0 {
				t.Errorf("no bytes should be written on rejection, got %d", buf.Len())
			}
		})
	}
}

// TestLnkFile_ASCIIStillWorks guards against the ASCII restriction rejecting
// valid input.
func TestLnkFile_ASCIIStillWorks(t *testing.T) {
	lnk := LnkFile{
		Target:  `C:\Program Files\MyApp\app.exe`,
		WorkDir: `C:\Program Files\MyApp`,
		Args:    `--verbose --out="C:\tmp\out.txt"`,
		Icon:    `C:\Program Files\MyApp\app.exe,0`,
	}
	var buf bytes.Buffer
	if _, err := lnk.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}
}

func TestSplitIconLocation(t *testing.T) {
	cases := []struct {
		in    string
		path  string
		index uint32
	}{
		{``, ``, 0},
		{`C:\app.exe`, `C:\app.exe`, 0},
		{`C:\app.exe,0`, `C:\app.exe`, 0},
		{`C:\app.exe,5`, `C:\app.exe`, 5},
		{`C:\icons, archive\app.ico`, `C:\icons, archive\app.ico`, 0},
	}
	for _, c := range cases {
		path, index := splitIconLocation(c.in)
		if path != c.path || index != c.index {
			t.Errorf("splitIconLocation(%q) = (%q, %d), want (%q, %d)", c.in, path, index, c.path, c.index)
		}
	}
}

// TestLnkFile_IconIndexInHeader verifies that a ",index" suffix is moved to the
// ShellLinkHeader.IconIndex field instead of being stored in the ICON_LOCATION
// path, matching what Windows itself writes.
func TestLnkFile_IconIndexInHeader(t *testing.T) {
	lnk := LnkFile{Target: `C:\Windows\notepad.exe`, Icon: `C:\Windows\notepad.exe,5`}
	var buf bytes.Buffer
	if _, err := lnk.WriteTo(&buf); err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}
	data := buf.Bytes()
	if got := binary.LittleEndian.Uint32(data[56:60]); got != 5 {
		t.Errorf("header IconIndex = %d, want 5", got)
	}
	// The ICON_LOCATION must end with ".exe" and carry no subscript digit.
	iconWithIndex := []byte{'.', 0x00, 'e', 0x00, 'x', 0x00, 'e', 0x00, ',', 0x00, '5', 0x00}
	if bytes.Contains(data, iconWithIndex) {
		t.Error("icon index leaked into the ICON_LOCATION StringData")
	}
}
