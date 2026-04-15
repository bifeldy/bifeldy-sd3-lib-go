package models

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ===========================================================================
// GORM Models
// ===========================================================================

// ApiKey adalah tabel penyimpanan API key dengan whitelist IP.
// IP whitelist diisi comma-separated, kosong = semua IP diizinkan.
type ApiKey struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Key         string    `gorm:"uniqueIndex;not null;size:256" json:"key"`
	Name        string    `gorm:"not null;size:256" json:"name"`
	IPWhitelist string    `gorm:"size:1024" json:"ip_whitelist"` // "192.168.1.1,10.0.0.1"
	Active      bool      `gorm:"default:true" json:"active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// User adalah tabel pengguna aplikasi.
type User struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Username  string    `gorm:"uniqueIndex;not null;size:256" json:"username"`
	Password  string    `gorm:"not null;size:512" json:"-"` // bcrypt hash
	Role      int       `gorm:"default:0" json:"role"`
	Active    bool      `gorm:"default:true" json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ===========================================================================
// JWT
// ===========================================================================

// JwtClaims adalah payload JWT token.
type JwtClaims struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     int    `json:"role"`
	jwt.RegisteredClaims
}

// JwtSession adalah data user yang sudah terautentikasi,
// disimpan ke dalam echo.Context agar bisa diakses di handler.
type JwtSession struct {
	UserID   uint   `json:"user_id"`
	Username string `json:"username"`
	Role     int    `json:"role"`
}

// ===========================================================================
// Context Keys
// ===========================================================================

const (
	JwtContextKey    = "user"     // echo.Context.Get(JwtContextKey)
	ApiKeyContextKey = "api_key"  // echo.Context.Get(ApiKeyContextKey)
)
