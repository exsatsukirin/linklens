package lnk

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestBuildIDList(t *testing.T) {
	tests := []struct {
		name     string
		target   string
		wantErr  bool
		minBytes int
	}{
		{
			name:     "simple exe",
			target:   `C:\Program Files\MyApp\app.exe`,
			wantErr:  false,
			minBytes: 10,
		},
		{
			name:     "root file",
			target:   `C:\file.txt`,
			wantErr:  false,
			minBytes: 10,
		},
		{
			name:     "deep path",
			target:   `D:\a\b\c\d\e\f\file.doc`,
			wantErr:  false,
			minBytes: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildIDList(tt.target)
			if (err != nil) != tt.wantErr {
				t.Fatalf("BuildIDList() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if len(got) < tt.minBytes {
				t.Errorf("BuildIDList() returned %d bytes, want >= %d", len(got), tt.minBytes)
			}
			// IDList starts with uint16 itemIDListSize (total size minus this field)
			if len(got) < 2 {
				t.Fatal("too short for size field")
			}
			size := int(binary.LittleEndian.Uint16(got[0:2]))
			if size != len(got)-2 {
				t.Errorf("declared size %d != actual data size %d", size, len(got)-2)
			}
			// Last 2 bytes should be terminal ID (0x0000)
			terminal := binary.LittleEndian.Uint16(got[len(got)-2:])
			if terminal != 0 {
				t.Errorf("expected terminal 0x0000, got 0x%04x", terminal)
			}
		})
	}
}

func TestBuildIDList_Empty(t *testing.T) {
	_, err := BuildIDList("")
	if err == nil {
		t.Error("expected error for empty target")
	}
}

func TestBuildIDList_NonASCII(t *testing.T) {
	// Chinese file name in path
	got, err := BuildIDList(`C:\用户\文档\报告.docx`)
	if err != nil {
		t.Fatalf("BuildIDList() error = %v", err)
	}
	if len(got) < 10 {
		t.Fatalf("output too short: %d bytes", len(got))
	}

	// Verify size field
	size := int(binary.LittleEndian.Uint16(got[0:2]))
	if size != len(got)-2 {
		t.Errorf("declared size %d != actual data size %d", size, len(got)-2)
	}

	// Verify terminal ID
	terminal := binary.LittleEndian.Uint16(got[len(got)-2:])
	if terminal != 0 {
		t.Errorf("expected terminal 0x0000, got 0x%04x", terminal)
	}

	// The ITEMIDLIST entries should contain the Chinese characters as UTF-16LE
	// We can't easily parse the structure without a full ITEMIDLIST parser,
	// but we can verify the raw UTF-16LE encoding of the file name appears
	chineseBytes := []byte{
		// '报' in UTF-16LE = U+62A5
		0xA5, 0x62,
		// '告' in UTF-16LE = U+544A
		0x4A, 0x54,
		// '.'
		0x2E, 0x00,
		// 'd'
		0x64, 0x00,
	}
	if !bytes.Contains(got, chineseBytes) {
		t.Errorf("IDList should contain Chinese filename '报告.docx' in UTF-16LE")
	}
}

func TestBuildIDList_RelativePath(t *testing.T) {
	_, err := BuildIDList(`relative\path\file.txt`)
	if err == nil {
		t.Error("expected error for relative path without drive")
	}
}
