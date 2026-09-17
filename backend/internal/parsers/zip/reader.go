package zip

import (
	"archive/zip"
	"fmt"
	"io"
	"path/filepath"
	"strings"
)

// FB2ZipEntry представляет найденный файл FB2 внутри ZIP-контейнера.
type FB2ZipEntry struct {
	Name     string
	Size     int64
	OpenFunc func() (io.ReadCloser, error)
}

// FindFB2InZip открывает ZIP-архив и находит внутри него файл .fb2 без распаковки на диск.
func FindFB2InZip(r io.ReaderAt, size int64) (*FB2ZipEntry, error) {
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("open zip archive: %w", err)
	}

	for _, file := range zr.File {
		// Защита от Zip Slip уязвимости
		cleanName := filepath.Clean(file.Name)
		if strings.HasPrefix(cleanName, "..") || strings.HasPrefix(cleanName, "/") || strings.HasPrefix(cleanName, "\\") {
			continue
		}

		lower := strings.ToLower(file.Name)
		if strings.HasSuffix(lower, ".fb2") && !file.FileInfo().IsDir() {
			return &FB2ZipEntry{
				Name: file.Name,
				Size: int64(file.UncompressedSize64),
				OpenFunc: func() (io.ReadCloser, error) {
					return file.Open()
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("no .fb2 file found inside zip archive")
}

// OpenFB2FromZipFile открывает zip-архив по пути на диске и возвращает ридер для внутреннего fb2.
func OpenFB2FromZipFile(zipPath string) (*zip.ReadCloser, *zip.File, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open zip file %s: %w", zipPath, err)
	}

	for _, file := range zr.File {
		cleanName := filepath.Clean(file.Name)
		if strings.HasPrefix(cleanName, "..") || strings.HasPrefix(cleanName, "/") || strings.HasPrefix(cleanName, "\\") {
			continue
		}

		lower := strings.ToLower(file.Name)
		if strings.HasSuffix(lower, ".fb2") && !file.FileInfo().IsDir() {
			return zr, file, nil
		}
	}

	zr.Close()
	return nil, nil, fmt.Errorf("no .fb2 file found inside %s", zipPath)
}

// ExtractFB2Stream открывает стрим FB2 из файла zip.
func ExtractFB2Stream(zipPath string) (io.ReadCloser, error) {
	zr, file, err := OpenFB2FromZipFile(zipPath)
	if err != nil {
		return nil, err
	}

	rc, err := file.Open()
	if err != nil {
		zr.Close()
		return nil, fmt.Errorf("open inner zip file: %w", err)
	}

	// Оборачиваем, чтобы при закрытии стрима закрывался и сам ZIP-контейнер
	return &zipStreamCloser{ReadCloser: rc, zipCloser: zr}, nil
}

type zipStreamCloser struct {
	io.ReadCloser
	zipCloser io.Closer
}

func (z *zipStreamCloser) Close() error {
	err1 := z.ReadCloser.Close()
	err2 := z.zipCloser.Close()
	if err1 != nil {
		return err1
	}
	return err2
}
