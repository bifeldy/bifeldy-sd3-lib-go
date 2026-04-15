# bifeldy-sd3-lib-go

Go library untuk mempercepat development API — migrasi dari [bifeldy_lib_90 (.NET 9)](https://github.com/bifeldy/bifeldy_lib_90).

Ditenagai **Echo v4** + **GORM** + **zerolog** + **robfig/cron**.

> Project baru hanya perlu membuat file `controller` + `service`, tanpa setup ulang HTTP server, logger, database, middleware, dsb.

---

## Mengapa Echo, bukan Fiber?

Fiber menggunakan `fasthttp` (bukan `net/http` standar Go). Ini menjadi masalah untuk:
- **Stream proxy zero-copy** — `io.Copy(w, resp.Body)` hanya bekerja mulus dengan `net/http`
- **Kompatibilitas library** — banyak library Go menggunakan `net/http` interface

**Echo v4** dipilih karena native `net/http`, API mirip .NET (Group, Use, middleware chaining), dan ekosistem lengkap.

---

## Fitur

| Fitur | Keterangan |
|---|---|
| **HTTP Server** | Echo v4, CORS, graceful shutdown |
| **JWT Middleware** | `Authorization: Bearer <token>`, klaim tersimpan di context |
| **ApiKey Middleware** | Header `X-Api-Key` + whitelist IP per-key |
| **Stream Proxy** | Forward HTTP tanpa buffer RAM (`io.Copy`) |
| **Cron Scheduler** | Fluent API, `ScheduleJob("*/5 * * * *").AddJob(...)` |
| **Daily Log File** | zerolog → file rolling per hari, hapus otomatis via job |
| **SQLite** | Pure Go (tanpa CGO), via `glebarez/sqlite` |
| **PostgreSQL** | via GORM + pgx driver |
| **MS SQL Server** | via GORM + sqlserver driver |
| **LockerService** | Per-key mutex untuk critical section |
| **HttpService** | GET/POST/PUT/DELETE wrapper + `ForwardStream` |
| **GlobalService** | `GetRealIP`, `IsIPInWhitelist`, `TruncateString`, dll |

---

## Instalasi Library

```bash
go get github.com/bifeldy/bifeldy-sd3-lib-go
```

---

## Struktur Project Baru (Minimal)

```
MyProject/
├── go.mod
├── .env
├── main.go              ← hanya ~40 baris
├── controllers/
│   ├── register.go      ← daftarkan semua controller di sini
│   └── foo_controller.go
└── services/
    └── foo_service.go
```

---

## Cara Pakai

### 1. `main.go`

```go
package main

import (
    "context"
    bifeldy "github.com/bifeldy/bifeldy-sd3-lib-go"
    "MyProject/controllers"
    "MyProject/services"
)

func main() {
    lib := bifeldy.New()              // muat .env, setup logger + echo
    lib.AddDependencyInjection()      // koneksi DB + services

    svc := services.NewFooService(lib)
    if err := svc.Migrate(); err != nil {
        lib.GetLogger().Error().Err(err).Msg("migrate gagal")
    }

    lib.StartJobScheduler()
    lib.ScheduleJob("0 * * * *").AddJob("HourlySync", func(ctx context.Context) error {
        return svc.HourlyJob(ctx)
    })

    api := lib.StartApiWithPrefix()   // default prefix: "api" dari .env
    controllers.RegisterAll(api, lib, svc)

    lib.Run()                         // blocking, graceful shutdown via Ctrl+C
}
```

### 2. `.env`

```env
APP_NAME=MyProject
PORT=8080
API_PREFIX=api
DEBUG=true

JWT_SECRET=ganti-dengan-secret-kuat
JWT_EXPIRE_HOUR=24

DB_SQLITE=_data/myproject.db
# DB_POSTGRES=host=localhost user=postgres password=secret dbname=mydb port=5432 sslmode=disable
# DB_MSSQL=sqlserver://sa:secret@localhost:1433?database=mydb

LOG_DIR=_data/logs
LOG_RETAIN_DAYS=30
```

### 3. Controller

```go
package controllers

import (
    "net/http"
    bifeldy "github.com/bifeldy/bifeldy-sd3-lib-go"
    "github.com/bifeldy/bifeldy-sd3-lib-go/models"
    "github.com/bifeldy/bifeldy-sd3-lib-go/middlewares"
    "github.com/labstack/echo/v4"
    "MyProject/services"
)

type FooController struct {
    lib *bifeldy.Bifeldy
    svc *services.FooService
}

func RegisterFooController(api *echo.Group, lib *bifeldy.Bifeldy, svc *services.FooService) {
    ctrl := &FooController{lib: lib, svc: svc}

    g := api.Group("/foo")
    g.Use(lib.Middleware.JWT())       // lindungi dengan JWT
    g.GET("", ctrl.GetList)
    g.POST("", ctrl.Create)
}

func (c *FooController) GetList(ctx echo.Context) error {
    session := middlewares.GetJwtSession(ctx)   // ambil user dari token
    items, err := c.svc.GetAll()
    if err != nil {
        return ctx.JSON(http.StatusInternalServerError, models.Err(500, err.Error()))
    }
    return ctx.JSON(http.StatusOK, models.OkList(items))
}
```

### 4. Middleware: ApiKey dengan IP Whitelist

```go
// Pasang ke route group tertentu
external := api.Group("/external")
external.Use(lib.Middleware.ApiKey())

// Data ApiKey disimpan di tabel api_keys (GORM auto-migrate):
// db.AutoMigrate(&models.ApiKey{})
//
// Kolom ip_whitelist: "192.168.1.1,10.0.0.5" — kosong = semua IP diizinkan
```

### 5. Stream Proxy (Zero Buffer)

```go
func (c *MyController) Proxy(ctx echo.Context) error {
    // Forward semua request (method + headers + body) ke target
    // Data mengalir chunk per chunk — tidak numpuk di RAM
    target := "http://backend:9000" + ctx.Request().URL.RequestURI()
    return c.lib.Http.ForwardStream(ctx, target, map[string]string{
        "Authorization": "Bearer internal-service-token",
    })
}
```

### 6. HTTP Client

```go
// GET biasa
resp, err := lib.Http.GET(ctx, "https://api.example.com/data", nil)

// POST JSON + unmarshal langsung
var result MyStruct
err := lib.Http.PostJSON(ctx, "https://api.example.com/submit", payload, &result, nil)
```

### 7. Scheduler

```go
lib.StartJobScheduler()

// Setiap 5 menit
lib.ScheduleJob("*/5 * * * *").AddJob("Quick", func(ctx context.Context) error {
    return svc.QuickJob(ctx)
})

// Chaining — dua job pada cron yang sama
lib.ScheduleJob("0 2 * * *").
    AddJob("CleanTemp", svc.CleanTempJob).
    AddJob("ReportDaily", svc.DailyReportJob)
```

---

## API Response Format

```go
// Single data
models.Ok(data)           // {"info":"200 - OK","result":{...}}

// List data
models.OkList(items)      // {"info":"200 - OK","count":5,"result":[...]}

// Error
models.Err(404, "pesan") // {"info":"404 - Error","result":{"message":"pesan"}}
```

---

## Development Lokal (replace directive)

Selama library belum di-push atau ingin test perubahan lokal:

```go
// DataDc/go.mod
replace github.com/bifeldy/bifeldy-sd3-lib-go => ../bifeldy-sd3-lib-go
```

Setelah library di-push ke GitHub dan ingin pakai versi publik:

```bash
# Hapus replace directive dari go.mod, lalu:
go get github.com/bifeldy/bifeldy-sd3-lib-go@latest
go mod tidy
```

---

## Logging

- **Console**: semua level (`DEBUG` jika `DEBUG=true`, default `INFO`)
- **File**: hanya `ERROR` ke atas → `_data/logs/error_YYYYMMDD.log`
- **Auto-cleanup**: job bawaan tiap tengah malam hapus file > `LOG_RETAIN_DAYS` hari

---

## Struktur Library

```
bifeldy-sd3-lib-go/
├── bifeldy.go           ← container utama: New(), AddDependencyInjection(), Run()
├── models/
│   ├── config.go        ← Config struct + LoadConfig() dari .env
│   ├── response.go      ← ResponseJsonSingle[T], ResponseJsonList[T], Ok(), OkList(), Err()
│   └── user.go          ← ApiKey, User (GORM), JwtClaims, JwtSession
├── logger/
│   └── logger.go        ← zerolog + daily rolling error file
├── databases/
│   ├── base.go          ← IGormDB interface
│   ├── sqlite.go        ← pure Go, tanpa CGO
│   ├── postgres.go
│   └── mssql.go
├── middlewares/
│   ├── factory.go       ← MiddlewareFactory
│   ├── api_key.go       ← X-Api-Key + IP whitelist
│   └── jwt.go           ← Bearer JWT + GetJwtSession()
├── services/
│   ├── application.go   ← AppName, IsDebug, Uptime
│   ├── http.go          ← GET/POST/PUT/DELETE + ForwardStream
│   ├── global.go        ← GetRealIP, IsIPInWhitelist, ContainsString
│   └── locker.go        ← per-key mutex Lock/Unlock/TryLock
└── scheduler/
    ├── scheduler.go      ← CronScheduler + ScheduleBuilder (fluent)
    └── cleanup.go        ← built-in log cleanup job
```

---

## License

GPL-3.0
