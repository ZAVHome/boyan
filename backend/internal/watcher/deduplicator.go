package watcher

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"strings"

	"boyan/internal/models"
	"boyan/internal/storage"
)

type DuplicateType string

const (
	DuplicateNone       DuplicateType = "none"
	DuplicateExactHash  DuplicateType = "exact_hash"
	DuplicateNewFormat  DuplicateType = "new_format"
	DuplicateSameFormat DuplicateType = "same_format"
	DuplicateFuzzyMatch DuplicateType = "fuzzy_match"
)

type DuplicateResult struct {
	Type           DuplicateType
	ExistingBook   *models.Book
	ExistingFile   *models.BookFile
	CalculatedHash string
}

type Deduplicator struct {
	bookRepo *storage.BookRepository
}

func NewDeduplicator(bookRepo *storage.BookRepository) *Deduplicator {
	return &Deduplicator{bookRepo: bookRepo}
}

// CalculateFileSHA256 вычисляет SHA-256 хеш содержимого файла.
func CalculateFileSHA256(filePath string) (string, int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("open file for hash: %w", err)
	}
	defer f.Close()

	hasher := sha256.New()
	size, err := io.Copy(hasher, f)
	if err != nil {
		return "", 0, fmt.Errorf("hash file copy: %w", err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), size, nil
}

// CheckDuplicate анализирует файл на предмет дубликатов в библиотеке.
func (d *Deduplicator) CheckDuplicate(
	ctx context.Context,
	filePath string,
	parsedTitle string,
	parsedAuthor string,
	format string,
) (*DuplicateResult, error) {
	hash, _, err := CalculateFileSHA256(filePath)
	if err != nil {
		return nil, err
	}

	// 1. Проверка на точное совпадение хеша (Exact SHA-256)
	existingFile, err := d.bookRepo.FindFileBySHA256(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("check sha256 duplicate: %w", err)
	}
	if existingFile != nil {
		existingBook, _ := d.bookRepo.GetBookByID(ctx, existingFile.BookID)
		return &DuplicateResult{
			Type:           DuplicateExactHash,
			ExistingBook:   existingBook,
			ExistingFile:   existingFile,
			CalculatedHash: hash,
		}, nil
	}

	// 2. Проверка по автору и названию книги (Metadata Match)
	cleanTitle := strings.TrimSpace(parsedTitle)
	cleanAuthor := strings.TrimSpace(parsedAuthor)

	if cleanTitle != "" {
		existingBook, err := d.bookRepo.FindBookByTitleAndAuthor(ctx, cleanTitle, cleanAuthor)
		if err != nil {
			return nil, fmt.Errorf("check title/author match: %w", err)
		}
		if existingBook != nil {
			// Проверяем, есть ли уже файл такого же формата
			hasSameFormat := false
			normFormat := normalizeFormat(format)

			for _, f := range existingBook.Files {
				if normalizeFormat(f.Format) == normFormat {
					hasSameFormat = true
					break
				}
			}

			if hasSameFormat {
				return &DuplicateResult{
					Type:           DuplicateSameFormat,
					ExistingBook:   existingBook,
					CalculatedHash: hash,
				}, nil
			}

			// Это новый формат той же книги
			return &DuplicateResult{
				Type:           DuplicateNewFormat,
				ExistingBook:   existingBook,
				CalculatedHash: hash,
			}, nil
		}
	}

	return &DuplicateResult{
		Type:           DuplicateNone,
		CalculatedHash: hash,
	}, nil
}

func normalizeFormat(fmt string) string {
	fmt = strings.ToLower(strings.TrimSpace(fmt))
	if fmt == "fb2.zip" {
		return "fb2"
	}
	return fmt
}
