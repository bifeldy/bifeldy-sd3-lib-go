package services

import (
	"sync"

	"github.com/rs/zerolog"
)

// ILockerService menyediakan per-key mutex locking dalam satu proses.
// Berguna untuk mencegah race condition pada operasi kritis.
// Catatan: ini proses-lokal; untuk multi-instance pakai Redis-based lock.
type ILockerService interface {
	// Lock mengunci key. Blokir sampai kunci tersedia.
	Lock(key string)
	// Unlock melepas kunci untuk key.
	Unlock(key string)
	// TryLock mencoba mengunci key. Mengembalikan false jika sudah terkunci.
	TryLock(key string) bool
}

type lockerService struct {
	log  *zerolog.Logger
	mu   sync.Mutex          // melindungi map keys
	keys map[string]*sync.Mutex
}

// NewLockerService membuat instance LockerService.
func NewLockerService(log *zerolog.Logger) ILockerService {
	return &lockerService{
		log:  log,
		keys: make(map[string]*sync.Mutex),
	}
}

// getOrCreate memastikan mutex untuk key sudah ada.
func (s *lockerService) getOrCreate(key string) *sync.Mutex {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keys[key]; !ok {
		s.keys[key] = &sync.Mutex{}
	}
	return s.keys[key]
}

func (s *lockerService) Lock(key string) {
	s.getOrCreate(key).Lock()
}

func (s *lockerService) Unlock(key string) {
	s.getOrCreate(key).Unlock()
}

func (s *lockerService) TryLock(key string) bool {
	return s.getOrCreate(key).TryLock()
}
