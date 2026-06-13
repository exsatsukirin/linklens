package lnk

import (
	"bytes"
	"encoding/binary"
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
