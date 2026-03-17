package binary

import (
	"context"
	"os/exec"
	"time"

	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	"github.com/bobacgo/cron-job/internal/executor"
)

type Executor struct{}

func NewExecutor() *Executor { return &Executor{} }

func (e *Executor) Execute(ctx context.Context, req executor.Request) (executor.Result, error) {
	startedAt := time.Now().UTC()
	cmd := exec.CommandContext(ctx, req.Job.Executor.Binary.Command, req.Job.Executor.Binary.Args...)
	output, err := cmd.CombinedOutput()
	status := jobrundomain.StatusSucceeded
	message := "binary command completed"
	if err != nil {
		status = jobrundomain.StatusFailed
		message = err.Error()
	}

	return executor.Result{
		Status:     status,
		Message:    message,
		Output:     string(output),
		StartedAt:  startedAt,
		FinishedAt: time.Now().UTC(),
	}, nil
}
