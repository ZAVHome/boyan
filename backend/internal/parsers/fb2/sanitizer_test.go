package fb2_test

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"boyan/internal/parsers/fb2"
)

func TestSanitizeFB2Bytes_BOMAndWhitespace(t *testing.T) {
	input := []byte("\xef\xbb\xbf   \r\n<?xml version=\"1.0\" encoding=\"utf-8\"?><FictionBook xmlns=\"http://www.gribuser.ru/xml/fictionbook/2.0\" xmlns:l=\"http://www.w3.org/1999/xlink\" xmlns:xlink=\"http://www.w3.org/1999/xlink\"><body><p>Hello</p></body></FictionBook>")
	sanitized, modified := fb2.SanitizeFB2Bytes(input)

	if !modified {
		t.Fatalf("expected modified=true")
	}

	if bytes.HasPrefix(sanitized, []byte("\xef\xbb\xbf")) {
		t.Errorf("BOM was not stripped")
	}

	if !bytes.HasPrefix(sanitized, []byte("<?xml")) {
		t.Errorf("Leading whitespace was not trimmed")
	}
}

func TestSanitizeFB2Bytes_MissingNamespaces(t *testing.T) {
	// Типичный случай как у Чехова: объявлен только xmlns:l, а xmlns:xlink отсутствует
	input := []byte(`<?xml version="1.0" encoding="windows-1251"?><FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink"><description><title-info><coverpage><image xlink:href="#_0.jpg"/></coverpage></title-info></description><body><p>Test</p></body></FictionBook>`)
	sanitized, modified := fb2.SanitizeFB2Bytes(input)

	if !modified {
		t.Fatalf("expected modified=true")
	}

	sanitizedStr := string(sanitized)
	if !bytes.Contains(sanitized, []byte("xmlns:xlink=\"http://www.w3.org/1999/xlink\"")) {
		t.Errorf("xmlns:xlink was not injected, got: %s", sanitizedStr)
	}
	if !bytes.Contains(sanitized, []byte("xmlns:l=\"http://www.w3.org/1999/xlink\"")) {
		t.Errorf("xmlns:l was lost, got: %s", sanitizedStr)
	}
}

func TestSanitizeFB2Bytes_HTMLEntities(t *testing.T) {
	input := []byte(`<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink" xmlns:xlink="http://www.w3.org/1999/xlink"><body><p>Слово&nbsp;&mdash;&nbsp;&laquo;дело&raquo;&hellip;</p></body></FictionBook>`)
	sanitized, modified := fb2.SanitizeFB2Bytes(input)

	if !modified {
		t.Fatalf("expected modified=true")
	}

	sanitizedStr := string(sanitized)
	if bytes.Contains(sanitized, []byte("&nbsp;")) {
		t.Errorf("&nbsp; was not replaced: %s", sanitizedStr)
	}
	if !bytes.Contains(sanitized, []byte("&#160;&#8212;&#160;&#171;дело&#187;&#8230;")) {
		t.Errorf("unexpected entity replacement: %s", sanitizedStr)
	}
}

func TestSanitizeFB2File(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "sanitizer_test_*")
	if err != nil {
		t.Fatalf("temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "book.fb2")
	brokenContent := []byte("\xef\xbb\xbf<FictionBook xmlns=\"http://www.gribuser.ru/xml/fictionbook/2.0\"><body><p>A&nbsp;B</p></body></FictionBook>")
	if err := os.WriteFile(filePath, brokenContent, 0644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	modified, err := fb2.SanitizeFB2File(filePath)
	if err != nil {
		t.Fatalf("SanitizeFB2File failed: %v", err)
	}
	if !modified {
		t.Fatalf("expected modified=true")
	}

	updated, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}

	if bytes.HasPrefix(updated, []byte("\xef\xbb\xbf")) {
		t.Errorf("BOM was not removed from file")
	}
	if !bytes.Contains(updated, []byte("xmlns:xlink")) {
		t.Errorf("xmlns:xlink was not added to file")
	}
	if bytes.Contains(updated, []byte("&nbsp;")) {
		t.Errorf("&nbsp; was not replaced in file")
	}
}
