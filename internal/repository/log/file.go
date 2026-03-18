package log

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

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
	streamPath := filepath.Join(r.baseDir, fmt.Sprintf("%s.%s.log", record.RunID, sanitizeStream(record.Stream)))
	line := fmt.Sprintf("[%s] [%s] %s\n", record.OccurredAt.UTC().Format("2006-01-02T15:04:05Z07:00"), record.Stream, record.Content)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(line); err != nil {
		return err
	}
	streamFile, err := os.OpenFile(streamPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer streamFile.Close()
	_, err = streamFile.WriteString(line)
	return err
}

func (r *FileRepository) Read(_ context.Context, runID string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	path := filepath.Join(r.baseDir, runID+".log")
	data, err := readPath(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *FileRepository) ReadStream(_ context.Context, runID, stream string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	path := filepath.Join(r.baseDir, fmt.Sprintf("%s.%s.log", runID, sanitizeStream(stream)))
	data, err := readPath(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *FileRepository) Search(_ context.Context, query Query) ([]SearchItem, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	limit := query.Limit
	if limit <= 0 {
		limit = 100
	}
	paths, err := r.searchPaths(query)
	if err != nil {
		return nil, err
	}
	items := make([]SearchItem, 0)
	contains := strings.ToLower(strings.TrimSpace(query.Contains))
	streamFilter := strings.ToLower(strings.TrimSpace(query.Stream))
	for _, path := range paths {
		data, err := readPath(path)
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) == "" {
				continue
			}
			entry, ok := parseLine(line)
			if !ok {
				continue
			}
			if query.RunID != "" {
				entry.RunID = query.RunID
			} else {
				entry.RunID = runIDFromPath(path)
			}
			if streamFilter != "" && strings.ToLower(entry.Stream) != streamFilter {
				continue
			}
			if contains != "" && !strings.Contains(strings.ToLower(entry.Content), contains) {
				continue
			}
			items = append(items, entry)
			if len(items) >= limit {
				return items, nil
			}
		}
	}
	return items, nil
}

func (r *FileRepository) searchPaths(query Query) ([]string, error) {
	if query.RunID != "" {
		if strings.TrimSpace(query.Stream) != "" {
			return []string{filepath.Join(r.baseDir, fmt.Sprintf("%s.%s.log", query.RunID, sanitizeStream(query.Stream)))}, nil
		}
		return []string{filepath.Join(r.baseDir, query.RunID+".log")}, nil
	}
	entries, err := os.ReadDir(r.baseDir)
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".log") {
			continue
		}
		if strings.Contains(name, ".stdout.log") || strings.Contains(name, ".stderr.log") {
			continue
		}
		paths = append(paths, filepath.Join(r.baseDir, name))
	}
	sort.Strings(paths)
	return paths, nil
}

func readPath(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

func sanitizeStream(stream string) string {
	trimmed := strings.ToLower(strings.TrimSpace(stream))
	if trimmed == "" {
		return "stdout"
	}
	if trimmed == "stdout" || trimmed == "stderr" {
		return trimmed
	}
	return "stdout"
}

func parseLine(line string) (SearchItem, bool) {
	if !strings.HasPrefix(line, "[") {
		return SearchItem{}, false
	}
	first := strings.Index(line, "]")
	if first <= 1 {
		return SearchItem{}, false
	}
	remaining := strings.TrimSpace(line[first+1:])
	if !strings.HasPrefix(remaining, "[") {
		return SearchItem{}, false
	}
	second := strings.Index(remaining, "]")
	if second <= 1 {
		return SearchItem{}, false
	}
	rawTime := line[1:first]
	rawStream := remaining[1:second]
	rawContent := strings.TrimSpace(remaining[second+1:])
	occurredAt, err := time.Parse("2006-01-02T15:04:05Z07:00", rawTime)
	if err != nil {
		occurredAt = time.Time{}
	}
	return SearchItem{Stream: rawStream, Content: rawContent, OccurredAt: occurredAt}, true
}

func runIDFromPath(path string) string {
	base := filepath.Base(path)
	return strings.TrimSuffix(base, ".log")
}
