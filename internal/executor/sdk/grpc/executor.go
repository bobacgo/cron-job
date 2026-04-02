package grpc

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	"github.com/bobacgo/cron-job/internal/executor"
	sdkprotocol "github.com/bobacgo/cron-job/internal/executor/sdk/protocol"
	grpcpkg "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Executor struct{}

func NewExecutor() *Executor { return &Executor{} }

func (e *Executor) Execute(ctx context.Context, req executor.Request) (executor.Result, error) {
	startedAt := time.Now().UTC()
	conn, err := grpcpkg.DialContext(
		ctx,
		req.Job.Executor.SDK.URL,
		grpcpkg.WithTransportCredentials(insecure.NewCredentials()),
		grpcpkg.WithDefaultCallOptions(grpcpkg.ForceCodec(sdkprotocol.JSONCodec{})),
	)
	if err != nil {
		return executor.Result{Status: jobrundomain.StatusFailed, StartedAt: startedAt, FinishedAt: time.Now().UTC(), Message: err.Error()}, nil
	}
	defer conn.Close()

	request := sdkprotocol.RunRequest{
		ProtocolVersion:   sdkprotocol.CurrentVersion,
		SupportedVersions: append([]string(nil), sdkprotocol.SupportedVersions...),
		JobID:             req.Job.ID,
		JobName:           req.Job.Name,
		RunID:             req.Run.ID,
		ScheduledAt:       time.Unix(req.Run.ScheduledAt, 0).UTC(),
		Attempt:           req.Run.Attempt,
		TriggerType:       req.Run.TriggerType,
	}
	response := sdkprotocol.RunResponse{}
	method := req.Job.Executor.SDK.Method
	if method == "" {
		method = sdkprotocol.MethodExecutorRun
	}
	if err := conn.Invoke(ctx, method, &request, &response); err != nil {
		return executor.Result{Status: jobrundomain.StatusFailed, StartedAt: startedAt, FinishedAt: time.Now().UTC(), Message: err.Error()}, nil
	}

	output := response.Output
	if output == "" {
		outputBytes, err := json.Marshal(response)
		if err != nil {
			return executor.Result{}, err
		}
		output = string(outputBytes)
	}
	if response.ProtocolVersion != "" && !sdkprotocol.IsSupportedVersion(response.ProtocolVersion) {
		return executor.Result{
			Status:     jobrundomain.StatusFailed,
			Message:    "unsupported worker protocol version: " + response.ProtocolVersion,
			Output:     output,
			StartedAt:  startedAt,
			FinishedAt: time.Now().UTC(),
		}, nil
	}
	status := sdkprotocol.StatusFromResponse(response)
	message := strings.TrimSpace(response.Message)
	if message == "" {
		message = "sdk grpc request completed"
	}
	if response.ErrorCode != "" && response.ErrorCode != sdkprotocol.ErrorCodeNone {
		message = string(response.ErrorCode) + ": " + message
	}
	return executor.Result{
		Status:     status,
		Message:    message,
		Output:     output,
		StartedAt:  startedAt,
		FinishedAt: time.Now().UTC(),
	}, nil
}

func init() {
	sdkprotocol.RegisterJSONCodec()
}
