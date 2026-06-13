package lnk

import (
	"bytes"
	"encoding/binary"
	"testing"
	"time"
)

func TestShellLinkHeader_MarshalBinary(t *testing.T) {
	h := ShellLinkHeader{
		LinkFlags:      0x000000BF,
		FileAttributes: 0x00000020,
		IconIndex:      0,
		ShowCommand:    1,
	}
	got, err := h.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	if len(got) != 76 {
		t.Fatalf("MarshalBinary() returned %d bytes, want 76", len(got))
	}
	// Check magic: first 4 bytes = 0x0000004C (LE)
	if got[0] != 0x4C || got[1] != 0x00 || got[2] != 0x00 || got[3] != 0x00 {
		t.Errorf("bad header size magic: got %x", got[:4])
	}
	// Check CLSID: 00021401-0000-0000-C000-000000000046 at offset 4
	expectedClsid := []byte{
		0x01, 0x14, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xC0, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x46,
	}
	if !bytes.Equal(got[4:20], expectedClsid) {
		t.Errorf("bad CLSID: got %x, want %x", got[4:20], expectedClsid)
	}
	// Check LinkFlags at offset 20
	expectedFlags := []byte{0xBF, 0x00, 0x00, 0x00}
	if !bytes.Equal(got[20:24], expectedFlags) {
		t.Errorf("bad LinkFlags: got %x, want %x", got[20:24], expectedFlags)
	}
}

func TestShellLinkHeader_Times(t *testing.T) {
	now := time.Date(2025, 6, 13, 12, 0, 0, 0, time.UTC)
	h := ShellLinkHeader{
		CreationTime: now,
		AccessTime:   now,
		WriteTime:    now,
	}
	got, err := h.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	// Read CreationTime as uint64 LE from offset 28
	ft := binary.LittleEndian.Uint64(got[28:36])
	if ft == 0 {
		t.Errorf("CreationTime should not be zero")
	}
	// Verify AccessTime and WriteTime are also non-zero
	at := binary.LittleEndian.Uint64(got[36:44])
	wt := binary.LittleEndian.Uint64(got[44:52])
	if at == 0 {
		t.Errorf("AccessTime should not be zero")
	}
	if wt == 0 {
		t.Errorf("WriteTime should not be zero")
	}
}

func TestShellLinkHeader_EmptyTime(t *testing.T) {
	h := ShellLinkHeader{}
	got, err := h.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	ft := binary.LittleEndian.Uint64(got[28:36])
	if ft != 0 {
		t.Errorf("expected zero CreationTime for zero time, got %d", ft)
	}
}

func TestShellLinkHeader_WriteTo(t *testing.T) {
	h := ShellLinkHeader{
		LinkFlags:   0x00000080,
		ShowCommand: 1,
	}
	var buf bytes.Buffer
	n, err := h.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}
	if n != 76 {
		t.Errorf("WriteTo() wrote %d bytes, want 76", n)
	}
	if buf.Len() != 76 {
		t.Errorf("buf.Len() = %d, want 76", buf.Len())
	}
}
