package fb2

import (
	"bytes"
	"os"
	"regexp"
)

var (
	fictionBookTagRegex = regexp.MustCompile(`(?i)<FictionBook([^>]*)>`)
)

// Named HTML entities mapped to numeric XML character references.
// Standard XML 1.0 only allows &lt;, &gt;, &amp;, &quot;, &apos;.
// Any other entity reference without a DTD causes strict XML parsers to abort.
var htmlEntities = map[string]string{
	"&nbsp;":   "&#160;",
	"&copy;":   "&#169;",
	"&reg;":    "&#174;",
	"&trade;":  "&#8482;",
	"&mdash;":  "&#8212;",
	"&ndash;":  "&#8211;",
	"&laquo;":  "&#171;",
	"&raquo;":  "&#187;",
	"&ldquo;":  "&#8220;",
	"&rdquo;":  "&#8221;",
	"&lsquo;":  "&#8216;",
	"&rsquo;":  "&#8217;",
	"&hellip;": "&#8230;",
	"&bull;":   "&#8226;",
	"&sect;":   "&#167;",
	"&para;":   "&#182;",
	"&euro;":   "&#8364;",
	"&pound;":  "&#163;",
	"&yen;":    "&#165;",
}

// SanitizeFB2Bytes анализирует срез байт FB2-файла и исправляет типичные дефекты разметки:
// 1. Удаляет начальный UTF-8 BOM (\xef\xbb\xbf) и ведущие пробелы.
// 2. Гарантирует объявление пространств имён xmlns, xmlns:xlink и xmlns:l на корневом теге <FictionBook>.
// 3. Заменяет нестандартные именованные HTML-сущности на числовые XML-коды.
// Возвращает исправленные данные и флаг modified (были ли внесены изменения).
func SanitizeFB2Bytes(data []byte) ([]byte, bool) {
	modified := false

	// 1. Очистка от UTF-8 BOM
	if bytes.HasPrefix(data, []byte("\xef\xbb\xbf")) {
		data = data[3:]
		modified = true
	}

	// Очистка от ведущих пустых пробелов/переносов строк перед <?xml
	trimmed := bytes.TrimLeft(data, " \t\r\n")
	if len(trimmed) != len(data) {
		data = trimmed
		modified = true
	}

	// 2. Проверка и нормализация корневого тега <FictionBook>
	loc := fictionBookTagRegex.FindIndex(data)
	if loc != nil {
		tagBytes := data[loc[0]:loc[1]]
		tagStr := string(tagBytes)

		needXlink := !bytes.Contains(tagBytes, []byte("xmlns:xlink"))
		needL := !bytes.Contains(tagBytes, []byte("xmlns:l"))
		needDefaultNs := !bytes.Contains(tagBytes, []byte("xmlns=")) && !bytes.Contains(tagBytes, []byte("xmlns ="))

		if needXlink || needL || needDefaultNs {
			newTag := fictionBookTagRegex.ReplaceAllStringFunc(tagStr, func(m string) string {
				attrs := m[len("<FictionBook") : len(m)-1]
				if needDefaultNs {
					attrs += ` xmlns="http://www.gribuser.ru/xml/fictionbook/2.0"`
				}
				if needXlink {
					attrs += ` xmlns:xlink="http://www.w3.org/1999/xlink"`
				}
				if needL {
					attrs += ` xmlns:l="http://www.w3.org/1999/xlink"`
				}
				return "<FictionBook" + attrs + ">"
			})

			var buf bytes.Buffer
			buf.Grow(len(data) + 120)
			buf.Write(data[:loc[0]])
			buf.WriteString(newTag)
			buf.Write(data[loc[1]:])
			data = buf.Bytes()
			modified = true
		}
	}

	// 3. Замена именованных HTML-сущностей
	for ent, num := range htmlEntities {
		entBytes := []byte(ent)
		if bytes.Contains(data, entBytes) {
			data = bytes.ReplaceAll(data, entBytes, []byte(num))
			modified = true
		}
	}

	return data, modified
}

// SanitizeFB2File считывает файл с диска, нормализует его и, при наличии изменений,
// безопасно перезаписывает его через временный файл.
func SanitizeFB2File(filePath string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	sanitized, modified := SanitizeFB2Bytes(data)
	if !modified {
		return false, nil
	}

	tmpPath := filePath + ".tmp_repair"
	if err := os.WriteFile(tmpPath, sanitized, 0644); err != nil {
		return false, err
	}

	if err := os.Rename(tmpPath, filePath); err != nil {
		_ = os.Remove(tmpPath)
		return false, err
	}

	return true, nil
}
