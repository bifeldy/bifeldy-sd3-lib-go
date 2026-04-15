package bifeldy

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/bifeldy/bifeldy-sd3-lib-go/databases"
	"github.com/bifeldy/bifeldy-sd3-lib-go/logger"
	"github.com/bifeldy/bifeldy-sd3-lib-go/middlewares"
	"github.com/bifeldy/bifeldy-sd3-lib-go/models"
	"github.com/bifeldy/bifeldy-sd3-lib-go/scheduler"
	"github.com/bifeldy/bifeldy-sd3-lib-go/services"
	"github.com/labstack/echo/v4"
	echomw "github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

const (
	DefaultDataFolder = "_data"
)

// ===========================================================================
// Bifeldy — container utama library
// ===========================================================================

// Bifeldy adalah container utama yang menyatukan seluruh komponen library.
// Di project baru, cukup panggil bifeldy.New() lalu daftarkan controller.
type Bifeldy struct {
	echo      *echo.Echo
	config    *models.Config
	log       *zerolog.Logger
	sched     *scheduler.CronScheduler
	apiPrefix string

	// Middleware factory — akses melalui lib.Middleware.ApiKey(), lib.Middleware.JWT()
	Middleware *middlewares.MiddlewareFactory

	// Services — dapat diinjeksikan ke controller / service project
	App    services.IApplicationService
	Http   services.IHttpService
	Global services.IGlobalService
	Locker services.ILockerService

	// Databases — gunakan .GetDB() untuk akses GORM fluent API
	Sqlite   databases.IGormDB
	Postgres databases.IGormDB
	MsSQL    databases.IGormDB
}

// ===========================================================================
// Constructor
// ===========================================================================

// New membuat instance Bifeldy baru.
//   - envFiles: path ke file .env (default: ".env")
//   - Variabel OS yang sudah ada TIDAK akan ditimpa
//
// Contoh:
//
//	lib := bifeldy.New()           // pakai .env
//	lib := bifeldy.New("prod.env") // pakai file lain
func New(envFiles ...string) *Bifeldy {
	// 1. Load config dari .env
	cfg := models.LoadConfig(envFiles...)

	// 2. Buat direktori data
	_ = os.MkdirAll(DefaultDataFolder, 0o755)

	// 3. Setup logger (console + daily error file)
	log := logger.NewLogger(cfg)
	log.Info().Str("app", cfg.AppName).Str("port", cfg.Port).Msg("=> Bifeldy starting")

	// 4. Setup Echo
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	// Middleware global bawaan
	e.Use(echomw.Recover())
	e.Use(echomw.RequestID())

	b := &Bifeldy{
		echo:      e,
		config:    cfg,
		log:       log,
		apiPrefix: cfg.APIPrefix,
	}

	// 5. Middleware factory (DB di-set nanti via AddDependencyInjection)
	b.Middleware = middlewares.NewMiddlewareFactory(cfg, log)

	return b
}

// ===========================================================================
// Dependency Injection
// ===========================================================================

// AddDependencyInjection menginisialisasi semua database dan service.
// Panggil sebelum StartApiWithPrefix() dan StartJobScheduler().
func (b *Bifeldy) AddDependencyInjection() *Bifeldy {
	// Databases
	b.Sqlite = databases.NewSQLite(b.config, b.log)
	b.Postgres = databases.NewPostgres(b.config, b.log)
	b.MsSQL = databases.NewMsSQL(b.config, b.log)

	// Set DB ke middleware factory (untuk ApiKey validation)
	// Prioritas: Postgres → MsSQL → Sqlite
	switch {
	case b.Postgres.IsConnected():
		b.Middleware.SetDB(b.Postgres)
	case b.MsSQL.IsConnected():
		b.Middleware.SetDB(b.MsSQL)
	case b.Sqlite.IsConnected():
		b.Middleware.SetDB(b.Sqlite)
	}

	// Services
	b.App = services.NewApplicationService(b.config, b.log)
	b.Http = services.NewHttpService(b.config, b.log)
	b.Global = services.NewGlobalService(b.config, b.log)
	b.Locker = services.NewLockerService(b.log)

	b.log.Info().Msg("=> Dependency injection selesai")
	return b
}

// ===========================================================================
// Job Scheduler
// ===========================================================================

// StartJobScheduler mengaktifkan cron scheduler.
// Job built-in: cleanup log file harian (tengah malam).
// Panggil sebelum Run().
func (b *Bifeldy) StartJobScheduler() *Bifeldy {
	b.sched = scheduler.NewCronScheduler(b.log)

	// Job bawaan: hapus log lama setiap tengah malam
	b.sched.Schedule("0 0 * * *").AddJob(
		"CleanupLogs",
		scheduler.CleanupLogsJob(b.config, b.log),
	)

	b.sched.Start()
	return b
}

// ScheduleJob mengembalikan ScheduleBuilder untuk menambah job custom.
// Contoh:
//
//	lib.ScheduleJob("*/5 * * * *").AddJob("SyncData", mySvc.SyncJob)
func (b *Bifeldy) ScheduleJob(cronExpr string) *scheduler.ScheduleBuilder {
	if b.sched == nil {
		b.log.Fatal().Msg("Panggil StartJobScheduler() sebelum ScheduleJob()")
	}
	return b.sched.Schedule(cronExpr)
}

// ===========================================================================
// HTTP Server
// ===========================================================================

// StartApiWithPrefix mendaftarkan route group dengan prefix API dan
// mengatur handler error global.
// Mengembalikan *echo.Group untuk registrasi controller.
//
// Contoh:
//
//	api := lib.StartApiWithPrefix()
//	controllers.RegisterAll(api, lib)
func (b *Bifeldy) StartApiWithPrefix(prefix ...string) *echo.Group {
	if len(prefix) > 0 && prefix[0] != "" {
		b.apiPrefix = prefix[0]
	}

	// CORS global
	b.echo.Use(echomw.CORSWithConfig(echomw.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowMethods: []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete, http.MethodOptions,
		},
		AllowHeaders: []string{
			echo.HeaderOrigin, echo.HeaderContentType,
			echo.HeaderAccept, echo.HeaderAuthorization,
			"X-Api-Key", "X-Request-ID",
		},
	}))

	// Global error handler — semua error dikembalikan sebagai JSON
	b.echo.HTTPErrorHandler = func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}
		code := http.StatusInternalServerError
		var msg string
		if he, ok := err.(*echo.HTTPError); ok {
			code = he.Code
			msg = fmt.Sprintf("%v", he.Message)
		} else {
			msg = err.Error()
		}
		_ = c.JSON(code, models.Err(code, msg))
	}

	// Redirect root "/" → "/<prefix>"
	b.echo.GET("/", func(c echo.Context) error {
		return c.Redirect(http.StatusPermanentRedirect, "/"+b.apiPrefix)
	})

	// Route group utama
	group := b.echo.Group("/" + b.apiPrefix)

	// Default info endpoint di /<prefix>
	group.GET("", func(c echo.Context) error {
		return c.JSON(http.StatusOK, models.Ok(map[string]any{
			"app":     b.config.AppName,
			"status":  "running",
			"uptime":  b.App.Uptime().String(),
			"version": "1.0.0",
		}))
	})

	b.log.Info().Msgf("=> API prefix: /%s", b.apiPrefix)
	return group
}

// ===========================================================================
// Getters — akses internal untuk controller / service di project
// ===========================================================================

// GetEcho mengembalikan instance *echo.Echo untuk konfigurasi lanjutan.
func (b *Bifeldy) GetEcho() *echo.Echo { return b.echo }

// GetConfig mengembalikan Config yang sudah dimuat.
func (b *Bifeldy) GetConfig() *models.Config { return b.config }

// GetLogger mengembalikan zerolog.Logger.
func (b *Bifeldy) GetLogger() *zerolog.Logger { return b.log }

// ===========================================================================
// Run — start server dengan graceful shutdown
// ===========================================================================

// Run menjalankan HTTP server dan menunggu sinyal interrupt (Ctrl+C).
// Graceful shutdown: tunggu request aktif selesai, hentikan scheduler.
func (b *Bifeldy) Run() {
	port := b.config.Port
	if port == "" {
		port = "8080"
	}

	b.log.Info().Msgf("=> Server berjalan di http://localhost:%s", port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	go func() {
		if err := b.echo.Start(":" + port); err != nil && err != http.ErrServerClosed {
			b.log.Fatal().Err(err).Msg("Server error")
		}
	}()

	<-ctx.Done()
	b.log.Info().Msg("=> Shutdown signal diterima...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if b.sched != nil {
		b.sched.Stop()
	}

	if err := b.echo.Shutdown(shutdownCtx); err != nil {
		b.log.Error().Err(err).Msg("Server shutdown error")
	}

	b.log.Info().Msg("=> Server berhenti. Sampai jumpa!")
}
