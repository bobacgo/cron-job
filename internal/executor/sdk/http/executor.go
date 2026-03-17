package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	stdhttp "net/http"
	"time"

	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	"github.com/bobacgo/cron-job/internal/executor"
)

type Executor struct {
	client *stdhttp.Client
}

func NewExecutor(client *stdhttp.Client) *Executor {
	return &Executor{client: client}
}

func (e *Executor) Execute(ctx context.Context, req executor.Request) (executor.Result, error) {
	startedAt := time.Now().UTC()
	payload := map[string]any{
		"job_id":       req.Job.ID,
		"job_name":     req.Job.Name,
		"run_id":       req.Run.ID,
		"scheduled_at": req.Run.ScheduledAt,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return executor.Result{}, err
	}

	httpReq, err := stdhttp.NewRequestWithContext(ctx, req.Job.Executor.SDK.Method, req.Job.Executor.SDK.URL, bytes.NewReader(body))
	if err != nil {
		return executor.Result{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := e.client.Do(httpReq)
	if err != nil {
		return executor.Result{Status: jobrundomain.StatusFailed, StartedAt: startedAt, FinishedAt: time.Now().UTC(), Message: err.Error()}, nil
	}
	defer resp.Body.Close()
	bodyBytes, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return executor.Result{}, readErr
	}

	status := jobrundomain.StatusSucceeded
	message := "sdk http request completed"
	if resp.StatusCode >= 400 {
		status = jobrundomain.StatusFailed
		message = fmt.Sprintf("sdk http returned status %d", resp.StatusCode)
	}

	return executor.Result{
		Status:     status,
		Message:    message,
		Output:     string(bodyBytes),
		StartedAt:  startedAt,
		FinishedAt: time.Now().UTC(),
	}, nil
}
