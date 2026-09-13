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

// TestLinkInfo_VolumeID locks in the MS-SHLLINK 2.3.1 VolumeID layout: every
// field before the label is 4 bytes wide. Encoding DriveType as only 2 bytes
// used to shift the serial number and label offset and corrupt the block.
func TestLinkInfo_VolumeID(t *testing.T) {
	data, err := LinkInfo{Target: `C:\Windows\notepad.exe`}.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	const volOff = 28 // VolumeID starts right after the 28-byte LinkInfo header
	if v := binary.LittleEndian.Uint32(data[volOff : volOff+4]); v != 21 {
		t.Errorf("VolumeIDSize = %d, want 21", v)
	}
	if v := binary.LittleEndian.Uint32(data[volOff+4 : volOff+8]); v != 3 {
		t.Errorf("DriveType = %d, want 3 (DRIVE_FIXED)", v)
	}
	if v := binary.LittleEndian.Uint32(data[volOff+12 : volOff+16]); v != 16 {
		t.Errorf("VolumeLabelOffset = %d, want 16", v)
	}
	if label := string(data[volOff+16 : volOff+21]); label != "Data\x00" {
		t.Errorf("VolumeLabel = %q, want %q", label, "Data\x00")
	}
	// LocalBasePathOffset must point exactly past the 21-byte VolumeID.
	if v := binary.LittleEndian.Uint32(data[16:20]); v != uint32(volOff+21) {
		t.Errorf("LocalBasePathOffset = %d, want %d", v, volOff+21)
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
