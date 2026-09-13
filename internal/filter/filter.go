package filter

import (
	"net/url"
	"regexp"
	"strings"

	"github.com/techboy365/Parser/internal/config"
)

type Engine struct {
	includeKeywords     []string
	excludeKeywords     []string
	includeDomains      []string
	excludeDomains      []string
	includeURLPatterns  []*regexp.Regexp
	excludeURLPatterns  []*regexp.Regexp
	includePathPatterns []*regexp.Regexp
	excludePathPatterns []*regexp.Regexp
	includeQueryParams  map[string]struct{}
	excludeQueryParams  map[string]struct{}
	customRules         []*regexp.Regexp
}

func NewEngine(cfg config.FilterConfig) (*Engine, error) {
	e := &Engine{
		includeKeywords:    toLowerSlice(cfg.IncludeKeywords),
		excludeKeywords:    toLowerSlice(cfg.ExcludeKeywords),
		includeDomains:     toLowerSlice(cfg.IncludeDomains),
		excludeDomains:     toLowerSlice(cfg.ExcludeDomains),
		includeQueryParams: toSet(toLowerSlice(cfg.IncludeQueryParams)),
		excludeQueryParams: toSet(toLowerSlice(cfg.ExcludeQueryParams)),
	}
	var err error
	if e.includeURLPatterns, err = compilePatterns(cfg.IncludeURLPatterns); err != nil {
		return nil, err
	}
	if e.excludeURLPatterns, err = compilePatterns(cfg.ExcludeURLPatterns); err != nil {
		return nil, err
	}
	if e.includePathPatterns, err = compilePatterns(cfg.IncludePathPatterns); err != nil {
		return nil, err
	}
	if e.excludePathPatterns, err = compilePatterns(cfg.ExcludePathPatterns); err != nil {
		return nil, err
	}
	if e.customRules, err = compilePatterns(cfg.CustomRules); err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) Keep(raw string) bool {
	rawLower := strings.ToLower(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}

	if len(e.excludeKeywords) > 0 && containsAny(rawLower, e.excludeKeywords) {
		return false
	}
	if len(e.includeKeywords) > 0 && !containsAny(rawLower, e.includeKeywords) {
		return false
	}

	host := strings.ToLower(u.Host)
	if len(e.excludeDomains) > 0 && domainMatches(host, e.excludeDomains) {
		return false
	}
	if len(e.includeDomains) > 0 && !domainMatches(host, e.includeDomains) {
		return false
	}

	if matchesAny(raw, e.excludeURLPatterns) {
		return false
	}
	if len(e.includeURLPatterns) > 0 && !matchesAny(raw, e.includeURLPatterns) {
		return false
	}

	path := strings.ToLower(u.Path)
	if matchesAny(path, e.excludePathPatterns) {
		return false
	}
	if len(e.includePathPatterns) > 0 && !matchesAny(path, e.includePathPatterns) {
		return false
	}

	for key := range u.Query() {
		keyLower := strings.ToLower(key)
		if _, ok := e.excludeQueryParams[keyLower]; ok {
			return false
		}
	}
	if len(e.includeQueryParams) > 0 {
		found := false
		for key := range u.Query() {
			if _, ok := e.includeQueryParams[strings.ToLower(key)]; ok {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(e.customRules) > 0 && !matchesAny(raw, e.customRules) {
		return false
	}
	return true
}

func compilePatterns(patterns []string) ([]*regexp.Regexp, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	out := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, err
		}
		out = append(out, re)
	}
	return out, nil
}

func toLowerSlice(in []string) []string {
	out := make([]string, len(in))
	for i, v := range in {
		out[i] = strings.ToLower(strings.TrimSpace(v))
	}
	return out
}

func toSet(in []string) map[string]struct{} {
	out := make(map[string]struct{}, len(in))
	for _, v := range in {
		if v != "" {
			out[v] = struct{}{}
		}
	}
	return out
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if n != "" && strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func domainMatches(host string, domains []string) bool {
	for _, d := range domains {
		if d == "" {
			continue
		}
		if host == d || strings.HasSuffix(host, "."+d) {
			return true
		}
	}
	return false
}

func matchesAny(s string, patterns []*regexp.Regexp) bool {
	for _, re := range patterns {
		if re.MatchString(s) {
			return true
		}
	}
	return false
}
