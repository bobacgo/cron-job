package grpc

import (
	"context"
	"encoding/json"
	"time"

	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	"github.com/bobacgo/cron-job/internal/executor"
	grpcpkg "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
)

type Executor struct{}

func NewExecutor() *Executor { return &Executor{} }

func (e *Executor) Execute(ctx context.Context, req executor.Request) (executor.Result, error) {
	startedAt := time.Now().UTC()
	conn, err := grpcpkg.DialContext(
		ctx,
		req.Job.Executor.SDK.URL,
		grpcpkg.WithTransportCredentials(insecure.NewCredentials()),
		grpcpkg.WithDefaultCallOptions(grpcpkg.ForceCodec(jsonCodec{})),
	)
	if err != nil {
		return executor.Result{Status: jobrundomain.StatusFailed, StartedAt: startedAt, FinishedAt: time.Now().UTC(), Message: err.Error()}, nil
	}
	defer conn.Close()

	payload := map[string]any{
		"job_id":       req.Job.ID,
		"job_name":     req.Job.Name,
		"run_id":       req.Run.ID,
		"scheduled_at": req.Run.ScheduledAt,
	}
	response := map[string]any{}
	method := req.Job.Executor.SDK.Method
	if method == "" {
		method = "/cronjob.v1.Executor/Run"
	}
	if err := conn.Invoke(ctx, method, payload, &response); err != nil {
		return executor.Result{Status: jobrundomain.StatusFailed, StartedAt: startedAt, FinishedAt: time.Now().UTC(), Message: err.Error()}, nil
	}

	outputBytes, err := json.Marshal(response)
	if err != nil {
		return executor.Result{}, err
	}
	status := jobrundomain.StatusSucceeded
	message := "sdk grpc request completed"
	if raw, ok := response["status"].(string); ok && raw == "failed" {
		status = jobrundomain.StatusFailed
		message = "sdk grpc returned failed status"
	}
	return executor.Result{
		Status:     status,
		Message:    message,
		Output:     string(outputBytes),
		StartedAt:  startedAt,
		FinishedAt: time.Now().UTC(),
	}, nil
}

type jsonCodec struct{}

func (jsonCodec) Marshal(v any) ([]byte, error) {
	return json.Marshal(v)
}

func (jsonCodec) Unmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

func (jsonCodec) Name() string {
	return "json"
}

func init() {
	encoding.RegisterCodec(jsonCodec{})
}
