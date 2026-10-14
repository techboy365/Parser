package browser

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/techboy365/Parser/internal/config"
	"github.com/techboy365/Parser/internal/pacing"
)

type SearchNavigator struct {
	cfg    config.BrowserConfig
	pacing *pacing.Limiter
}

func NewSearchNavigator(cfg config.BrowserConfig, limiter *pacing.Limiter) *SearchNavigator {
	return &SearchNavigator{cfg: cfg, pacing: limiter}
}

func (n *SearchNavigator) WarmSession(ctx context.Context, s *Session) error {
	if !n.cfg.WarmSession {
		return nil
	}
	base := strings.TrimRight(n.cfg.GoogleBaseURL, "/")
	if err := s.Navigate(ctx, base+"/"); err != nil {
		return fmt.Errorf("warm session: %w", err)
	}
	n.pacing.BeforeAction()
	s.WaitSettle(n.cfgSettle())
	_ = n.acceptConsent(s)
	n.pacing.BeforeAction()
	return nil
}

func (n *SearchNavigator) GoToResults(ctx context.Context, s *Session, query string, page int) error {
	if page == 0 && n.cfg.HumanSearch {
		return n.humanSearch(ctx, s, query)
	}
	url := BuildSearchURL(n.cfg.GoogleBaseURL, query, page)
	n.pacing.BeforeAction()
	if err := s.Navigate(ctx, url); err != nil {
		return err
	}
	s.WaitSettle(n.cfgSettle())
	return nil
}

func (n *SearchNavigator) humanSearch(ctx context.Context, s *Session, query string) error {
	base := strings.TrimRight(n.cfg.GoogleBaseURL, "/")
	if err := s.Navigate(ctx, base+"/"); err != nil {
		return fmt.Errorf("open google homepage: %w", err)
	}
	n.pacing.BeforeAction()
	s.WaitSettle(n.cfgSettle())
	_ = n.acceptConsent(s)

	selectors := []string{
		"textarea[name='q']",
		"input[name='q']",
		"[role='combobox']",
	}
	var box playwright.Locator
	for _, sel := range selectors {
		loc := s.Page.Locator(sel).First()
		count, err := loc.Count()
		if err == nil && count > 0 {
			box = loc
			break
		}
	}
	if box == nil {
		return fmt.Errorf("search box not found on google homepage")
	}

	n.pacing.BeforeAction()
	if err := box.Click(); err != nil {
		return fmt.Errorf("focus search box: %w", err)
	}
	delay := float64(n.cfg.TypingDelayMS)
	if delay <= 0 {
		delay = 45
	}
	if err := box.Fill(""); err != nil {
		return fmt.Errorf("clear search box: %w", err)
	}
	if err := box.Type(query, playwright.LocatorTypeOptions{Delay: playwright.Float(delay)}); err != nil {
		return fmt.Errorf("type query: %w", err)
	}
	n.pacing.BeforeAction()
	if err := s.Page.Keyboard().Press("Enter"); err != nil {
		return fmt.Errorf("submit search: %w", err)
	}
	s.WaitSettle(n.cfgSettle())
	return nil
}

func (n *SearchNavigator) acceptConsent(s *Session) error {
	buttons := []string{
		"button:has-text('Accept all')",
		"button:has-text('I agree')",
		"button:has-text('Accept')",
		"#L2AGLb",
	}
	for _, sel := range buttons {
		loc := s.Page.Locator(sel).First()
		count, err := loc.Count()
		if err != nil || count == 0 {
			continue
		}
		_ = loc.Click(playwright.LocatorClickOptions{Timeout: playwright.Float(3000)})
		s.WaitSettle(800 * time.Millisecond)
		return nil
	}
	return nil
}

func (n *SearchNavigator) cfgSettle() time.Duration {
	if n.cfg.PageSettleMS <= 0 {
		return 1500 * time.Millisecond
	}
	return time.Duration(n.cfg.PageSettleMS) * time.Millisecond
}

func (n *SearchNavigator) ScrollResults(s *Session) {
	_, _ = s.Evaluate(`() => window.scrollBy(0, Math.floor(window.innerHeight * 0.6))`)
	n.pacing.BeforeAction()
}
