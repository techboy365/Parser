package session

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/techboy365/Parser/internal/browser"
	"github.com/techboy365/Parser/internal/config"
	"github.com/techboy365/Parser/internal/kameleo"
)

type Lease struct {
	ID        string
	ProfileID string
	Session   *browser.Session
	createdAt time.Time
	searches  int
	captchas  int
}

type Manager struct {
	cfg      *config.Config
	kameleo  *kameleo.Client
	browser  *browser.Manager
	log      *slog.Logger
	mu       sync.Mutex
	active   map[string]*Lease
}

func NewManager(cfg *config.Config, client *kameleo.Client, browserMgr *browser.Manager, log *slog.Logger) *Manager {
	return &Manager{
		cfg:     cfg,
		kameleo: client,
		browser: browserMgr,
		log:     log,
		active:  make(map[string]*Lease),
	}
}

func (m *Manager) Acquire(ctx context.Context, requestID string) (*Lease, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if lease, ok := m.active[requestID]; ok {
		return lease, nil
	}

	profile, err := m.createProfile(ctx, requestID)
	if err != nil {
		return nil, err
	}

	session, err := m.browser.Connect(ctx, profile.ID)
	if err != nil {
		_ = m.kameleo.StopProfile(ctx, profile.ID)
		_ = m.kameleo.DeleteProfile(ctx, profile.ID)
		return nil, err
	}

	lease := &Lease{
		ID:        uuid.NewString(),
		ProfileID: profile.ID,
		Session:   session,
		createdAt: time.Now().UTC(),
	}
	m.active[requestID] = lease
	m.log.Info("session acquired", "request_id", requestID, "profile_id", profile.ID, "lease_id", lease.ID)
	return lease, nil
}

func (m *Manager) Release(ctx context.Context, requestID string, forceRotate bool) error {
	m.mu.Lock()
	lease, ok := m.active[requestID]
	if !ok {
		m.mu.Unlock()
		return nil
	}
	delete(m.active, requestID)
	m.mu.Unlock()

	return m.closeLease(ctx, lease, forceRotate)
}

func (m *Manager) ShouldRotate(lease *Lease) bool {
	if lease.captchas >= m.cfg.Session.RotateProfileAfterCaptchas {
		return true
	}
	if lease.searches >= m.cfg.Session.RotateProfileAfterSearches {
		return true
	}
	ttl := time.Duration(m.cfg.Session.IdleProfileTTLMinutes) * time.Minute
	if ttl > 0 && time.Since(lease.createdAt) > ttl {
		return true
	}
	return false
}

func (m *Manager) RecordSearch(lease *Lease) {
	lease.searches++
}

func (m *Manager) RecordCaptcha(lease *Lease) {
	lease.captchas++
}

func (m *Manager) Rotate(ctx context.Context, requestID string) (*Lease, error) {
	if err := m.Release(ctx, requestID, true); err != nil {
		return nil, err
	}
	return m.Acquire(ctx, requestID)
}

func (m *Manager) createProfile(ctx context.Context, requestID string) (*kameleo.Profile, error) {
	fps, err := m.kameleo.SearchFingerprints(ctx, m.cfg.Kameleo.DeviceType, m.cfg.Kameleo.BrowserProduct)
	if err != nil {
		return nil, err
	}
	name := fmt.Sprintf("%s-%s", m.cfg.Kameleo.ProfileNamePrefix, requestID[:8])
	return m.kameleo.CreateProfile(ctx, kameleo.CreateProfileRequest{
		FingerprintID: fps[0].ID,
		Name:          name,
	})
}

func (m *Manager) closeLease(ctx context.Context, lease *Lease, deleteProfile bool) error {
	if lease.Session != nil {
		if err := lease.Session.Close(); err != nil {
			m.log.Warn("close browser session", "profile_id", lease.ProfileID, "error", err)
		}
	}
	if err := m.kameleo.StopProfile(ctx, lease.ProfileID); err != nil {
		m.log.Warn("stop profile", "profile_id", lease.ProfileID, "error", err)
	}
	if deleteProfile || !m.cfg.Session.ReuseProfiles {
		if err := m.kameleo.DeleteProfile(ctx, lease.ProfileID); err != nil {
			m.log.Warn("delete profile", "profile_id", lease.ProfileID, "error", err)
		}
	}
	return nil
}

func (m *Manager) Shutdown(ctx context.Context) {
	m.mu.Lock()
	leases := make([]*Lease, 0, len(m.active))
	for _, lease := range m.active {
		leases = append(leases, lease)
	}
	m.active = make(map[string]*Lease)
	m.mu.Unlock()

	for _, lease := range leases {
		_ = m.closeLease(ctx, lease, false)
	}
}
