package middlewares

import (
	"net/http"
	"strings"

	"github.com/bifeldy/bifeldy-sd3-lib-go/models"
	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// ApiKeyMiddleware memvalidasi header X-Api-Key dan IP whitelist.
//
// Alur:
//  1. Ambil nilai header "X-Api-Key"
//  2. Cari di tabel api_keys via DB
//  3. Periksa apakah active = true
//  4. Periksa IP client ada di ip_whitelist (kosong = semua diizinkan)
//  5. Simpan data ApiKey ke context untuk dipakai handler
func (f *MiddlewareFactory) ApiKey() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			apiKey := strings.TrimSpace(c.Request().Header.Get("X-Api-Key"))
			if apiKey == "" {
				return c.JSON(http.StatusUnauthorized, models.Err(
					http.StatusUnauthorized,
					"X-Api-Key header wajib diisi",
				))
			}

			// Jika DB tidak tersedia, fallback ke validasi sederhana dari config
			if f.db == nil || !f.db.IsConnected() {
				if f.cfg.JWTSecret == "" || apiKey != f.cfg.JWTSecret {
					f.log.Warn().Str("key", apiKey).Msg("[ApiKeyMiddleware] DB tidak tersedia, validasi gagal")
					return c.JSON(http.StatusUnauthorized, models.Err(
						http.StatusUnauthorized,
						"API key tidak valid",
					))
				}
				return next(c)
			}

			// Cek ke database
			var record models.ApiKey
			result := f.db.GetDB().Where("key = ? AND active = ?", apiKey, true).First(&record)
			if result.Error != nil {
				if result.Error == gorm.ErrRecordNotFound {
					return c.JSON(http.StatusUnauthorized, models.Err(
						http.StatusUnauthorized,
						"API key tidak ditemukan atau tidak aktif",
					))
				}
				f.log.Error().Err(result.Error).Msg("[ApiKeyMiddleware] Query error")
				return c.JSON(http.StatusInternalServerError, models.Err(
					http.StatusInternalServerError,
					"Gagal memvalidasi API key",
				))
			}

			// Periksa IP whitelist
			clientIP := c.RealIP()
			if record.IPWhitelist != "" && !isIPAllowed(clientIP, record.IPWhitelist) {
				f.log.Warn().
					Str("ip", clientIP).
					Str("whitelist", record.IPWhitelist).
					Str("key_name", record.Name).
					Msg("[ApiKeyMiddleware] IP ditolak")
				return c.JSON(http.StatusForbidden, models.Err(
					http.StatusForbidden,
					"IP address tidak diizinkan untuk API key ini",
				))
			}

			// Simpan info ApiKey ke context
			c.Set(models.ApiKeyContextKey, &record)
			f.log.Debug().Str("key_name", record.Name).Str("ip", clientIP).
				Msg("[ApiKeyMiddleware] OK")

			return next(c)
		}
	}
}

// isIPAllowed memeriksa apakah ip ada dalam whitelist comma-separated.
func isIPAllowed(ip, whitelist string) bool {
	for _, allowed := range strings.Split(whitelist, ",") {
		if strings.TrimSpace(allowed) == ip {
			return true
		}
	}
	return false
}
