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

func TestLnkFile_NonASCII(t *testing.T) {
	// Full test with Chinese paths throughout
	lnk := LnkFile{
		Target:  `C:\用户\文档\项目\报告.docx`,
		WorkDir: `C:\用户\文档`,
		Args:    `--编码=UTF-8 --输入=你好世界`,
		Icon:    `C:\软件\app.exe,0`,
	}

	var buf bytes.Buffer
	n, err := lnk.WriteTo(&buf)
	if err != nil {
		t.Fatalf("WriteTo() error = %v", err)
	}
	if n <= 76 {
		t.Fatalf("output too short: %d bytes", n)
	}

	data := buf.Bytes()

	// Verify header
	if data[0] != 0x4C || data[1] != 0x00 || data[2] != 0x00 || data[3] != 0x00 {
		t.Errorf("bad header magic: %x", data[:4])
	}

	// Verify LinkFlags include all our flags
	linkFlags := binary.LittleEndian.Uint32(data[20:24])
	if linkFlags&0x00000001 == 0 {
		t.Error("HasLinkTargetIDList flag not set")
	}
	if linkFlags&0x00000002 == 0 {
		t.Error("HasLinkInfo flag not set")
	}
	if linkFlags&0x00000010 == 0 {
		t.Error("HasWorkingDir flag not set")
	}
	if linkFlags&0x00000020 == 0 {
		t.Error("HasArguments flag not set")
	}
	if linkFlags&0x00000040 == 0 {
		t.Error("HasIconLocation flag not set")
	}
	if linkFlags&0x00000080 == 0 {
		t.Error("IsUnicode flag not set")
	}

	// Verify Chinese characters in StringData section (after header + IDList + LinkInfo)
	// Check for the Chinese working dir in UTF-16LE
	// '用户' — contiguous in UTF-16LE stream
	chineseUser := []byte{
		// '用' U+7528
		0x28, 0x75,
		// '户' U+6237
		0x37, 0x62,
	}
	if !bytes.Contains(data, chineseUser) {
		t.Error("Output should contain Chinese chars '用户' in UTF-16LE")
	}
	// '文档' — contiguous in UTF-16LE stream
	chineseDoc := []byte{
		// '文' U+6587
		0x87, 0x65,
		// '档' U+6863
		0x63, 0x68,
	}
	if !bytes.Contains(data, chineseDoc) {
		t.Error("Output should contain Chinese chars '文档' in UTF-16LE")
	}
	// '报告' — from the target path
	chineseReport := []byte{
		// '报' U+62A5
		0xA5, 0x62,
		// '告' U+544A
		0x4A, 0x54,
	}
	if !bytes.Contains(data, chineseReport) {
		t.Error("Output should contain Chinese chars '报告' in UTF-16LE")
	}

	// Check for Chinese args
	chineseArgs := []byte{
		// '你' U+4F60
		0x60, 0x4F,
		// '好' U+597D
		0x7D, 0x59,
		// '世' U+4E16
		0x16, 0x4E,
		// '界' U+754C
		0x4C, 0x75,
	}
	if !bytes.Contains(data, chineseArgs) {
		t.Error("Output should contain Chinese args '你好世界' in UTF-16LE")
	}
}
