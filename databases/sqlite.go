package databases

import (
	"github.com/bifeldy/bifeldy-sd3-lib-go/models"
	gsqlite "github.com/glebarez/sqlite"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SQLiteDB adalah koneksi SQLite menggunakan driver pure-Go (tanpa CGO).
type SQLiteDB struct {
	baseDB
}

// NewSQLite membuat koneksi SQLite dari DSN di Config.
// Jika DB_SQLITE kosong, koneksi dilewati (tidak error).
func NewSQLite(cfg *models.Config, log *zerolog.Logger) IGormDB {
	db := &SQLiteDB{}

	if cfg.SQLiteDSN == "" {
		log.Debug().Msg("[SQLite] DSN kosong, koneksi dilewati")
		return db
	}

	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	if cfg.Debug {
		gormCfg.Logger = logger.Default.LogMode(logger.Info)
	}

	conn, err := gorm.Open(gsqlite.Open(cfg.SQLiteDSN), gormCfg)
	if err != nil {
		log.Error().Err(err).Str("dsn", cfg.SQLiteDSN).Msg("[SQLite] Gagal konek")
		return db
	}

	db.db = conn
	db.connected = true
	log.Info().Str("dsn", cfg.SQLiteDSN).Msg("[SQLite] Terhubung")
	return db
}
