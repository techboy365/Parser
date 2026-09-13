package pagination

import (
	"fmt"
	"strings"

	"github.com/techboy365/Parser/internal/browser"
)

type Navigator struct {
	baseURL        string
	resultsPerPage int
	maxPages       int
}

func NewNavigator(baseURL string, resultsPerPage, maxPages int) *Navigator {
	return &Navigator{
		baseURL:        baseURL,
		resultsPerPage: resultsPerPage,
		maxPages:       maxPages,
	}
}

func (n *Navigator) URL(query string, page int) string {
	start := page * n.resultsPerPage
	return browser.BuildSearchURL(n.baseURL, query, start)
}

func (n *Navigator) MaxPages() int {
	return n.maxPages
}

func (n *Navigator) HasNext(session *browser.Session, currentPage int, foundOnPage int) bool {
	if currentPage+1 >= n.maxPages {
		return false
	}
	if foundOnPage == 0 {
		return false
	}
	content, err := session.Content()
	if err != nil {
		return false
	}
	lower := strings.ToLower(content)
	if strings.Contains(lower, "next") && strings.Contains(lower, "pnnext") {
		return true
	}
	return foundOnPage >= n.resultsPerPage
}

func (n *Navigator) ClickNext(session *browser.Session) error {
	clicked, err := session.Evaluate(`() => {
    const next = document.querySelector('#pnnext, a[aria-label="Next page"], a[aria-label="Next"]');
    if (!next) return false;
    next.click();
    return true;
  }`)
	if err != nil {
		return err
	}
	if ok, _ := clicked.(bool); ok {
		return nil
	}
	return fmt.Errorf("next page control not found")
}
