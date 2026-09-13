package pacing

import (
	"math/rand"
	"sync"
	"time"

	"github.com/techboy365/Parser/internal/config"
)

type Limiter struct {
	cfg       config.PacingConfig
	mu        sync.Mutex
	searchTS  []time.Time
	rng       *rand.Rand
}

func NewLimiter(cfg config.PacingConfig) *Limiter {
	return &Limiter{
		cfg: cfg,
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (l *Limiter) BeforeAction() {
	if !l.cfg.Enabled {
		return
	}
	min := l.cfg.MinActionDelayMS
	max := l.cfg.MaxActionDelayMS
	if max < min {
		max = min
	}
	delay := min
	if max > min {
		delay = min + l.rng.Intn(max-min+1)
	}
	time.Sleep(time.Duration(delay) * time.Millisecond)
}

func (l *Limiter) AllowSearch() error {
	if !l.cfg.Enabled || l.cfg.MaxSearchesPerHour <= 0 {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := time.Now().Add(-1 * time.Hour)
	kept := make([]time.Time, 0, len(l.searchTS))
	for _, ts := range l.searchTS {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	l.searchTS = kept
	if len(l.searchTS) >= l.cfg.MaxSearchesPerHour {
		return ErrRateLimited
	}
	l.searchTS = append(l.searchTS, time.Now())
	return nil
}

var ErrRateLimited = errRateLimited{}

type errRateLimited struct{}

func (errRateLimited) Error() string {
	return "search rate limit reached for current hour; wait before continuing"
}
