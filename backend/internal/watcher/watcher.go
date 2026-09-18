package watcher

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"boyan/internal/config"
	"boyan/internal/models"
	"boyan/internal/parsers/cover"
	"boyan/internal/parsers/fb2"
	"boyan/internal/parsers/zip"
	"boyan/internal/storage"

	"github.com/fsnotify/fsnotify"
	"github.com/google/uuid"
)

type ProcessResult struct {
	Status         string `json:"status"` // "imported", "format_attached", "quarantined", "skipped"
	BookID         string `json:"book_id,omitempty"`
	ExistingBookID string `json:"existing_book_id,omitempty"`
	ConflictType   string `json:"conflict_type,omitempty"`
	QuarantineID   string `json:"quarantine_id,omitempty"`
	Message        string `json:"message,omitempty"`
	SourceFile     string `json:"source_file"`
	LibraryFile    string `json:"library_file,omitempty"`
	CalculatedSHA  string `json:"sha256,omitempty"`
}

type Watcher struct {
	cfg            *config.Config
	bookRepo       *storage.BookRepository
	quarantineRepo *storage.QuarantineRepository
	coverCache     *cover.CoverCache
	deduplicator   *Deduplicator

	processingMu sync.Mutex
	activeFiles  map[string]bool

	settleMu sync.Mutex
	pending  map[string]time.Time
}

func NewWatcher(
	cfg *config.Config,
	bookRepo *storage.BookRepository,
	quarantineRepo *storage.QuarantineRepository,
	coverCache *cover.CoverCache,
) *Watcher {
	return &Watcher{
		cfg:            cfg,
		bookRepo:       bookRepo,
		quarantineRepo: quarantineRepo,
		coverCache:     coverCache,
		deduplicator:   NewDeduplicator(bookRepo),
		activeFiles:    make(map[string]bool),
		pending:        make(map[string]time.Time),
	}
}

// Start запускает мониторинг каталога watch_dir и периодическую обработку стабилизировавшихся файлов.
func (w *Watcher) Start(ctx context.Context) error {
	watchDir := w.cfg.Storage.WatchDir
	if err := os.MkdirAll(watchDir, 0755); err != nil {
		return fmt.Errorf("create watch dir: %w", err)
	}
	if err := os.MkdirAll(w.cfg.Storage.LibraryDir, 0755); err != nil {
		return fmt.Errorf("create library dir: %w", err)
	}

	slog.Info("Ingest watcher initialized", "watch_dir", watchDir, "library_dir", w.cfg.Storage.LibraryDir)

	// 1. Начальное сканирование файлов в watch_dir
	go w.scanDirectory(ctx, watchDir)

	// 2. Инициализация fsnotify
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("init fsnotify: %w", err)
	}

	// Рекурсивно добавляем папки для мониторинга
	_ = filepath.Walk(watchDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && info.IsDir() {
			_ = fsw.Add(path)
		}
		return nil
	})

	// 3. Фоновый таймер контроля стабилизации файлов (settle delay)
	settleDelay := time.Duration(w.cfg.Storage.Watcher.SettleDelaySeconds) * time.Second
	if settleDelay < time.Second {
		settleDelay = 3 * time.Second
	}

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case now := <-ticker.C:
				w.checkSettledFiles(ctx, now)
			}
		}
	}()

	// 4. Цикл обработки событий fsnotify
	go func() {
		defer fsw.Close()

		for {
			select {
			case <-ctx.Done():
				slog.Info("Ingest watcher stopping on context cancellation")
				return
			case event, ok := <-fsw.Events:
				if !ok {
					return
				}

				if event.Has(fsnotify.Create) || event.Has(fsnotify.Write) {
					info, err := os.Stat(event.Name)
					if err != nil {
						continue
					}

					if info.IsDir() {
						_ = fsw.Add(event.Name)
						continue
					}

					if isSupportedBookFile(event.Name) {
						w.settleMu.Lock()
						w.pending[event.Name] = time.Now().Add(settleDelay)
						w.settleMu.Unlock()
					}
				}
			case err, ok := <-fsw.Errors:
				if !ok {
					return
				}
				slog.Warn("Watcher error", "err", err)
			}
		}
	}()

	return nil
}

func (w *Watcher) checkSettledFiles(ctx context.Context, now time.Time) {
	w.settleMu.Lock()
	var readyFiles []string
	for path, readyTime := range w.pending {
		if now.After(readyTime) {
			readyFiles = append(readyFiles, path)
			delete(w.pending, path)
		}
	}
	w.settleMu.Unlock()

	for _, path := range readyFiles {
		// Проверяем, существует ли еще файл
		info, err := os.Stat(path)
		if err != nil || info.IsDir() || info.Size() == 0 {
			continue
		}

		go func(p string) {
			res, err := w.ProcessFile(ctx, p, filepath.Base(p))
			if err != nil {
				slog.Error("Failed to ingest book file", "path", p, "err", err)
			} else {
				slog.Info("Ingested book file", "path", p, "status", res.Status, "book_id", res.BookID)
			}
		}(path)
	}
}

func (w *Watcher) scanDirectory(ctx context.Context, dir string) {
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}

		if isSupportedBookFile(path) && info.Size() > 0 {
			res, err := w.ProcessFile(ctx, path, filepath.Base(path))
			if err != nil {
				slog.Error("Failed to ingest book file during initial scan", "path", path, "err", err)
			} else if res.Status != "skipped" {
				slog.Info("Initial scan processed file", "path", path, "status", res.Status, "book_id", res.BookID)
			}
		}
		return nil
	})
}

// ScanDirectoryWithProgress выполняет полное сканирование директории с оповещением о прогрессе.
func (w *Watcher) ScanDirectoryWithProgress(
	ctx context.Context,
	dir string,
	onProgress func(processed, total int, currentItem string, err error),
) error {
	var files []string
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if isSupportedBookFile(path) && info.Size() > 0 {
			files = append(files, path)
		}
		return nil
	})

	total := len(files)
	if total == 0 {
		if onProgress != nil {
			onProgress(0, 0, "", nil)
		}
		return nil
	}

	for i, path := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		filename := filepath.Base(path)
		_, err := w.ProcessFile(ctx, path, filename)
		if onProgress != nil {
			onProgress(i+1, total, filename, err)
		}
	}

	return nil
}


// ProcessFile выполняет полный пайплайн инжеста книги (детекция формата, парсинг, дедупликация, сохранение).
func (w *Watcher) ProcessFile(ctx context.Context, sourcePath, originalFilename string) (*ProcessResult, error) {
	w.processingMu.Lock()
	if w.activeFiles[sourcePath] {
		w.processingMu.Unlock()
		return &ProcessResult{Status: "skipped", Message: "file already being processed", SourceFile: sourcePath}, nil
	}
	w.activeFiles[sourcePath] = true
	w.processingMu.Unlock()

	defer func() {
		w.processingMu.Lock()
		delete(w.activeFiles, sourcePath)
		w.processingMu.Unlock()
	}()

	fileInfo, err := os.Stat(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("stat source file: %w", err)
	}
	if fileInfo.Size() == 0 {
		return nil, fmt.Errorf("file is empty")
	}

	// Автоматическая санитизация FB2 перед анализом и сохранением в библиотеку
	lowerName := strings.ToLower(originalFilename)
	lowerPath := strings.ToLower(sourcePath)
	if strings.HasSuffix(lowerName, ".fb2") || strings.HasSuffix(lowerPath, ".fb2") {
		if modified, err := fb2.SanitizeFB2File(sourcePath); err == nil && modified {
			if updatedInfo, err := os.Stat(sourcePath); err == nil {
				fileInfo = updatedInfo
			}
		}
	}

	// 1. Определение формата и парсинг книги
	format, innerPath, bookInfo, err := w.parseBook(sourcePath, originalFilename)
	if err != nil {
		return nil, fmt.Errorf("parse book %s: %w", sourcePath, err)
	}

	authorName := ""
	if len(bookInfo.Authors) > 0 {
		authorName = bookInfo.Authors[0].Name
	}
	authorsSummary := formatAuthors(bookInfo.Authors)

	// 2. Проверка дубликатов
	dupResult, err := w.deduplicator.CheckDuplicate(ctx, sourcePath, bookInfo.Book.Title, authorName, format)
	if err != nil {
		return nil, fmt.Errorf("check duplicate: %w", err)
	}

	// 3. Обработка сценариев дубликатов
	switch dupResult.Type {
	case DuplicateExactHash:
		// Точная копия файла уже есть в библиотеке
		if w.cfg.Storage.Watcher.QuarantineDuplicates {
			qItem := &storage.QuarantineItem{
				ID:             uuid.NewString(),
				FilePath:       sourcePath,
				FileSize:       fileInfo.Size(),
				SHA256:         dupResult.CalculatedHash,
				Format:         format,
				ParsedTitle:    bookInfo.Book.Title,
				ParsedAuthors:  authorsSummary,
				ConflictType:   "exact_hash",
				CreatedAt:      time.Now().UTC(),
			}
			if dupResult.ExistingBook != nil {
				qItem.ExistingBookID = &dupResult.ExistingBook.ID
			}
			_ = w.quarantineRepo.AddToQuarantine(ctx, qItem)

			if w.cfg.Storage.Watcher.DeleteSourceAfterImport {
				_ = os.Remove(sourcePath)
			}

			return &ProcessResult{
				Status:        "quarantined",
				ConflictType:  "exact_hash",
				QuarantineID:  qItem.ID,
				Message:       "Exact SHA-256 match found; sent to quarantine",
				SourceFile:    sourcePath,
				CalculatedSHA: dupResult.CalculatedHash,
			}, nil
		}

		if w.cfg.Storage.Watcher.DeleteSourceAfterImport {
			_ = os.Remove(sourcePath)
		}
		return &ProcessResult{
			Status:        "skipped",
			Message:       "Exact file already exists in library",
			SourceFile:    sourcePath,
			CalculatedSHA: dupResult.CalculatedHash,
		}, nil

	case DuplicateSameFormat:
		// Та же книга и тот же формат, но другой хеш (другое издание/перевод)
		qItem := &storage.QuarantineItem{
			ID:             uuid.NewString(),
			FilePath:       sourcePath,
			FileSize:       fileInfo.Size(),
			SHA256:         dupResult.CalculatedHash,
			Format:         format,
			ParsedTitle:    bookInfo.Book.Title,
			ParsedAuthors:  authorsSummary,
			ConflictType:   "same_format",
			CreatedAt:      time.Now().UTC(),
		}
		if dupResult.ExistingBook != nil {
			qItem.ExistingBookID = &dupResult.ExistingBook.ID
		}
		_ = w.quarantineRepo.AddToQuarantine(ctx, qItem)

		return &ProcessResult{
			Status:        "quarantined",
			ConflictType:  "same_format",
			QuarantineID:  qItem.ID,
			ExistingBookID: dupResult.ExistingBook.ID,
			Message:       "Same book and format already exists with different hash; sent to quarantine",
			SourceFile:    sourcePath,
			CalculatedSHA: dupResult.CalculatedHash,
		}, nil

	case DuplicateNewFormat:
		// Книга уже есть, но в другом формате (например, был FB2, а пришел EPUB) -> прикрепляем формат!
		existingBook := dupResult.ExistingBook
		ext := "." + format
		relPath := FormatPath(w.cfg.Storage.PathTemplate, existingBook, existingBook.Authors, existingBook.Series, ext)
		destPath := filepath.Join(w.cfg.Storage.LibraryDir, relPath)

		if err := copyOrMoveFile(sourcePath, destPath, w.cfg.Storage.Watcher.DeleteSourceAfterImport); err != nil {
			return nil, fmt.Errorf("copy file to library: %w", err)
		}

		newFile := &models.BookFile{
			ID:               uuid.NewString(),
			BookID:           existingBook.ID,
			Format:           format,
			FilePath:         relPath,
			ArchiveInnerPath: innerPath,
			FileSize:         fileInfo.Size(),
			SHA256:           dupResult.CalculatedHash,
			CreatedAt:        time.Now().UTC(),
		}
		if err := w.bookRepo.AddBookFile(ctx, newFile); err != nil {
			return nil, fmt.Errorf("add book file record: %w", err)
		}

		return &ProcessResult{
			Status:        "format_attached",
			BookID:        existingBook.ID,
			SourceFile:    sourcePath,
			LibraryFile:   relPath,
			CalculatedSHA: dupResult.CalculatedHash,
			Message:       fmt.Sprintf("New format %s attached to existing book %s", format, existingBook.ID),
		}, nil

	case DuplicateNone:
		// 4. Новая книга!
		bookID := uuid.NewString()
		bookInfo.Book.ID = bookID

		ext := "." + format
		relPath := FormatPath(w.cfg.Storage.PathTemplate, &bookInfo.Book, bookInfo.Authors, bookInfo.Series, ext)
		destPath := filepath.Join(w.cfg.Storage.LibraryDir, relPath)

		if err := copyOrMoveFile(sourcePath, destPath, w.cfg.Storage.Watcher.DeleteSourceAfterImport); err != nil {
			return nil, fmt.Errorf("copy file to library: %w", err)
		}

		// Обработка и сохранение обложки книги в LRU кеш
		if len(bookInfo.CoverBytes) > 0 && w.coverCache != nil {
			processedCover, err := cover.ProcessCover(bookInfo.CoverBytes, w.cfg.Metadata.CoverThumbnailSize)
			if err == nil {
				if _, err := w.coverCache.SaveCover(bookID, processedCover); err == nil {
					bookInfo.Book.CoverCached = true
				}
			}
		}

		bookFile := &models.BookFile{
			ID:               uuid.NewString(),
			BookID:           bookID,
			Format:           format,
			FilePath:         relPath,
			ArchiveInnerPath: innerPath,
			FileSize:         fileInfo.Size(),
			SHA256:           dupResult.CalculatedHash,
			CreatedAt:        time.Now().UTC(),
		}

		if err := w.bookRepo.SaveBook(ctx, &bookInfo.Book, bookInfo.Authors, bookInfo.Series, bookInfo.Genres, bookFile); err != nil {
			return nil, fmt.Errorf("save book to db: %w", err)
		}

		return &ProcessResult{
			Status:        "imported",
			BookID:        bookID,
			SourceFile:    sourcePath,
			LibraryFile:   relPath,
			CalculatedSHA: dupResult.CalculatedHash,
			Message:       "Successfully imported new book",
		}, nil
	}

	return &ProcessResult{Status: "skipped", SourceFile: sourcePath}, nil
}

func (w *Watcher) parseBook(filePath, originalFilename string) (format string, innerPath string, info *fb2.FB2BookInfo, err error) {
	lowerName := strings.ToLower(originalFilename)
	lowerPath := strings.ToLower(filePath)

	isZip := strings.HasSuffix(lowerName, ".zip") || strings.HasSuffix(lowerPath, ".zip")
	isFb2Zip := strings.HasSuffix(lowerName, ".fb2.zip") || strings.HasSuffix(lowerPath, ".fb2.zip")

	if isZip || isFb2Zip {
		// Открываем zip и стримим .fb2
		stream, err := zip.ExtractFB2Stream(filePath)
		if err != nil {
			return "", "", nil, fmt.Errorf("extract fb2 from zip: %w", err)
		}
		defer stream.Close()

		info, err = fb2.ParseFB2(stream)
		if err != nil {
			return "", "", nil, fmt.Errorf("parse fb2 stream: %w", err)
		}

		// Найдем внутреннее имя файла
		zr, file, err := zip.OpenFB2FromZipFile(filePath)
		if err == nil {
			innerPath = file.Name
			zr.Close()
		}

		return "fb2.zip", innerPath, info, nil
	}

	if strings.HasSuffix(lowerName, ".fb2") || strings.HasSuffix(lowerPath, ".fb2") {
		f, err := os.Open(filePath)
		if err != nil {
			return "", "", nil, fmt.Errorf("open fb2 file: %w", err)
		}
		defer f.Close()

		info, err = fb2.ParseFB2(f)
		if err != nil {
			return "", "", nil, fmt.Errorf("parse fb2: %w", err)
		}

		return "fb2", "", info, nil
	}

	return "", "", nil, fmt.Errorf("unsupported file format: %s", originalFilename)
}

func isSupportedBookFile(path string) bool {
	lower := strings.ToLower(path)
	return strings.HasSuffix(lower, ".fb2") ||
		strings.HasSuffix(lower, ".fb2.zip") ||
		strings.HasSuffix(lower, ".zip")
}

func formatAuthors(authors []models.AuthorDetail) string {
	var names []string
	for _, a := range authors {
		if a.Name != "" {
			names = append(names, a.Name)
		}
	}
	return strings.Join(names, ", ")
}

func copyOrMoveFile(src, dst string, deleteSource bool) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("mkdir for dst: %w", err)
	}

	// Попытка прямого переименования (быстро, если на одном дисковом разделе)
	if deleteSource {
		if err := os.Rename(src, dst); err == nil {
			return nil
		}
	}

	// Если разные разделы или deleteSource == false, копируем поток
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open src: %w", err)
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("create dst: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("copy bytes: %w", err)
	}

	if deleteSource {
		_ = os.Remove(src)
	}

	return nil
}
