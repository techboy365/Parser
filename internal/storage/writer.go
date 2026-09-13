package storage

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

type Writer struct {
	mu   sync.Mutex
	path string
}

func NewWriter(outputDir, fileName string) (*Writer, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("create output dir: %w", err)
	}
	path := filepath.Join(outputDir, fileName)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open results file: %w", err)
	}
	_ = f.Close()
	return &Writer{path: path}, nil
}

func (w *Writer) Append(url string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	f, err := os.OpenFile(w.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, url)
	return err
}

func (w *Writer) Path() string {
	return w.path
}

func LoadExisting(path string) (map[string]struct{}, error) {
	seen := make(map[string]struct{})
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return seen, nil
		}
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			seen[line] = struct{}{}
		}
	}
	return seen, scanner.Err()
}
