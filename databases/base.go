package databases

import (
	"gorm.io/gorm"
)

// IGormDB adalah interface untuk semua koneksi database.
// Gunakan GetDB() untuk mengakses *gorm.DB dengan fluent API penuh.
type IGormDB interface {
	// GetDB mengembalikan *gorm.DB. Periksa IsConnected() dahulu.
	GetDB() *gorm.DB
	// IsConnected memeriksa apakah koneksi berhasil dibuat.
	IsConnected() bool
	// AutoMigrate menjalankan migrasi otomatis untuk model GORM.
	AutoMigrate(dst ...any) error
}

// baseDB adalah implementasi dasar yang digunakan oleh semua driver.
type baseDB struct {
	db        *gorm.DB
	connected bool
}

func (b *baseDB) GetDB() *gorm.DB {
	return b.db
}

func (b *baseDB) IsConnected() bool {
	return b.connected
}

func (b *baseDB) AutoMigrate(dst ...any) error {
	if !b.connected || b.db == nil {
		return nil
	}
	return b.db.AutoMigrate(dst...)
}
