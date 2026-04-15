package middlewares

import (
	"github.com/bifeldy/bifeldy-sd3-lib-go/databases"
	"github.com/bifeldy/bifeldy-sd3-lib-go/models"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

// MiddlewareFactory adalah factory yang memproduksi semua middleware library.
// Dibuat satu kali di Bifeldy.New(), DB di-set setelah AddDependencyInjection().
type MiddlewareFactory struct {
	cfg *models.Config
	log *zerolog.Logger
	db  databases.IGormDB // opsional: untuk validasi ApiKey dari DB
}

// NewMiddlewareFactory membuat MiddlewareFactory baru.
func NewMiddlewareFactory(cfg *models.Config, log *zerolog.Logger) *MiddlewareFactory {
	return &MiddlewareFactory{cfg: cfg, log: log}
}

// SetDB menetapkan database yang akan digunakan oleh middleware (ApiKey, dll).
// Panggil setelah AddDependencyInjection().
func (f *MiddlewareFactory) SetDB(db databases.IGormDB) *MiddlewareFactory {
	f.db = db
	return f
}

// RequestLogger middleware yang mencatat setiap request masuk.
// Format: METHOD PATH status latency ip
func (f *MiddlewareFactory) RequestLogger() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			err := next(c)
			req := c.Request()
			res := c.Response()
			f.log.Info().
				Str("method", req.Method).
				Str("path", req.URL.Path).
				Int("status", res.Status).
				Str("ip", c.RealIP()).
				Msg("[HTTP]")
			return err
		}
	}
}
