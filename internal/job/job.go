package job

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending   Status = "pending"
	StatusRunning   Status = "running"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusPaused    Status = "paused"
)

type DorkJob struct {
	ID          string    `json:"id"`
	Query       string    `json:"query"`
	Status      Status    `json:"status"`
	CurrentPage int       `json:"current_page"`
	TotalURLs   int       `json:"total_urls"`
	LastError   string    `json:"last_error,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Store struct {
	dir string
}

func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{dir: dir}, nil
}

func (s *Store) path(id string) string {
	return filepath.Join(s.dir, id+".json")
}

func (s *Store) Save(job *DorkJob) error {
	job.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(job, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(job.ID), data, 0o644)
}

func (s *Store) Load(id string) (*DorkJob, error) {
	data, err := os.ReadFile(s.path(id))
	if err != nil {
		return nil, err
	}
	var job DorkJob
	if err := json.Unmarshal(data, &job); err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *Store) List() ([]*DorkJob, error) {
	entries, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}
	out := make([]*DorkJob, 0)
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		id := e.Name()[:len(e.Name())-5]
		job, err := s.Load(id)
		if err != nil {
			continue
		}
		out = append(out, job)
	}
	return out, nil
}

func NewDorkJob(query string) *DorkJob {
	now := time.Now().UTC()
	return &DorkJob{
		ID:        uuid.NewString(),
		Query:     query,
		Status:    StatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func ResumeOrCreate(store *Store, query, resumeID string) (*DorkJob, error) {
	if resumeID != "" {
		job, err := store.Load(resumeID)
		if err != nil {
			return nil, fmt.Errorf("load job %s: %w", resumeID, err)
		}
		return job, nil
	}
	job := NewDorkJob(query)
	if err := store.Save(job); err != nil {
		return nil, err
	}
	return job, nil
}
