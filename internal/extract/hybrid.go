package extract

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/playwright-community/playwright-go"
	"github.com/techboy365/Parser/internal/browser"
	"github.com/techboy365/Parser/internal/normalize"
)

const fetchHookScript = `
(() => {
  if (window.__parserHookInstalled) return;
  window.__parserHookInstalled = true;
  window.__parserCapturedURLs = window.__parserCapturedURLs || [];

  const pushURL = (value) => {
    if (typeof value !== 'string') return;
    const trimmed = value.trim();
    if (!trimmed) return;
    if (!/^https?:\/\//i.test(trimmed)) return;
    window.__parserCapturedURLs.push(trimmed);
  };

  const originalFetch = window.fetch.bind(window);
  window.fetch = async (...args) => {
    const response = await originalFetch(...args);
    try {
      const clone = response.clone();
      clone.text().then((body) => {
        const matches = body.match(/https?:\\\/\\\/[^"'\\s<>]+/g) || [];
        matches.forEach((m) => pushURL(m.replace(/\\\/\//g, '//')));
      }).catch(() => {});
    } catch (_) {}
    return response;
  };

  const originalOpen = XMLHttpRequest.prototype.open;
  XMLHttpRequest.prototype.open = function(method, url, ...rest) {
    pushURL(String(url));
    return originalOpen.call(this, method, url, ...rest);
  };
})();
`

type HybridExtractor struct {
	networkURLs []string
}

func NewHybridExtractor() *HybridExtractor {
	return &HybridExtractor{}
}

func (h *HybridExtractor) Install(session *browser.Session) error {
	if err := session.AddInitScript(fetchHookScript); err != nil {
		return fmt.Errorf("install fetch hook: %w", err)
	}
	return session.OnResponse(func(resp playwright.Response) {
		url := resp.URL()
		if strings.Contains(url, "google.") {
			return
		}
		h.networkURLs = append(h.networkURLs, url)
	})
}

func (h *HybridExtractor) Extract(session *browser.Session) ([]string, error) {
	h.networkURLs = nil

	domURLs, err := extractFromDOM(session)
	if err != nil {
		return nil, err
	}
	fetchURLs, err := extractFromFetchHook(session)
	if err != nil {
		return nil, err
	}
	cdpURLs := append([]string{}, h.networkURLs...)

	merged := mergeUnique(domURLs, fetchURLs, cdpURLs)
	out := make([]string, 0, len(merged))
	for _, raw := range merged {
		normalized, err := normalize.URL(raw)
		if err != nil || normalized == "" {
			continue
		}
		if isGoogleInternal(normalized) {
			continue
		}
		out = append(out, normalized)
	}
	return out, nil
}

func extractFromDOM(session *browser.Session) ([]string, error) {
	result, err := session.Evaluate(`() => {
    const urls = new Set();
    const push = (href) => {
      if (!href) return;
      try {
        const absolute = new URL(href, window.location.href).href;
        urls.add(absolute);
      } catch (_) {}
    };

    document.querySelectorAll('#search a[href], div.g a[href], a[data-ved][href]').forEach((a) => {
      push(a.getAttribute('href'));
    });

    document.querySelectorAll('cite, .VuuXrf, .tjvcx').forEach((el) => {
      const text = (el.textContent || '').trim();
      if (text.includes('.')) {
        push('https://' + text.replace(/^https?:\/\//, ''));
      }
    });

    return Array.from(urls);
  }`)
	if err != nil {
		return nil, err
	}
	return toStringSlice(result), nil
}

func extractFromFetchHook(session *browser.Session) ([]string, error) {
	result, err := session.Evaluate(`() => Array.from(new Set(window.__parserCapturedURLs || []))`)
	if err != nil {
		return nil, err
	}
	return toStringSlice(result), nil
}

func ExtractFromHTML(html string) []string {
	hrefRe := regexp.MustCompile(`href="([^"]+)"`)
	matches := hrefRe.FindAllStringSubmatch(html, -1)
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) > 1 {
			out = append(out, m[1])
		}
	}
	return out
}

func mergeUnique(groups ...[]string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, group := range groups {
		for _, u := range group {
			u = strings.TrimSpace(u)
			if u == "" {
				continue
			}
			if _, ok := seen[u]; ok {
				continue
			}
			seen[u] = struct{}{}
			out = append(out, u)
		}
	}
	return out
}

func toStringSlice(v any) []string {
	switch t := v.(type) {
	case []any:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	case []string:
		return t
	default:
		return nil
	}
}

func isGoogleInternal(u string) bool {
	lower := strings.ToLower(u)
	return strings.Contains(lower, "google.com") ||
		strings.Contains(lower, "googleusercontent.com") ||
		strings.Contains(lower, "gstatic.com") ||
		strings.Contains(lower, "accounts.google.com")
}

func ParseJSONURLs(body string) []string {
	var data any
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return nil
	}
	found := make(map[string]struct{})
	collectJSONURLs(data, found)
	out := make([]string, 0, len(found))
	for u := range found {
		out = append(out, u)
	}
	return out
}

func collectJSONURLs(v any, found map[string]struct{}) {
	switch t := v.(type) {
	case map[string]any:
		for _, val := range t {
			collectJSONURLs(val, found)
		}
	case []any:
		for _, val := range t {
			collectJSONURLs(val, found)
		}
	case string:
		if strings.HasPrefix(t, "http://") || strings.HasPrefix(t, "https://") {
			found[t] = struct{}{}
		}
	}
}
