package normalize

import (
	"net/url"
	"sort"
	"strings"
)

var trackingParams = map[string]struct{}{
	"utm_source": {}, "utm_medium": {}, "utm_campaign": {}, "utm_term": {}, "utm_content": {},
	"gclid": {}, "fbclid": {}, "mc_eid": {}, "igshid": {},
}

func URL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", nil
	}
	raw = unwrapGoogleRedirect(raw)
	u, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""

	q := u.Query()
	for key := range q {
		if _, drop := trackingParams[strings.ToLower(key)]; drop {
			q.Del(key)
		}
	}
	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	clean := url.Values{}
	for _, k := range keys {
		vals := q[k]
		sort.Strings(vals)
		for _, v := range vals {
			clean.Add(k, v)
		}
	}
	u.RawQuery = clean.Encode()
	u.Path = strings.TrimRight(u.Path, "/")
	if u.Path == "" {
		u.Path = "/"
	}
	return u.String(), nil
}

func unwrapGoogleRedirect(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	if strings.Contains(u.Host, "google.") && u.Path == "/url" {
		if target := u.Query().Get("q"); target != "" {
			return target
		}
		if target := u.Query().Get("url"); target != "" {
			return target
		}
	}
	return raw
}
