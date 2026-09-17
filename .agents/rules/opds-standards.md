# OPDS Standards & E-Ink Compatibility Guidelines

## 1. OPDS v1.2 (Atom / XML) Implementation
- **Корневой тег:** `<feed xmlns="http://www.w3.org/2005/Atom" xmlns:dc="http://purl.org/dc/terms/" xmlns:opds="http://opds-spec.org/2010/catalog">`.
- **Серии и индексы:** Обязательно передавать порядковый номер книги в цикле через расширение `<calibre:series>` и `<calibre:series_index>` (поддерживается AlReaderX, KOReader, Moon+ Reader).
- **Обложки:** Каждая запись книги `<entry>` обязана иметь две ссылки на изображения:
  - `rel="http://opds-spec.org/image"` (полноразмерная обложка).
  - `rel="http://opds-spec.org/image/thumbnail"` (быстрое превью для E-Ink экрана).
- **MIME-типы форматов:**
  - FB2: `application/x-fictionbook+xml`
  - FB2.ZIP: `application/x-zip-compressed-fb2` или `application/x-fictionbook+xml` с потоковой распаковкой.
  - EPUB: `application/epub+zip`
  - MOBI: `application/x-mobipocket-ebook`
  - PDF: `application/pdf`
- **Кодировка:** Строго `UTF-8` без BOM. Текстовые спецсимволы в названиях и аннотациях (`&`, `<`, `>`, `"`) должны быть корректно экранированы как `&amp;`, `&lt;`, `&gt;`, `&quot;`.

## 2. OPDS v2.0 (JSON-LD) Implementation
- Content-Type: `application/opds+json`.
- Соответствие спецификации Readium OPDS 2.0.
- Навигация по ссылкам (`self`, `start`, `up`, `search`).

## 3. Streaming Direct from ZIP
- Если книга запрошена читалкой в формате FB2, а на диске хранится как `.fb2.zip`:
  - Сервер НЕ должен распаковывать архив на диск во временные файлы.
  - Сервер должен открыть zip-поток (`zip.Reader`), найти нужный файл внутри архива и стримить его напрямую в HTTP-ответ `w.Write()`.
