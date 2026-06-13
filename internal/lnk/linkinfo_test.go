package lnk

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestLinkInfo_MarshalBinary(t *testing.T) {
	li := LinkInfo{
		Target: `C:\Program Files\MyApp\app.exe`,
	}
	got, err := li.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	if len(got) < 31 {
		t.Errorf("LinkInfo too short: %d bytes, want >= 31", len(got))
	}

	// LinkInfo header:
	// uint32 LinkInfoSize (total including this)
	// uint32 LinkInfoHeaderSize (28)
	// uint32 LinkInfoFlags (0 = VolumeIDAndLocalBasePath)
	// uint32 VolumeIDAndLocalBasePath (offset to local path)

	linkInfoSize := binary.LittleEndian.Uint32(got[0:4])
	if linkInfoSize != uint32(len(got)) {
		t.Errorf("LinkInfoSize: got %d, want %d", linkInfoSize, len(got))
	}

	linkInfoHeaderSize := binary.LittleEndian.Uint32(got[4:8])
	if linkInfoHeaderSize != 28 {
		t.Errorf("LinkInfoHeaderSize: got %d, want 28", linkInfoHeaderSize)
	}

	// Check the local base path appears in ANSI encoding
	if !bytes.Contains(got, []byte("app.exe")) {
		t.Errorf("LinkInfo should contain ANSI encoded 'app.exe'")
	}
}

func TestLinkInfo_DriveOnly(t *testing.T) {
	li := LinkInfo{
		Target: `D:\`,
	}
	got, err := li.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	linkInfoSize := binary.LittleEndian.Uint32(got[0:4])
	if int(linkInfoSize) != len(got) {
		t.Errorf("LinkInfoSize: got %d, want %d", linkInfoSize, len(got))
	}
}

func TestLinkInfo_EmptyTarget(t *testing.T) {
	li := LinkInfo{}
	_, err := li.MarshalBinary()
	if err == nil {
		t.Error("expected error for empty target")
	}
}

func TestLinkInfo_WriteTo(t *testing.T) {
	li := LinkInfo{
		Target: `C:\test\file.txt`,
	}
	var buf bytes.Buffer
	n, err := li.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}
	if n <= 0 {
		t.Fatalf("WriteTo() wrote %d bytes", n)
	}
}
