package services

import (
	"time"

	"github.com/bifeldy/bifeldy-sd3-lib-go/models"
	"github.com/rs/zerolog"
)

// IApplicationService menyediakan info dasar aplikasi.
type IApplicationService interface {
	AppName() string
	IsDebug() bool
	StartTime() time.Time
	Uptime() time.Duration
}

type applicationService struct {
	cfg       *models.Config
	log       *zerolog.Logger
	startTime time.Time
}

// NewApplicationService membuat instance ApplicationService.
func NewApplicationService(cfg *models.Config, log *zerolog.Logger) IApplicationService {
	return &applicationService{
		cfg:       cfg,
		log:       log,
		startTime: time.Now(),
	}
}

func (s *applicationService) AppName() string {
	return s.cfg.AppName
}

func (s *applicationService) IsDebug() bool {
	return s.cfg.Debug
}

func (s *applicationService) StartTime() time.Time {
	return s.startTime
}

func (s *applicationService) Uptime() time.Duration {
	return time.Since(s.startTime)
}
