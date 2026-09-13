package search

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/techboy365/Parser/internal/captcha"
	"github.com/techboy365/Parser/internal/config"
	"github.com/techboy365/Parser/internal/dedupe"
	"github.com/techboy365/Parser/internal/extract"
	"github.com/techboy365/Parser/internal/filter"
	"github.com/techboy365/Parser/internal/job"
	"github.com/techboy365/Parser/internal/logging"
	"github.com/techboy365/Parser/internal/pagination"
	"github.com/techboy365/Parser/internal/retry"
	"github.com/techboy365/Parser/internal/session"
	"github.com/techboy365/Parser/internal/storage"
)

type DorkRequest struct {
	ID       string
	Query    string
	ResumeID string
}

type Result struct {
	JobID      string
	RequestID  string
	Query      string
	URLs       []string
	TotalFound int
	PagesScraped int
}

type Runner struct {
	cfg       *config.Config
	sessions  *session.Manager
	filter    *filter.Engine
	writer    *storage.Writer
	jobs      *job.Store
	captcha   *captcha.Recovery
	navigator *pagination.Navigator
	log       *slog.Logger
}

func NewRunner(
	cfg *config.Config,
	sessions *session.Manager,
	filterEngine *filter.Engine,
	writer *storage.Writer,
	jobs *job.Store,
	log *slog.Logger,
) *Runner {
	return &Runner{
		cfg:       cfg,
		sessions:  sessions,
		filter:    filterEngine,
		writer:    writer,
		jobs:      jobs,
		captcha:   captcha.NewRecovery(cfg.Captcha),
		navigator: pagination.NewNavigator(cfg.Browser.GoogleBaseURL, cfg.Browser.ResultsPerPage, cfg.Browser.MaxPages),
		log:       log,
	}
}

func (r *Runner) Run(ctx context.Context, req DorkRequest) (*Result, error) {
	if strings.TrimSpace(req.Query) == "" {
		return nil, fmt.Errorf("query is required")
	}
	if req.ID == "" {
		return nil, fmt.Errorf("request id is required")
	}

	dorkJob, err := job.ResumeOrCreate(r.jobs, req.Query, req.ResumeID)
	if err != nil {
		return nil, err
	}
	jlog := logging.NewJobLogger(r.log, dorkJob.ID, req.ID)

	dorkJob.Status = job.StatusRunning
	if err := r.jobs.Save(dorkJob); err != nil {
		return nil, err
	}

	deduper := dedupe.NewStore()
	collected := make([]string, 0)

	runErr := retry.Do(ctx, r.cfg.Retry, func(attempt int) error {
		jlog.Info("search attempt", "attempt", attempt, "page", dorkJob.CurrentPage)
		lease, err := r.sessions.Acquire(ctx, req.ID)
		if err != nil {
			return err
		}
		defer func() {
			if r.sessions.ShouldRotate(lease) {
				_ = r.sessions.Release(ctx, req.ID, true)
				return
			}
			_ = r.sessions.Release(ctx, req.ID, false)
		}()

		extractor := extract.NewHybridExtractor()
		if err := extractor.Install(lease.Session); err != nil {
			return err
		}

		for page := dorkJob.CurrentPage; page < r.navigator.MaxPages(); page++ {
			if err := ctx.Err(); err != nil {
				return err
			}

			searchURL := r.navigator.URL(req.Query, page)
			if err := lease.Session.Navigate(ctx, searchURL); err != nil {
				return fmt.Errorf("navigate page %d: %w", page, err)
			}
			lease.Session.WaitSettle(r.cfg.PageSettle())
			r.sessions.RecordSearch(lease)

			detection, err := r.captcha.Handle(ctx, lease.Session)
			if err != nil {
				r.sessions.RecordCaptcha(lease)
				dorkJob.LastError = err.Error()
				_ = r.jobs.Save(dorkJob)
				if rotateErr := r.rotateSession(ctx, req.ID); rotateErr != nil {
					return rotateErr
				}
				return err
			}
			if detection.Detected {
				r.sessions.RecordCaptcha(lease)
				return fmt.Errorf("captcha still present: %s", detection.Type)
			}

			rawURLs, err := extractor.Extract(lease.Session)
			if err != nil {
				return fmt.Errorf("extract page %d: %w", page, err)
			}

			pageCount := 0
			for _, u := range rawURLs {
				if !r.filter.Keep(u) {
					continue
				}
				normalized, added := deduper.Add(u)
				if !added {
					continue
				}
				if err := r.writer.Append(normalized); err != nil {
					return fmt.Errorf("write result: %w", err)
				}
				collected = append(collected, normalized)
				pageCount++
			}

			dorkJob.CurrentPage = page + 1
			dorkJob.TotalURLs = deduper.Count()
			dorkJob.LastError = ""
			if err := r.jobs.Save(dorkJob); err != nil {
				return err
			}

			jlog.Info("page scraped", "page", page, "page_urls", pageCount, "total_urls", dorkJob.TotalURLs)
			if !r.navigator.HasNext(lease.Session, page, len(rawURLs)) {
				break
			}
		}
		return nil
	})

	if runErr != nil {
		dorkJob.Status = job.StatusFailed
		dorkJob.LastError = runErr.Error()
		_ = r.jobs.Save(dorkJob)
		return nil, runErr
	}

	dorkJob.Status = job.StatusCompleted
	dorkJob.TotalURLs = deduper.Count()
	_ = r.jobs.Save(dorkJob)

	return &Result{
		JobID:        dorkJob.ID,
		RequestID:    req.ID,
		Query:        req.Query,
		URLs:         collected,
		TotalFound:   deduper.Count(),
		PagesScraped: dorkJob.CurrentPage,
	}, nil
}

func (r *Runner) rotateSession(ctx context.Context, requestID string) error {
	_, err := r.sessions.Rotate(ctx, requestID)
	return err
}

func LoadDorks(path string) ([]string, error) {
	data, err := readLines(path)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(data))
	for _, line := range data {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no dorks found in %s", path)
	}
	return out, nil
}

func readLines(path string) ([]string, error) {
	content, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return strings.Split(content, "\n"), nil
}

func readFile(path string) (string, error) {
	b, err := osReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
