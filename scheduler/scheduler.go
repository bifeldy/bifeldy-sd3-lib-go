package scheduler

import (
	"context"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
)

// ===========================================================================
// CronScheduler
// ===========================================================================

// CronScheduler membungkus robfig/cron dengan logging otomatis.
type CronScheduler struct {
	c   *cron.Cron
	log *zerolog.Logger
}

// NewCronScheduler membuat instance CronScheduler baru.
// Menggunakan cron 5-field standar: "menit jam hari-bulan bulan hari-minggu"
func NewCronScheduler(log *zerolog.Logger) *CronScheduler {
	c := cron.New(cron.WithLogger(cron.DiscardLogger))
	return &CronScheduler{c: c, log: log}
}

// Schedule mengembalikan ScheduleBuilder untuk cron expression tertentu.
// Contoh: scheduler.Schedule("0 0 * * *") → setiap hari tengah malam
// Contoh: scheduler.Schedule("*/5 * * * *") → setiap 5 menit
func (s *CronScheduler) Schedule(expr string) *ScheduleBuilder {
	return &ScheduleBuilder{expr: expr, cs: s}
}

// Start menjalankan scheduler di background goroutine.
func (s *CronScheduler) Start() {
	s.c.Start()
	s.log.Info().Msg("[Scheduler] Berjalan")
}

// Stop menghentikan scheduler dengan graceful (menunggu job aktif selesai).
func (s *CronScheduler) Stop() {
	ctx := s.c.Stop()
	<-ctx.Done()
	s.log.Info().Msg("[Scheduler] Berhenti")
}

// ===========================================================================
// ScheduleBuilder — fluent API untuk menambah job
// ===========================================================================

// ScheduleBuilder membangun daftar job untuk satu cron expression.
type ScheduleBuilder struct {
	expr string
	cs   *CronScheduler
}

// AddJob mendaftarkan sebuah job function ke cron expression ini.
// name: nama job untuk logging
// fn: function yang dijalankan; kembalikan error jika gagal
//
// Bisa chaining: Schedule("* * * * *").AddJob("a", fn1).AddJob("b", fn2)
func (sb *ScheduleBuilder) AddJob(name string, fn func(ctx context.Context) error) *ScheduleBuilder {
	log := sb.cs.log
	expr := sb.expr

	_, err := sb.cs.c.AddFunc(expr, func() {
		log.Info().Str("job", name).Str("cron", expr).Msg("[Scheduler] Mulai")
		if jobErr := fn(context.Background()); jobErr != nil {
			log.Error().Err(jobErr).Str("job", name).Msg("[Scheduler] Error")
		} else {
			log.Info().Str("job", name).Msg("[Scheduler] Selesai")
		}
	})

	if err != nil {
		log.Error().Err(err).Str("job", name).Str("cron", expr).
			Msg("[Scheduler] Gagal daftar job")
	} else {
		log.Info().Str("job", name).Str("cron", expr).Msg("[Scheduler] Job terdaftar")
	}

	return sb
}
