package calibre

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"html"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"boyan/internal/config"
	"boyan/internal/models"
	"boyan/internal/parsers/cover"
	"boyan/internal/parsers/fb2"
	"boyan/internal/storage"

	"github.com/google/uuid"
)

type ImportOptions struct {
	CopyFiles bool `json:"copy_files"`
}

type ImportStats struct {
	TotalCalibreBooks int      `json:"total_calibre_books"`
	ImportedBooks     int      `json:"imported_books"`
	FormatsAttached   int      `json:"formats_attached"`
	Skipped           int      `json:"skipped"`
	Errors            []string `json:"errors,omitempty"`
}

type Importer struct {
	cfg        *config.Config
	bookRepo   *storage.BookRepository
	coverCache *cover.CoverCache
}

func NewImporter(cfg *config.Config, bookRepo *storage.BookRepository, coverCache *cover.CoverCache) *Importer {
	return &Importer{
		cfg:        cfg,
		bookRepo:   bookRepo,
		coverCache: coverCache,
	}
}

// ProgressFunc вызывается при обработке каждой книги из каталога Calibre.
type ProgressFunc func(processed, total int, currentItem string, err error)

// ImportLibrary выполняет импорт каталога Calibre в хранилище Бояна.
func (imp *Importer) ImportLibrary(ctx context.Context, calibreDir string, opts ImportOptions) (*ImportStats, error) {
	return imp.ImportLibraryWithProgress(ctx, calibreDir, opts, nil)
}

// ImportLibraryWithProgress выполняет импорт каталога Calibre с вызовом функции уведомления о прогрессе.
func (imp *Importer) ImportLibraryWithProgress(
	ctx context.Context,
	calibreDir string,
	opts ImportOptions,
	onProgress ProgressFunc,
) (*ImportStats, error) {
	reader, err := Open(calibreDir)
	if err != nil {
		return nil, fmt.Errorf("open calibre library: %w", err)
	}
	defer reader.Close()

	calibreBooks, err := reader.ListBooks(ctx)
	if err != nil {
		return nil, fmt.Errorf("list calibre books: %w", err)
	}

	stats := &ImportStats{
		TotalCalibreBooks: len(calibreBooks),
		Errors:            make([]string, 0),
	}

	total := len(calibreBooks)
	for idx, cBook := range calibreBooks {
		select {
		case <-ctx.Done():
			return stats, ctx.Err()
		default:
		}

		var bookErr error
		if err := imp.importSingleBook(ctx, reader, calibreDir, cBook, opts, stats); err != nil {
			bookErr = err
			msg := fmt.Sprintf("book '%s' (ID %d): %v", cBook.Title, cBook.ID, err)
			slog.Warn("Failed to import calibre book", "book_id", cBook.ID, "title", cBook.Title, "error", err)
			stats.Errors = append(stats.Errors, msg)
		}

		if onProgress != nil {
			onProgress(idx+1, total, cBook.Title, bookErr)
		}
	}

	slog.Info("Calibre import completed",
		"total", stats.TotalCalibreBooks,
		"imported", stats.ImportedBooks,
		"formats_attached", stats.FormatsAttached,
		"skipped", stats.Skipped,
		"errors", len(stats.Errors),
	)

	return stats, nil
}

func (imp *Importer) importSingleBook(
	ctx context.Context,
	reader *Reader,
	calibreDir string,
	cBook CalibreBook,
	opts ImportOptions,
	stats *ImportStats,
) error {
	// 1. Получаем форматы данных
	dataList, err := reader.GetDataForBook(ctx, cBook.ID)
	if err != nil {
		return fmt.Errorf("get data formats: %w", err)
	}
	if len(dataList) == 0 {
		stats.Skipped++
		return nil // Книга без файлов (только метаданные)
	}

	// 2. Получаем авторов, серии, теги, аннотацию и идентификаторы
	cAuthors, err := reader.GetAuthorsForBook(ctx, cBook.ID)
	if err != nil {
		return fmt.Errorf("get authors: %w", err)
	}

	cSeries, err := reader.GetSeriesForBook(ctx, cBook.ID)
	if err != nil {
		return fmt.Errorf("get series: %w", err)
	}

	cTags, err := reader.GetTagsForBook(ctx, cBook.ID)
	if err != nil {
		return fmt.Errorf("get tags: %w", err)
	}

	annotationHtml, err := reader.GetCommentForBook(ctx, cBook.ID)
	if err != nil {
		return fmt.Errorf("get comment: %w", err)
	}
	cleanAnnotation := cleanHtml(annotationHtml)

	identifiers, err := reader.GetIdentifiersForBook(ctx, cBook.ID)
	if err != nil {
		return fmt.Errorf("get identifiers: %w", err)
	}

	// 3. Проверяем, существует ли уже книга в Бояне
	authorName := ""
	if len(cAuthors) > 0 {
		authorName = cAuthors[0].Name
	}
	existingBook, _ := imp.bookRepo.FindBookByTitleAndAuthor(ctx, cBook.Title, authorName)

	now := time.Now().UTC()
	var targetBook *models.Book

	// Подготавливаем файлы для книги
	createdFiles := make([]*models.BookFile, 0, len(dataList))

	for _, d := range dataList {
		ext := strings.ToLower(d.Format)
		sourceFileName := fmt.Sprintf("%s.%s", d.Name, ext)
		sourceFilePath := filepath.Join(calibreDir, cBook.Path, sourceFileName)

		info, err := os.Stat(sourceFilePath)
		if err != nil {
			sourceFilePath = findFileCaseInsensitive(filepath.Join(calibreDir, cBook.Path), sourceFileName)
			info, err = os.Stat(sourceFilePath)
			if err != nil {
				continue // Файл не найден на диске
			}
		}

		sha, err := computeFileSHA256(sourceFilePath)
		if err != nil {
			return fmt.Errorf("compute sha256 for %s: %w", sourceFilePath, err)
		}

		finalFilePath := sourceFilePath
		fileSize := info.Size()
		fileSHA := sha

		if opts.CopyFiles {
			destDir := filepath.Join(imp.cfg.Storage.LibraryDir, sanitizePath(authorName), sanitizePath(cBook.Title))
			_ = os.MkdirAll(destDir, 0755)
			destPath := filepath.Join(destDir, filepath.Base(sourceFilePath))

			if err := copyFile(sourceFilePath, destPath); err != nil {
				return fmt.Errorf("copy book file: %w", err)
			}
			finalFilePath = destPath

			// Автоматическая санитизация скопированного FB2 файла в библиотеке
			if ext == "fb2" {
				if modified, err := fb2.SanitizeFB2File(destPath); err == nil && modified {
					if updatedInfo, err := os.Stat(destPath); err == nil {
						fileSize = updatedInfo.Size()
					}
					if updatedSHA, err := computeFileSHA256(destPath); err == nil {
						fileSHA = updatedSHA
					}
				}
			}
		}

		fileModel := &models.BookFile{
			ID:        uuid.NewString(),
			Format:    ext,
			FilePath:  finalFilePath,
			FileSize:  fileSize,
			SHA256:    fileSHA,
			CreatedAt: now,
		}
		createdFiles = append(createdFiles, fileModel)
	}

	if len(createdFiles) == 0 {
		stats.Skipped++
		return nil
	}

	if existingBook != nil {
		targetBook = existingBook
		existingFormats := make(map[string]bool)
		for _, ef := range existingBook.Files {
			existingFormats[strings.ToLower(ef.Format)] = true
		}

		for _, f := range createdFiles {
			if !existingFormats[f.Format] {
				f.BookID = targetBook.ID
				if err := imp.bookRepo.AddBookFile(ctx, f); err == nil {
					stats.FormatsAttached++
					existingFormats[f.Format] = true
				}
			}
		}
	} else {
		newBookID := uuid.NewString()
		isbn := identifiers["isbn"]

		targetBook = &models.Book{
			ID:          newBookID,
			Title:       cBook.Title,
			Annotation:  cleanAnnotation,
			CoverCached: false,
			ISBN:        isbn,
			CreatedAt:   now,
			UpdatedAt:   now,
		}

		// Авторы
		authors := make([]models.AuthorDetail, 0, len(cAuthors))
		for idx, a := range cAuthors {
			authors = append(authors, models.AuthorDetail{
				Author: models.Author{
					ID:       uuid.NewString(),
					Name:     a.Name,
					SortName: a.Sort,
				},
				Role:  "author",
				Order: idx,
			})
		}

		// Серии
		series := make([]models.SeriesDetail, 0, len(cSeries))
		for _, s := range cSeries {
			series = append(series, models.SeriesDetail{
				Series: models.Series{
					ID:       uuid.NewString(),
					Name:     s.Name,
					SortName: s.Sort,
				},
				Index: cBook.SeriesIndex,
			})
		}

		// Жанры
		genres := make([]models.Genre, 0, len(cTags))
		for _, t := range cTags {
			code := strings.ToLower(strings.ReplaceAll(t.Name, " ", "_"))
			genres = append(genres, models.Genre{
				Code:       code,
				NameRU:     t.Name,
				NameEN:     t.Name,
				CategoryRU: "Теги",
				CategoryEN: "Tags",
			})
		}

		primaryFile := createdFiles[0]
		primaryFile.BookID = targetBook.ID

		if err := imp.bookRepo.SaveBook(ctx, targetBook, authors, series, genres, primaryFile); err != nil {
			return fmt.Errorf("save book to repository: %w", err)
		}
		stats.ImportedBooks++

		// Добавляем остальные форматы (если их больше одного)
		for _, extraFile := range createdFiles[1:] {
			extraFile.BookID = targetBook.ID
			_ = imp.bookRepo.AddBookFile(ctx, extraFile)
			stats.FormatsAttached++
		}
	}

	// 4. Обработка обложки (cover.jpg)
	if cBook.HasCover && imp.coverCache != nil {
		coverPath := filepath.Join(calibreDir, cBook.Path, "cover.jpg")
		if coverBytes, err := os.ReadFile(coverPath); err == nil && len(coverBytes) > 0 {
			if _, err := imp.coverCache.SaveCover(targetBook.ID, coverBytes); err == nil {
				_ = imp.bookRepo.SetCoverCached(ctx, targetBook.ID, true)
			}
		}
	}

	return nil
}

func computeFileSHA256(filePath string) (string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}

var blockTagRegex = regexp.MustCompile(`(?i)</?(?:p|div|br|li|h[1-6])[^>]*>`)
var anyTagRegex = regexp.MustCompile(`<[^>]*>`)
var punctuationSpaceRegex = regexp.MustCompile(`\s+([.,;:!?])`)
var multiSpaceRegex = regexp.MustCompile(`\s+`)

func cleanHtml(input string) string {
	if input == "" {
		return ""
	}
	withSpaces := blockTagRegex.ReplaceAllString(input, " ")
	noTags := anyTagRegex.ReplaceAllString(withSpaces, "")
	decoded := html.UnescapeString(noTags)
	noPunctSpace := punctuationSpaceRegex.ReplaceAllString(decoded, "$1")
	return strings.TrimSpace(multiSpaceRegex.ReplaceAllString(noPunctSpace, " "))
}

func sanitizePath(name string) string {
	bad := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	res := name
	for _, b := range bad {
		res = strings.ReplaceAll(res, b, "_")
	}
	res = strings.TrimSpace(res)
	if res == "" {
		return "Unknown"
	}
	return res
}

func findFileCaseInsensitive(dir, targetName string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return filepath.Join(dir, targetName)
	}
	targetLower := strings.ToLower(targetName)
	for _, e := range entries {
		if strings.ToLower(e.Name()) == targetLower {
			return filepath.Join(dir, e.Name())
		}
	}
	return filepath.Join(dir, targetName)
}
