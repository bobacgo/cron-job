package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	runlog "github.com/bobacgo/cron-job/internal/domain/log"
)

type FileRepository struct {
	mu      sync.Mutex
	baseDir string
}

func NewFileRepository(baseDir string) (*FileRepository, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, err
	}
	return &FileRepository{baseDir: baseDir}, nil
}

func (r *FileRepository) Append(_ context.Context, record runlog.LogRecord) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	path := filepath.Join(r.baseDir, record.RunID+".log")
	line := fmt.Sprintf("[%s] [%s] %s\n", record.OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00"), record.Stream, record.Content)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = file.WriteString(line)
	return err
}

func (r *FileRepository) Read(_ context.Context, runID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	path := filepath.Join(r.baseDir, runID+".log")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}
