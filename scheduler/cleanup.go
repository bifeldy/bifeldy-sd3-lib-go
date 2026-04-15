package scheduler

import (
	"context"
	"time"

	"github.com/bifeldy/bifeldy-sd3-lib-go/models"
	"github.com/rs/zerolog"
	"os"
	"path/filepath"
)

// CleanupLogsJob menghapus file log yang lebih lama dari LogRetainDays.
// Job ini didaftarkan otomatis oleh library setiap hari tengah malam.
func CleanupLogsJob(cfg *models.Config, log *zerolog.Logger) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		log.Info().Str("dir", cfg.LogDir).Int("retain_days", cfg.LogRetainDays).
			Msg("[CleanupJob] Mulai hapus log lama")

		entries, err := os.ReadDir(cfg.LogDir)
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}

		cutoff := time.Now().AddDate(0, 0, -cfg.LogRetainDays)
		deleted := 0

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			info, err := entry.Info()
			if err != nil {
				continue
			}

			if info.ModTime().Before(cutoff) {
				path := filepath.Join(cfg.LogDir, entry.Name())
				if removeErr := os.Remove(path); removeErr != nil {
					log.Warn().Err(removeErr).Str("file", path).Msg("[CleanupJob] Gagal hapus file")
				} else {
					deleted++
					log.Debug().Str("file", path).Msg("[CleanupJob] File dihapus")
				}
			}
		}

		log.Info().Int("deleted", deleted).Msg("[CleanupJob] Selesai")
		return nil
	}
}
