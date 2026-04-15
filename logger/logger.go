package logger

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/bifeldy/bifeldy-sd3-lib-go/models"
	"github.com/rs/zerolog"
)

// ===========================================================================
// Daily Rolling File Writer
// ===========================================================================

// dailyFileWriter menulis log ke file baru setiap hari.
// Format nama file: <prefix>YYYYMMDD.log
type dailyFileWriter struct {
	dir     string
	prefix  string
	mu      sync.Mutex
	file    *os.File
	curDate string
}

func (w *dailyFileWriter) Write(p []byte) (n int, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	today := time.Now().Format("20060102")
	if w.curDate != today || w.file == nil {
		if w.file != nil {
			_ = w.file.Close()
		}

		if mkErr := os.MkdirAll(w.dir, 0o755); mkErr != nil {
			return 0, mkErr
		}

		filename := filepath.Join(w.dir, fmt.Sprintf("%s%s.log", w.prefix, today))
		f, openErr := os.OpenFile(filename, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if openErr != nil {
			return 0, openErr
		}

		w.file = f
		w.curDate = today
	}

	return w.file.Write(p)
}

func (w *dailyFileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// ===========================================================================
// Level Filter Writer
// Hanya meneruskan log pada level >= minLevel ke writer target.
// ===========================================================================

type levelFilterWriter struct {
	w        io.Writer
	minLevel zerolog.Level
}

// Write dipanggil tanpa info level — kita skip agar tidak bypass filter.
func (lfw *levelFilterWriter) Write(p []byte) (n int, err error) {
	return len(p), nil
}

// WriteLevel dipanggil oleh zerolog dengan info level.
// Implementasikan zerolog.LevelWriter agar MultiLevelWriter menggunakannya.
func (lfw *levelFilterWriter) WriteLevel(l zerolog.Level, p []byte) (n int, err error) {
	if l >= lfw.minLevel {
		return lfw.w.Write(p)
	}
	return len(p), nil
}

// Pastikan levelFilterWriter mengimplementasikan zerolog.LevelWriter
var _ zerolog.LevelWriter = (*levelFilterWriter)(nil)

// ===========================================================================
// Constructor
// ===========================================================================

// NewLogger membuat zerolog.Logger dengan:
//   - Console output: semua level (colorized)
//   - File output: hanya ERROR ke atas, rolling harian
func NewLogger(cfg *models.Config) *zerolog.Logger {
	// Console: semua level, format manusia
	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: "2006-01-02 15:04:05",
	}

	// File: hanya ERROR ke atas, rolling per hari
	dailyFile := &dailyFileWriter{
		dir:    cfg.LogDir,
		prefix: "error_",
	}
	fileWriter := &levelFilterWriter{
		w:        dailyFile,
		minLevel: zerolog.ErrorLevel,
	}

	multi := zerolog.MultiLevelWriter(consoleWriter, fileWriter)

	level := zerolog.InfoLevel
	if cfg.Debug {
		level = zerolog.DebugLevel
	}

	log := zerolog.New(multi).Level(level).With().Timestamp().Logger()
	return &log
}
