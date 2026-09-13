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
	// No field is flagged, so MS-SHLLINK 2.4 requires the StringData sequence to
	// contain no bytes at all. Writing zero-count placeholders here would shift
	// every following field and corrupt the shortcut.
	if len(got) != 0 {
		t.Errorf("expected 0 bytes for empty StringData, got %d (% x)", len(got), got)
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

	// WorkingDir is the first present field (Name/RelativePath are absent and
	// omitted), so its count sits at offset 0.
	wdCount := binary.LittleEndian.Uint16(got[0:2])
	expectedCount := uint16(len([]rune(`C:\Program Files\MyApp`)) + 1)
	if wdCount != expectedCount {
		t.Errorf("expected WDCount=%d, got %d", expectedCount, wdCount)
	}
	if want := 2 + int(expectedCount)*2; len(got) != want {
		t.Errorf("expected %d bytes, got %d", want, len(got))
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

	// Verify we can decode strings from the output. Only flagged fields are
	// present, emitted in MS-SHLLINK order:
	// [WDCnt(2)][WDData...][ArgsCnt(2)][ArgsData...][IconCnt(2)][IconData...]
	offset := 0

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
