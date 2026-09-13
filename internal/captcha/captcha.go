package captcha

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/techboy365/Parser/internal/browser"
	"github.com/techboy365/Parser/internal/config"
)

type ChallengeType string

const (
	ChallengeNone        ChallengeType = "none"
	ChallengeRecaptcha   ChallengeType = "recaptcha"
	ChallengeTurnstile   ChallengeType = "turnstile"
	ChallengeGoogleSorry ChallengeType = "google_sorry"
	ChallengeUnknown     ChallengeType = "unknown"
)

type Detection struct {
	Detected bool
	Type     ChallengeType
	Reason   string
}

type Solver interface {
	Name() string
	Solve(ctx context.Context, session *browser.Session, detection Detection) error
}

type Detector struct {
	timeout time.Duration
}

func NewDetector(cfg config.CaptchaConfig) *Detector {
	return &Detector{timeout: time.Duration(cfg.DetectTimeoutMS) * time.Millisecond}
}

func (d *Detector) Detect(session *browser.Session) (Detection, error) {
	url := strings.ToLower(session.CurrentURL())
	title, _ := session.Title()
	content, err := session.Content()
	if err != nil {
		return Detection{}, err
	}
	lower := strings.ToLower(content + " " + title + " " + url)

	switch {
	case strings.Contains(url, "/sorry/") || strings.Contains(lower, "unusual traffic"):
		return Detection{Detected: true, Type: ChallengeGoogleSorry, Reason: "google sorry page"}, nil
	case strings.Contains(lower, "recaptcha") || strings.Contains(lower, "g-recaptcha"):
		return Detection{Detected: true, Type: ChallengeRecaptcha, Reason: "recaptcha markers"}, nil
	case strings.Contains(lower, "turnstile") || strings.Contains(lower, "cf-challenge"):
		return Detection{Detected: true, Type: ChallengeTurnstile, Reason: "turnstile/cloudflare markers"}, nil
	case strings.Contains(lower, "captcha"):
		return Detection{Detected: true, Type: ChallengeUnknown, Reason: "generic captcha marker"}, nil
	default:
		return Detection{Detected: false, Type: ChallengeNone}, nil
	}
}

type NoopSolver struct{}

func (NoopSolver) Name() string { return "noop" }

func (NoopSolver) Solve(ctx context.Context, session *browser.Session, detection Detection) error {
	return fmt.Errorf("captcha detected (%s): configure captcha.solver=sidecar and run captcha-solver", detection.Type)
}

type ManualSolver struct {
	Wait time.Duration
}

func (m ManualSolver) Name() string { return "manual" }

func (m ManualSolver) Solve(ctx context.Context, session *browser.Session, detection Detection) error {
	deadline := time.Now().Add(m.Wait)
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		d, err := NewDetector(config.CaptchaConfig{DetectTimeoutMS: 1000}).Detect(session)
		if err != nil {
			return err
		}
		if !d.Detected {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	return fmt.Errorf("manual captcha wait timed out (%s)", detection.Type)
}

type SidecarSolver struct {
	endpoint   string
	kameleoURL string
	timeout    time.Duration
	tiers      config.CaptchaTiersConfig
	httpClient *http.Client
}

type sidecarRequest struct {
	URL           string `json:"url"`
	Type          string `json:"type"`
	ProfileID     string `json:"profile_id"`
	CDPEndpoint   string `json:"cdp_endpoint"`
	KameleoBase   string `json:"kameleo_base"`
	EnableScript  bool   `json:"enable_script"`
	EnableToken   bool   `json:"enable_token"`
	EnableProvider bool  `json:"enable_provider"`
}

type sidecarResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Tier    string `json:"tier,omitempty"`
}

func NewSidecarSolver(cfg config.CaptchaConfig, kameleoEndpoint string) SidecarSolver {
	return SidecarSolver{
		endpoint:   cfg.SidecarEndpoint,
		kameleoURL: kameleoEndpoint,
		timeout:    cfgSidecarTimeout(cfg),
		tiers:      cfg.Tiers,
		httpClient: &http.Client{Timeout: cfgSidecarTimeout(cfg)},
	}
}

func cfgSidecarTimeout(cfg config.CaptchaConfig) time.Duration {
	if cfg.SidecarTimeoutMS <= 0 {
		return 180 * time.Second
	}
	return time.Duration(cfg.SidecarTimeoutMS) * time.Millisecond
}

func (s SidecarSolver) Name() string { return "sidecar" }

func (s SidecarSolver) Solve(ctx context.Context, session *browser.Session, detection Detection) error {
	currentURL := session.CurrentURL()
	if err := session.Suspend(); err != nil {
		return fmt.Errorf("suspend session for sidecar: %w", err)
	}

	payload, err := json.Marshal(sidecarRequest{
		URL:            currentURL,
		Type:           string(detection.Type),
		ProfileID:      session.ProfileID,
		CDPEndpoint:    session.CDPEndpoint,
		KameleoBase:    s.kameleoURL,
		EnableScript:   s.tiers.Script,
		EnableToken:    s.tiers.Token,
		EnableProvider: s.tiers.Provider,
	})
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.endpoint, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sidecar solver request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return fmt.Errorf("sidecar solver returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	var out sidecarResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return fmt.Errorf("decode sidecar response: %w", err)
	}
	if !out.Success {
		return fmt.Errorf("sidecar solver failed: %s", out.Message)
	}
	return nil
}

func NewSolver(cfg config.CaptchaConfig, kameleoEndpoint string) Solver {
	switch strings.ToLower(cfg.Solver) {
	case "sidecar":
		return NewSidecarSolver(cfg, kameleoEndpoint)
	case "manual":
		return ManualSolver{Wait: 120 * time.Second}
	default:
		return NoopSolver{}
	}
}

type Recovery struct {
	cfg    config.CaptchaConfig
	detect *Detector
	solver Solver
}

func NewRecovery(cfg config.CaptchaConfig, kameleoEndpoint string) *Recovery {
	return &Recovery{
		cfg:    cfg,
		detect: NewDetector(cfg),
		solver: NewSolver(cfg, kameleoEndpoint),
	}
}

func (r *Recovery) Handle(ctx context.Context, session *browser.Session) (Detection, error) {
	if !r.cfg.Enabled {
		return Detection{Detected: false, Type: ChallengeNone}, nil
	}
	d, err := r.detect.Detect(session)
	if err != nil || !d.Detected {
		return d, err
	}

	for attempt := 1; attempt <= r.cfg.MaxSolveAttempts; attempt++ {
		if err := r.solver.Solve(ctx, session, d); err != nil {
			if attempt == r.cfg.MaxSolveAttempts {
				return d, fmt.Errorf("captcha recovery failed (%s): %w", d.Type, err)
			}
			continue
		}
		time.Sleep(2 * time.Second)
		d2, err2 := r.detect.Detect(session)
		if err2 != nil {
			return d, err2
		}
		if !d2.Detected {
			return Detection{Detected: false, Type: ChallengeNone}, nil
		}
		d = d2
	}
	return d, fmt.Errorf("captcha still present after recovery (%s)", d.Type)
}
