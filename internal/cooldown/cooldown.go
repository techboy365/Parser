package cooldown

import (
	"sync"
	"time"
)

type Gate struct {
	mu       sync.RWMutex
	blocked  bool
	until    time.Time
	reason   string
}

func NewGate() *Gate {
	return &Gate{}
}

func (g *Gate) Block(duration time.Duration, reason string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.blocked = true
	g.until = time.Now().Add(duration)
	g.reason = reason
}

func (g *Gate) Clear() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.blocked = false
	g.until = time.Time{}
	g.reason = ""
}

func (g *Gate) Check() error {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if !g.blocked {
		return nil
	}
	if time.Now().After(g.until) {
		return nil
	}
	remaining := time.Until(g.until).Round(time.Second)
	return &BlockedError{Reason: g.reason, Remaining: remaining}
}

type BlockedError struct {
	Reason    string
	Remaining time.Duration
}

func (e *BlockedError) Error() string {
	return "connection cooldown active (" + e.Reason + "), retry in " + e.Remaining.String()
}
