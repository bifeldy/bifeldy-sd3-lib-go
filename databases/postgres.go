package databases

import (
	"github.com/bifeldy/bifeldy-sd3-lib-go/models"
	"github.com/rs/zerolog"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// PostgresDB adalah koneksi PostgreSQL.
type PostgresDB struct {
	baseDB
}

// NewPostgres membuat koneksi PostgreSQL dari DSN di Config.
// Contoh DSN: "host=localhost user=postgres password=pass dbname=mydb port=5432 sslmode=disable"
// Jika DB_POSTGRES kosong, koneksi dilewati.
func NewPostgres(cfg *models.Config, log *zerolog.Logger) IGormDB {
	db := &PostgresDB{}

	if cfg.PostgresDSN == "" {
		log.Debug().Msg("[Postgres] DSN kosong, koneksi dilewati")
		return db
	}

	gormCfg := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	if cfg.Debug {
		gormCfg.Logger = logger.Default.LogMode(logger.Info)
	}

	conn, err := gorm.Open(postgres.Open(cfg.PostgresDSN), gormCfg)
	if err != nil {
		log.Error().Err(err).Msg("[Postgres] Gagal konek")
		return db
	}

	// Connection pool
	sqlDB, _ := conn.DB()
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetMaxOpenConns(25)

	db.db = conn
	db.connected = true
	log.Info().Msg("[Postgres] Terhubung")
	return db
}
