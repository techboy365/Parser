package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/google/uuid"
	"github.com/techboy365/Parser/internal/config"
	"github.com/techboy365/Parser/internal/filter"
	"github.com/techboy365/Parser/internal/job"
	"github.com/techboy365/Parser/internal/kameleo"
	"github.com/techboy365/Parser/internal/logging"
	"github.com/techboy365/Parser/internal/search"
	"github.com/techboy365/Parser/internal/session"
	"github.com/techboy365/Parser/internal/storage"
	browserpkg "github.com/techboy365/Parser/internal/browser"
)

func main() {
	cfgPath := flag.String("config", "configs/default.yaml", "path to config file")
	dork := flag.String("dork", "", "single google dork query")
	dorksFile := flag.String("dorks", "", "file containing one dork per line")
	resumeJob := flag.String("resume", "", "resume an existing job id")
	listJobs := flag.Bool("list-jobs", false, "list saved jobs")
	flag.Parse()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	log := logging.New(cfg.Logging.Level, cfg.Logging.Format)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	jobStore, err := job.NewStore(cfg.Storage.JobStateDir)
	if err != nil {
		log.Error("create job store", "error", err)
		os.Exit(1)
	}

	if *listJobs {
		jobs, err := jobStore.List()
		if err != nil {
			log.Error("list jobs", "error", err)
			os.Exit(1)
		}
		for _, j := range jobs {
			fmt.Printf("%s\t%s\t%s\tpages=%d\turls=%d\t%s\n", j.ID, j.Status, j.Query, j.CurrentPage, j.TotalURLs, j.UpdatedAt.Format("2006-01-02T15:04:05Z"))
		}
		return
	}

	queries, err := resolveQueries(*dork, *dorksFile)
	if err != nil {
		log.Error("resolve dorks", "error", err)
		os.Exit(1)
	}

	filterEngine, err := filter.NewEngine(cfg.Filter)
	if err != nil {
		log.Error("create filter engine", "error", err)
		os.Exit(1)
	}

	writer, err := storage.NewWriter(cfg.Storage.OutputDir, cfg.Storage.ResultsFile)
	if err != nil {
		log.Error("create writer", "error", err)
		os.Exit(1)
	}

	kClient := kameleo.NewClient(cfg.Kameleo.Endpoint)
	if err := kClient.VerifyEngineReady(ctx); err != nil {
		log.Error("kameleo not ready", "error", err)
		os.Exit(1)
	}

	browserMgr := browserpkg.NewManager(cfg, kClient)
	sessionMgr := session.NewManager(cfg, kClient, browserMgr, log)
	defer sessionMgr.Shutdown(ctx)

	runner := search.NewRunner(cfg, sessionMgr, filterEngine, writer, jobStore, log)

	for _, query := range queries {
		req := search.DorkRequest{
			ID:       uuid.NewString(),
			Query:    query,
			ResumeID: *resumeJob,
		}
		log.Info("starting dork", "query", query, "request_id", req.ID)
		result, err := runner.Run(ctx, req)
		if err != nil {
			log.Error("dork failed", "query", query, "error", err)
			continue
		}
		log.Info("dork completed",
			"job_id", result.JobID,
			"query", result.Query,
			"total_urls", result.TotalFound,
			"pages", result.PagesScraped,
			"output", writer.Path(),
		)
	}
}

func resolveQueries(single, file string) ([]string, error) {
	if file != "" {
		return search.LoadDorks(file)
	}
	if strings.TrimSpace(single) != "" {
		return []string{single}, nil
	}
	return nil, fmt.Errorf("provide --dork or --dorks")
}
