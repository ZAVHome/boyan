package cover_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"testing"
	"time"

	"boyan/internal/parsers/cover"
)

func createTestImage(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 100, A: 255})
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func TestProcessCover(t *testing.T) {
	// Создаем тестовое изображение 1000x800
	raw := createTestImage(1000, 800)

	// Масштабируем до 400px
	resized, err := cover.ProcessCover(raw, 400)
	if err != nil {
		t.Fatalf("ProcessCover failed: %v", err)
	}

	if len(resized) == 0 {
		t.Fatalf("empty resized cover output")
	}

	// Декодируем результат и проверяем размеры
	img, _, err := image.Decode(bytes.NewReader(resized))
	if err != nil {
		t.Fatalf("decode resized image: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 400 {
		t.Errorf("expected width 400, got %d", bounds.Dx())
	}
	if bounds.Dy() != 320 {
		t.Errorf("expected height 320, got %d", bounds.Dy())
	}
}

func TestCoverCache_LRU(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "cover_cache_test_*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Создаем кеш с лимитом 1 МБ
	cache, err := cover.NewCoverCache(tmpDir, 1)
	if err != nil {
		t.Fatalf("NewCoverCache failed: %v", err)
	}

	// 1. Сохраняем обложку
	data := createTestImage(100, 100)
	path, err := cache.SaveCover("book-1", data)
	if err != nil {
		t.Fatalf("SaveCover failed: %v", err)
	}

	if _, ok := cache.GetCoverPath("book-1"); !ok {
		t.Errorf("expected book-1 to be found in cache at %s", path)
	}

	// 2. Проверяем LRU-очистку
	// Создаем много данных, превышающих 1 МБ
	bigData := make([]byte, 300*1024) // 300 КБ каждый файл
	_, _ = cache.SaveCover("old-1", bigData)
	time.Sleep(10 * time.Millisecond)
	_, _ = cache.SaveCover("old-2", bigData)
	time.Sleep(10 * time.Millisecond)
	_, _ = cache.SaveCover("old-3", bigData)
	time.Sleep(10 * time.Millisecond)
	_, _ = cache.SaveCover("old-4", bigData)
	time.Sleep(10 * time.Millisecond)
	_, _ = cache.SaveCover("new-5", bigData)

	// Запускаем принудительную очистку
	cache.PruneLRU()

	// Проверяем, что каталог не пуст и файлы старые удаляются при переполнении
	entries, _ := os.ReadDir(tmpDir)
	var totalBytes int64
	for _, e := range entries {
		info, _ := e.Info()
		totalBytes += info.Size()
	}

	if totalBytes > 1024*1024 {
		t.Errorf("total cache size %d exceeded 1MB quota", totalBytes)
	}
}
