package lnk

import (
	"bytes"
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func TestStringData_MarshalBinary_Empty(t *testing.T) {
	sd := StringData{}
	got, err := sd.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	// With no strings set: 2 (NameCount=0) + 2 (RelPathCount=0) + 2 (WDCount=0) + 2 (ArgsCount=0) + 2 (IconCount=0) = 10 bytes
	if len(got) != 10 {
		t.Errorf("expected 10 bytes for empty StringData, got %d", len(got))
	}
	// All count fields should be 0
	for i := 0; i < len(got); i += 2 {
		count := binary.LittleEndian.Uint16(got[i:])
		if count != 0 {
			t.Errorf("at offset %d: expected count 0, got %d", i, count)
		}
	}
}

func TestStringData_MarshalBinary_WorkingDir(t *testing.T) {
	sd := StringData{
		HasWorkingDir: true,
		WorkingDir:    `C:\Program Files\MyApp`,
	}
	got, err := sd.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected non-empty output")
	}

	// NameCount=0 (offset 0)
	if binary.LittleEndian.Uint16(got[0:2]) != 0 {
		t.Errorf("expected NameCount=0")
	}
	// RelPathCount=0 (offset 2)
	if binary.LittleEndian.Uint16(got[2:4]) != 0 {
		t.Errorf("expected RelPathCount=0")
	}
	// WDCount (offset 4): "C:\Program Files\MyApp" = 21 chars + null = 22
	wdCount := binary.LittleEndian.Uint16(got[4:6])
	expectedCount := uint16(len([]rune(`C:\Program Files\MyApp`)) + 1)
	if wdCount != expectedCount {
		t.Errorf("expected WDCount=%d, got %d", expectedCount, wdCount)
	}
}

func TestStringData_MarshalBinary_All(t *testing.T) {
	sd := StringData{
		HasWorkingDir:   true,
		WorkingDir:      `C:\work`,
		HasArguments:    true,
		Arguments:       "-v",
		HasIconLocation: true,
		IconLocation:    `C:\app.exe,0`,
	}
	got, err := sd.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary() error = %v", err)
	}
	if len(got) == 0 {
		t.Fatal("expected non-empty output")
	}

	// Verify we can decode strings from the output
	// Structure: [NameCnt(2)][RelCnt(2)][WDCnt(2)][WDData...][ArgsCnt(2)][ArgsData...][IconCnt(2)][IconData...]
	offset := 0

	// NameCount = 0
	offset += 2
	// RelPathCount = 0
	offset += 2

	// WorkingDir
	wdCount := binary.LittleEndian.Uint16(got[offset:])
	offset += 2
	if wdCount > 0 {
		wdData := got[offset : offset+int(wdCount)*2]
		decoded := decodeUTF16Le(wdData)
		if decoded != `C:\work\x00` && decoded != `C:\work`+string(rune(0)) {
			// Just check it starts with the right text
			if len(decoded) < 7 || decoded[:7] != `C:\work` {
				t.Errorf("WorkingDir: got %q", decoded)
			}
		}
		offset += len(wdData)
	}

	// Arguments
	argsCount := binary.LittleEndian.Uint16(got[offset:])
	offset += 2
	if argsCount > 0 {
		argsData := got[offset : offset+int(argsCount)*2]
		_ = decodeUTF16Le(argsData)
		offset += len(argsData)
	}

	// IconLocation
	iconCount := binary.LittleEndian.Uint16(got[offset:])
	offset += 2
	if iconCount > 0 {
		iconData := got[offset : offset+int(iconCount)*2]
		_ = decodeUTF16Le(iconData)
	}
}

func decodeUTF16Le(data []byte) string {
	u16 := make([]uint16, len(data)/2)
	for i := range u16 {
		u16[i] = binary.LittleEndian.Uint16(data[i*2:])
	}
	runes := utf16.Decode(u16)
	return string(runes)
}

func TestEncodeUTF16Le(t *testing.T) {
	count, data := encodeUTF16Le("test")
	// "test" + null = 5 characters
	if count != 5 {
		t.Errorf("expected count=5, got %d", count)
	}
	// 5 chars * 2 bytes = 10 bytes
	if len(data) != 10 {
		t.Errorf("expected 10 bytes, got %d", len(data))
	}
	// Verify the encoded text
	decoded := decodeUTF16Le(data)
	if decoded != "test\x00" {
		t.Errorf("round-trip failed: got %q", decoded)
	}
}

func TestStringData_WriteTo(t *testing.T) {
	sd := StringData{
		HasWorkingDir: true,
		WorkingDir:    `C:\test`,
	}
	var buf bytes.Buffer
	n, err := sd.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}
	if n <= 0 {
		t.Fatalf("WriteTo() wrote %d bytes", n)
	}
}
