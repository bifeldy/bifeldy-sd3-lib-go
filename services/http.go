package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/bifeldy/bifeldy-sd3-lib-go/models"
	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog"
)

// IHttpService menyediakan HTTP client wrapper dan stream proxy.
type IHttpService interface {
	// GET melakukan HTTP GET ke url dengan headers opsional.
	GET(ctx context.Context, url string, headers map[string]string) (*http.Response, error)

	// POST melakukan HTTP POST dengan body JSON.
	POST(ctx context.Context, url string, body any, headers map[string]string) (*http.Response, error)

	// PUT melakukan HTTP PUT dengan body JSON.
	PUT(ctx context.Context, url string, body any, headers map[string]string) (*http.Response, error)

	// DELETE melakukan HTTP DELETE.
	DELETE(ctx context.Context, url string, headers map[string]string) (*http.Response, error)

	// GetJSON melakukan GET dan langsung unmarshal JSON ke dest.
	GetJSON(ctx context.Context, url string, dest any, headers map[string]string) error

	// PostJSON melakukan POST dan langsung unmarshal JSON ke dest.
	PostJSON(ctx context.Context, url string, body any, dest any, headers map[string]string) error

	// ForwardStream meneruskan request dari echo.Context ke targetURL secara streaming.
	// Data mengalir langsung tanpa di-buffer ke RAM — cocok untuk file besar / SSE.
	// Method, body, dan headers dari request asal ikut diteruskan.
	ForwardStream(c echo.Context, targetURL string, extraHeaders map[string]string) error
}

type httpService struct {
	cfg    *models.Config
	log    *zerolog.Logger
	client *http.Client
}

// NewHttpService membuat instance HttpService dengan timeout 30 detik.
func NewHttpService(cfg *models.Config, log *zerolog.Logger) IHttpService {
	return &httpService{
		cfg: cfg,
		log: log,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ===========================================================================
// Internal helper
// ===========================================================================

func (s *httpService) do(ctx context.Context, method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("buat request %s %s: %w", method, url, err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("eksekusi %s %s: %w", method, url, err)
	}
	return resp, nil
}

func marshalBody(body any) (io.Reader, error) {
	if body == nil {
		return nil, nil
	}
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(b), nil
}

// ===========================================================================
// Public methods
// ===========================================================================

func (s *httpService) GET(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	return s.do(ctx, http.MethodGet, url, nil, headers)
}

func (s *httpService) POST(ctx context.Context, url string, body any, headers map[string]string) (*http.Response, error) {
	r, err := marshalBody(body)
	if err != nil {
		return nil, err
	}
	return s.do(ctx, http.MethodPost, url, r, headers)
}

func (s *httpService) PUT(ctx context.Context, url string, body any, headers map[string]string) (*http.Response, error) {
	r, err := marshalBody(body)
	if err != nil {
		return nil, err
	}
	return s.do(ctx, http.MethodPut, url, r, headers)
}

func (s *httpService) DELETE(ctx context.Context, url string, headers map[string]string) (*http.Response, error) {
	return s.do(ctx, http.MethodDelete, url, nil, headers)
}

func (s *httpService) GetJSON(ctx context.Context, url string, dest any, headers map[string]string) error {
	resp, err := s.GET(ctx, url, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(dest)
}

func (s *httpService) PostJSON(ctx context.Context, url string, body any, dest any, headers map[string]string) error {
	resp, err := s.POST(ctx, url, body, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(dest)
}

// ForwardStream — ZERO BUFFER streaming proxy.
//
// Cara kerja:
//  1. Buat request baru ke targetURL dengan method + body dari request masuk
//  2. Copy semua request headers dari client
//  3. Eksekusi ke target server
//  4. Copy response headers ke client
//  5. io.Copy: stream body langsung dari target → client tanpa buffer RAM
//
// Gunakan untuk: file download, SSE, large JSON, export Excel, dll.
func (s *httpService) ForwardStream(c echo.Context, targetURL string, extraHeaders map[string]string) error {
	inReq := c.Request()

	// Buat proxy request — body langsung dari request masuk (streaming)
	proxyReq, err := http.NewRequestWithContext(inReq.Context(), inReq.Method, targetURL, inReq.Body)
	if err != nil {
		return fmt.Errorf("buat proxy request: %w", err)
	}

	// Copy semua request headers dari client ke target
	for key, values := range inReq.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Override / tambah header ekstra
	for k, v := range extraHeaders {
		proxyReq.Header.Set(k, v)
	}

	// Gunakan client tanpa timeout untuk streaming panjang
	streamClient := &http.Client{}
	resp, err := streamClient.Do(proxyReq)
	if err != nil {
		return fmt.Errorf("proxy request gagal: %w", err)
	}
	defer resp.Body.Close()

	// Copy response headers dari target ke client
	for key, values := range resp.Header {
		for _, value := range values {
			c.Response().Header().Add(key, value)
		}
	}

	// Set status code
	c.Response().WriteHeader(resp.StatusCode)

	// Stream body — io.Copy mengalirkan data dalam chunk kecil tanpa buffer penuh
	if _, err = io.Copy(c.Response(), resp.Body); err != nil {
		s.log.Warn().Err(err).Str("target", targetURL).Msg("[HttpService] Stream terputus")
	}

	return nil
}
