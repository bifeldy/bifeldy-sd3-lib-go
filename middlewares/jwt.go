package middlewares

import (
	"net/http"
	"strings"

	"github.com/bifeldy/bifeldy-sd3-lib-go/models"
	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

// JwtMiddleware memvalidasi Bearer token JWT dari header Authorization.
//
// Alur:
//  1. Ambil "Authorization: Bearer <token>"
//  2. Parse dan validasi signature dengan JWTSecret dari config
//  3. Simpan JwtSession ke context dengan key models.JwtContextKey
func (f *MiddlewareFactory) JWT() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return c.JSON(http.StatusUnauthorized, models.Err(
					http.StatusUnauthorized,
					"Authorization header wajib diisi",
				))
			}

			parts := strings.SplitN(authHeader, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return c.JSON(http.StatusUnauthorized, models.Err(
					http.StatusUnauthorized,
					"Format Authorization: Bearer <token>",
				))
			}

			tokenStr := parts[1]
			claims := &models.JwtClaims{}

			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, echo.ErrUnauthorized
				}
				return []byte(f.cfg.JWTSecret), nil
			})

			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, models.Err(
					http.StatusUnauthorized,
					"Token tidak valid atau sudah kadaluarsa",
				))
			}

			// Simpan session ke context
			session := &models.JwtSession{
				UserID:   claims.UserID,
				Username: claims.Username,
				Role:     claims.Role,
			}
			c.Set(models.JwtContextKey, session)

			f.log.Debug().
				Uint("user_id", session.UserID).
				Str("username", session.Username).
				Msg("[JwtMiddleware] OK")

			return next(c)
		}
	}
}

// GetJwtSession mengambil JwtSession dari echo.Context.
// Mengembalikan nil jika JWT middleware tidak dipasang atau token tidak ada.
func GetJwtSession(c echo.Context) *models.JwtSession {
	if v := c.Get(models.JwtContextKey); v != nil {
		if session, ok := v.(*models.JwtSession); ok {
			return session
		}
	}
	return nil
}

// GetApiKey mengambil ApiKey dari echo.Context.
func GetApiKey(c echo.Context) *models.ApiKey {
	if v := c.Get(models.ApiKeyContextKey); v != nil {
		if key, ok := v.(*models.ApiKey); ok {
			return key
		}
	}
	return nil
}
