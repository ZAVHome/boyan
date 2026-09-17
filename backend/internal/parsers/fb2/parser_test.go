package fb2_test

import (
	"bytes"
	"encoding/base64"
	"strings"
	"testing"

	"boyan/internal/parsers/fb2"

	"golang.org/x/text/encoding/charmap"
)

const sampleCoverBase64 = "/9j/4AAQSkZJRgABAQEASABIAAD/2wBDAP//////////////////////////////////////////////////////////////////////////////////////wgALCAABAAEBAREA/8QAFBABAAAAAAAAAAAAAAAAAAAAAP/aAAgBAQABPxA="

func TestParseFB2_UTF8(t *testing.T) {
	xmlContent := `<?xml version="1.0" encoding="utf-8"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0" xmlns:l="http://www.w3.org/1999/xlink">
  <description>
    <title-info>
      <genre>sf_history</genre>
      <author>
        <first-name>Сергей</first-name>
        <middle-name>Васильевич</middle-name>
        <last-name>Лукьяненко</last-name>
      </author>
      <book-title>Черновик</book-title>
      <annotation>
        <p>Увлекательный роман о параллельных мирах.</p>
      </annotation>
      <date value="2005">2005</date>
      <coverpage>
        <image l:href="#cover.jpg"/>
      </coverpage>
      <lang>ru</lang>
      <sequence name="Работа над ошибками" number="1"/>
    </title-info>
    <publish-info>
      <publisher>АСТ</publisher>
      <year>2005</year>
      <isbn>5-17-026135-8</isbn>
    </publish-info>
  </description>
  <body>
    <section><p>Текст книги...</p></section>
  </body>
  <binary id="cover.jpg" content-type="image/jpeg">` + sampleCoverBase64 + `</binary>
</FictionBook>`

	info, err := fb2.ParseFB2(strings.NewReader(xmlContent))
	if err != nil {
		t.Fatalf("ParseFB2 failed: %v", err)
	}

	if info.Book.Title != "Черновик" {
		t.Errorf("expected title 'Черновик', got '%s'", info.Book.Title)
	}
	if info.Book.Publisher != "АСТ" {
		t.Errorf("expected publisher 'АСТ', got '%s'", info.Book.Publisher)
	}
	if info.Book.ISBN != "5-17-026135-8" {
		t.Errorf("expected isbn '5-17-026135-8', got '%s'", info.Book.ISBN)
	}
	if len(info.Authors) != 1 || info.Authors[0].Name != "Сергей Васильевич Лукьяненко" {
		t.Errorf("unexpected author: %+v", info.Authors)
	}
	if len(info.Series) != 1 || info.Series[0].Name != "Работа над ошибками" || info.Series[0].Index != 1.0 {
		t.Errorf("unexpected series: %+v", info.Series)
	}
	if len(info.Genres) != 1 || info.Genres[0].NameRU != "Альтернативная история" {
		t.Errorf("unexpected genre: %+v", info.Genres)
	}
	if len(info.CoverBytes) == 0 {
		t.Errorf("expected cover bytes to be extracted")
	}
}

func TestParseFB2_Windows1251(t *testing.T) {
	utf8XML := `<?xml version="1.0" encoding="windows-1251"?>
<FictionBook xmlns="http://www.gribuser.ru/xml/fictionbook/2.0">
  <description>
    <title-info>
      <genre>popadanec</genre>
      <author>
        <first-name>Иван</first-name>
        <last-name>Иванов</last-name>
      </author>
      <book-title>Путешествие во времени</book-title>
      <lang>ru</lang>
    </title-info>
  </description>
</FictionBook>`

	// Кодируем в реальный Windows-1251
	encoder := charmap.Windows1251.NewEncoder()
	cp1251Bytes, err := encoder.Bytes([]byte(utf8XML))
	if err != nil {
		t.Fatalf("encode to cp1251 failed: %v", err)
	}

	info, err := fb2.ParseFB2(bytes.NewReader(cp1251Bytes))
	if err != nil {
		t.Fatalf("ParseFB2 Windows-1251 failed: %v", err)
	}

	if info.Book.Title != "Путешествие во времени" {
		t.Errorf("expected title 'Путешествие во времени', got '%s'", info.Book.Title)
	}
	if len(info.Authors) != 1 || info.Authors[0].Name != "Иван Иванов" {
		t.Errorf("expected author 'Иван Иванов', got %+v", info.Authors)
	}
	if len(info.Genres) != 1 || info.Genres[0].NameRU != "Попаданцы" {
		t.Errorf("expected genre 'Попаданцы', got %+v", info.Genres)
	}
}

func TestBase64Decode(t *testing.T) {
	decoded, err := base64.StdEncoding.DecodeString(sampleCoverBase64)
	if err != nil {
		t.Fatalf("decode sample base64: %v", err)
	}
	if len(decoded) == 0 {
		t.Fatalf("empty decoded bytes")
	}
}
