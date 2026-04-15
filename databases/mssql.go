package databases

import (
	"github.com/bifeldy/bifeldy-sd3-lib-go/models"
	"github.com/rs/zerolog"
	"gorm.io/driver/sqlserver"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MsSQLDB adalah koneksi Microsoft SQL Server.
type MsSQLDB struct {
	baseDB
}

// NewMsSQL membuat koneksi MS SQL Server dari DSN di Config.
// Contoh DSN: "sqlserver://sa:password@localhost:1433?database=mydb"
// Jika DB_MSSQL kosong, koneksi dilewati.
func NewMsSQL(cfg *models.Config, log *zerolog.Logger) IGormDB {
	db := &MsSQLDB{}

	if cfg.MsSQLDSN == "" {
		log.Debug().Msg("[MsSQL] DSN kosong, koneksi dilewati")
		return db
	}

	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	if cfg.Debug {
		gormCfg.Logger = logger.Default.LogMode(logger.Info)
	}

	conn, err := gorm.Open(sqlserver.Open(cfg.MsSQLDSN), gormCfg)
	if err != nil {
		log.Error().Err(err).Msg("[MsSQL] Gagal konek")
		return db
	}

	// Connection pool
	sqlDB, _ := conn.DB()
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(25)

	db.db = conn
	db.connected = true
	log.Info().Msg("[MsSQL] Terhubung")
	return db
}
