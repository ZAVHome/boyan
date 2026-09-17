package cover

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"boyan/internal/parsers/fb2"
	internalZip "boyan/internal/parsers/zip"
)

var (
	reItemCoverImage = regexp.MustCompile(`(?i)<item[^>]+(?:properties="[^"]*cover-image[^"]*"[^>]+href="([^"]+)"|href="([^"]+)"[^>]+properties="[^"]*cover-image)`)
	reItemCoverID    = regexp.MustCompile(`(?i)<item[^>]+(?:id="cover[^"]*"[^>]+href="([^"]+)"|href="([^"]+)"[^>]+id="cover)`)
	reMetaCover      = regexp.MustCompile(`(?i)<meta[^>]+name="cover"[^>]+content="([^"]+)"`)
)

// ExtractRawCoverFromFile пытается извлечь необработанные байты обложки из файла книги (FB2, FB2.ZIP, EPUB).
func ExtractRawCoverFromFile(filePath string, format string) ([]byte, error) {
	fmtLower := strings.ToLower(format)

	switch {
	case fmtLower == "fb2":
		f, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("open fb2: %w", err)
		}
		defer f.Close()

		info, err := fb2.ParseFB2(f)
		if err != nil {
			return nil, fmt.Errorf("parse fb2: %w", err)
		}
		if len(info.CoverBytes) == 0 {
			return nil, fmt.Errorf("no embedded cover in fb2")
		}
		return info.CoverBytes, nil

	case fmtLower == "fb2.zip" || fmtLower == "zip":
		rc, err := internalZip.ExtractFB2Stream(filePath)
		if err != nil {
			return nil, fmt.Errorf("open fb2.zip: %w", err)
		}
		defer rc.Close()

		info, err := fb2.ParseFB2(rc)
		if err != nil {
			return nil, fmt.Errorf("parse fb2 in zip: %w", err)
		}
		if len(info.CoverBytes) == 0 {
			return nil, fmt.Errorf("no embedded cover in fb2.zip")
		}
		return info.CoverBytes, nil

	case fmtLower == "epub":
		return extractRawCoverFromEPUB(filePath)

	default:
		return nil, fmt.Errorf("unsupported format %s for cover extraction", format)
	}
}

func extractRawCoverFromEPUB(filePath string) ([]byte, error) {
	zr, err := zip.OpenReader(filePath)
	if err != nil {
		return nil, fmt.Errorf("open epub zip: %w", err)
	}
	defer zr.Close()

	// 1. Попробуем найти файл обложки по типичным именам
	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		base := strings.ToLower(filepath.Base(name))
		if base == "cover.jpg" || base == "cover.jpeg" || base == "cover.png" || base == "cover.webp" {
			return readZipEntry(f)
		}
	}

	// 2. Ищем content.opf и парсим ссылку на обложку
	for _, f := range zr.File {
		if strings.HasSuffix(strings.ToLower(f.Name), ".opf") {
			rc, err := f.Open()
			if err != nil {
				continue
			}
			data, _ := io.ReadAll(rc)
			rc.Close()

			coverHref := findCoverHrefInOPF(string(data))
			if coverHref != "" {
				opfDir := path.Dir(f.Name)
				targetPath := coverHref
				if opfDir != "." && opfDir != "" {
					targetPath = path.Join(opfDir, coverHref)
				}
				targetPath = strings.TrimPrefix(targetPath, "/")

				for _, cf := range zr.File {
					if strings.EqualFold(cf.Name, targetPath) || strings.HasSuffix(strings.ToLower(cf.Name), strings.ToLower(coverHref)) {
						return readZipEntry(cf)
					}
				}
			}
		}
	}

	// 3. Эвристика: любое изображение со словом cover в названии
	for _, f := range zr.File {
		name := strings.ToLower(f.Name)
		if strings.Contains(name, "cover") && (strings.HasSuffix(name, ".jpg") || strings.HasSuffix(name, ".jpeg") || strings.HasSuffix(name, ".png")) {
			return readZipEntry(f)
		}
	}

	return nil, fmt.Errorf("no cover found in epub")
}

func findCoverHrefInOPF(opfContent string) string {
	// properties="cover-image"
	if m := reItemCoverImage.FindStringSubmatch(opfContent); len(m) > 1 {
		if m[1] != "" {
			return m[1]
		}
		if len(m) > 2 && m[2] != "" {
			return m[2]
		}
	}

	// <meta name="cover" content="ID"/>
	if m := reMetaCover.FindStringSubmatch(opfContent); len(m) > 1 && m[1] != "" {
		coverID := regexp.QuoteMeta(m[1])
		reTargetItem := regexp.MustCompile(fmt.Sprintf(`(?i)<item[^>]+id="%s"[^>]+href="([^"]+)"`, coverID))
		if im := reTargetItem.FindStringSubmatch(opfContent); len(im) > 1 && im[1] != "" {
			return im[1]
		}
		reTargetItemReverse := regexp.MustCompile(fmt.Sprintf(`(?i)<item[^>]+href="([^"]+)"[^>]+id="%s"`, coverID))
		if im := reTargetItemReverse.FindStringSubmatch(opfContent); len(im) > 1 && im[1] != "" {
			return im[1]
		}
	}

	// id="cover"
	if m := reItemCoverID.FindStringSubmatch(opfContent); len(m) > 1 {
		if m[1] != "" {
			return m[1]
		}
		if len(m) > 2 && m[2] != "" {
			return m[2]
		}
	}

	return ""
}

func readZipEntry(f *zip.File) ([]byte, error) {
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	return io.ReadAll(rc)
}
