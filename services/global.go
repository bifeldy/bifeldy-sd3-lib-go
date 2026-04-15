package services

import (
	"net"
	"net/http"
	"strings"

	"github.com/bifeldy/bifeldy-sd3-lib-go/models"
	"github.com/rs/zerolog"
)

// IGlobalService menyediakan utilitas umum yang sering dipakai.
type IGlobalService interface {
	// GetRealIP mengambil IP client asli dari request,
	// mempertimbangkan X-Real-IP dan X-Forwarded-For (dari reverse proxy).
	GetRealIP(r *http.Request) string

	// IsIPInWhitelist memeriksa apakah ip ada dalam whitelist (comma-separated).
	// Jika whitelist kosong string, semua IP diizinkan.
	IsIPInWhitelist(ip, whitelist string) bool

	// ContainsString memeriksa apakah slice s mengandung string v.
	ContainsString(s []string, v string) bool

	// TruncateString memotong string sepanjang maxLen karakter.
	TruncateString(s string, maxLen int) string
}

type globalService struct {
	cfg *models.Config
	log *zerolog.Logger
}

// NewGlobalService membuat instance GlobalService.
func NewGlobalService(cfg *models.Config, log *zerolog.Logger) IGlobalService {
	return &globalService{cfg: cfg, log: log}
}

func (s *globalService) GetRealIP(r *http.Request) string {
	// Dari header reverse proxy (nginx, dll)
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return strings.TrimSpace(ip)
	}
	if forwarded := r.Header.Get("X-Forwarded-For"); forwarded != "" {
		// Format: "client, proxy1, proxy2" → ambil yang pertama
		parts := strings.Split(forwarded, ",")
		return strings.TrimSpace(parts[0])
	}
	// Fallback: RemoteAddr langsung
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return ip
}

func (s *globalService) IsIPInWhitelist(ip, whitelist string) bool {
	whitelist = strings.TrimSpace(whitelist)
	if whitelist == "" {
		return true // kosong = semua diizinkan
	}
	for _, allowed := range strings.Split(whitelist, ",") {
		if strings.TrimSpace(allowed) == ip {
			return true
		}
	}
	return false
}

func (s *globalService) ContainsString(sl []string, v string) bool {
	for _, item := range sl {
		if item == v {
			return true
		}
	}
	return false
}

func (s *globalService) TruncateString(str string, maxLen int) string {
	runes := []rune(str)
	if len(runes) <= maxLen {
		return str
	}
	return string(runes[:maxLen]) + "..."
}
