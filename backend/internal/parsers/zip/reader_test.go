package zip_test

import (
	"archive/zip"
	"bytes"
	"io"
	"testing"

	zipParser "boyan/internal/parsers/zip"
)

func TestFindFB2InZip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Добавляем файл .fb2 в архив
	w, err := zw.Create("folder/subfolder/mybook.fb2")
	if err != nil {
		t.Fatalf("create zip entry: %v", err)
	}
	sampleFB2 := "<FictionBook><description><title-info><book-title>Test In Zip</book-title></title-info></description></FictionBook>"
	_, _ = w.Write([]byte(sampleFB2))

	// Добавляем другой файл для проверки фильтрации
	wTxt, _ := zw.Create("readme.txt")
	_, _ = wTxt.Write([]byte("some readme text"))

	if err := zw.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}

	zipBytes := buf.Bytes()
	readerAt := bytes.NewReader(zipBytes)

	entry, err := zipParser.FindFB2InZip(readerAt, int64(len(zipBytes)))
	if err != nil {
		t.Fatalf("FindFB2InZip failed: %v", err)
	}

	if entry.Name != "folder/subfolder/mybook.fb2" {
		t.Errorf("expected entry name 'folder/subfolder/mybook.fb2', got '%s'", entry.Name)
	}

	rc, err := entry.OpenFunc()
	if err != nil {
		t.Fatalf("open entry stream: %v", err)
	}
	defer rc.Close()

	readBytes, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read stream: %v", err)
	}

	if string(readBytes) != sampleFB2 {
		t.Errorf("content mismatch: expected '%s', got '%s'", sampleFB2, string(readBytes))
	}
}
