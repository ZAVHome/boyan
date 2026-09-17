//go:build !windows

package services

import (
	"syscall"
)

// GetDiskUsage возвращает общее и свободное дисковое пространство для пути path на Unix/Linux.
func GetDiskUsage(path string) (total uint64, free uint64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}

	// Общее число байт = Blocks * Bsize
	total = stat.Blocks * uint64(stat.Bsize)
	// Доступное число байт для непривилегированных пользователей = Bavail * Bsize
	free = stat.Bavail * uint64(stat.Bsize)

	return total, free, nil
}
