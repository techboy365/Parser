package browser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/techboy365/Parser/internal/config"
	"github.com/techboy365/Parser/internal/kameleo"
)

type Session struct {
	ProfileID   string
	CDPEndpoint string
	Browser     playwright.Browser
	Context     playwright.BrowserContext
	Page        playwright.Page
	PW          *playwright.Playwright
	warmed      bool
}

type Manager struct {
	cfg    *config.Config
	client *kameleo.Client
	pw     *playwright.Playwright
}

func NewManager(cfg *config.Config, client *kameleo.Client) *Manager {
	return &Manager{cfg: cfg, client: client}
}

func (m *Manager) EnsurePlaywright() error {
	if m.pw != nil {
		return nil
	}
	pw, err := playwright.Run()
	if err != nil {
		return fmt.Errorf("start playwright: %w", err)
	}
	m.pw = pw
	return nil
}

func (m *Manager) Connect(ctx context.Context, profileID string) (*Session, error) {
	if err := m.EnsurePlaywright(); err != nil {
		return nil, err
	}

	endpoint := m.client.PlaywrightCDPEndpoint(profileID)
	timeout := float64(m.cfg.ConnectTimeout().Milliseconds())
	browser, err := m.pw.Chromium.ConnectOverCDP(endpoint, playwright.BrowserTypeConnectOverCDPOptions{
		Timeout: playwright.Float(timeout),
	})
	if err != nil {
		return nil, fmt.Errorf("connect over CDP (%s): %w", endpoint, err)
	}

	contexts := browser.Contexts()
	if len(contexts) == 0 {
		browser.Close()
		return nil, fmt.Errorf("no browser contexts available from kameleo profile")
	}
	bctx := contexts[0]
	pages := bctx.Pages()
	var page playwright.Page
	if len(pages) > 0 {
		page = pages[0]
	} else {
		page, err = bctx.NewPage()
		if err != nil {
			browser.Close()
			return nil, fmt.Errorf("new page: %w", err)
		}
	}

	timeoutMS := float64(m.cfg.NavigationTimeout().Milliseconds())
	page.SetDefaultNavigationTimeout(timeoutMS)
	page.SetDefaultTimeout(timeoutMS)

	return &Session{
		ProfileID:   profileID,
		CDPEndpoint: endpoint,
		Browser:     browser,
		Context:     bctx,
		Page:        page,
		PW:          m.pw,
	}, nil
}

func (s *Session) MarkWarmed() {
	s.warmed = true
}

func (s *Session) IsWarmed() bool {
	return s.warmed
}

func (s *Session) Close() error {
	if s.Page != nil {
		_ = s.Page.Close()
		s.Page = nil
	}
	if s.Browser != nil {
		err := s.Browser.Close()
		s.Browser = nil
		s.Context = nil
		return err
	}
	return nil
}

// Suspend closes the playwright connection without stopping the Kameleo profile,
// allowing the CAPTCHA sidecar to attach to the same profile CDP endpoint.
func (s *Session) Suspend() error {
	return s.Close()
}

func (s *Session) Navigate(ctx context.Context, url string) error {
	_, err := s.Page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *Session) WaitSettle(d time.Duration) {
	time.Sleep(d)
}

func (s *Session) CurrentURL() string {
	return s.Page.URL()
}

func (s *Session) Title() (string, error) {
	return s.Page.Title()
}

func (s *Session) Content() (string, error) {
	return s.Page.Content()
}

func (s *Session) Evaluate(js string) (any, error) {
	return s.Page.Evaluate(js)
}

func (s *Session) AddInitScript(script string) error {
	return s.Page.AddInitScript(playwright.Script{Content: playwright.String(script)})
}

func (s *Session) OnResponse(fn func(playwright.Response)) error {
	s.Page.OnResponse(func(resp playwright.Response) {
		fn(resp)
	})
	return nil
}

func BuildSearchURL(base, query string, start int) string {
	base = strings.TrimRight(base, "/")
	q := strings.TrimSpace(query)
	if start <= 0 {
		return fmt.Sprintf("%s/search?q=%s&num=10&hl=en", base, urlEncode(q))
	}
	return fmt.Sprintf("%s/search?q=%s&start=%d&num=10&hl=en", base, urlEncode(q), start)
}

func (m *Manager) Reconnect(ctx context.Context, profileID string) (*Session, error) {
	return m.Connect(ctx, profileID)
}

func urlEncode(s string) string {
	replacer := strings.NewReplacer(" ", "+", "\"", "%22")
	return replacer.Replace(s)
}
