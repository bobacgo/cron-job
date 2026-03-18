package shell

import (
	"bytes"
	"context"
	"time"

	"os/exec"

	jobrundomain "github.com/bobacgo/cron-job/internal/domain/jobrun"
	"github.com/bobacgo/cron-job/internal/executor"
)

type Executor struct{}

func NewExecutor() *Executor { return &Executor{} }

func (e *Executor) Execute(ctx context.Context, req executor.Request) (executor.Result, error) {
	target := req.Job.Executor.Shell
	if target == nil {
		return executor.Result{
			Status:     jobrundomain.StatusFailed,
			Message:    "shell target not configured",
			StartedAt:  time.Now().UTC(),
			FinishedAt: time.Now().UTC(),
		}, nil
	}

	shell := target.Shell
	if shell == "" {
		shell = "/bin/sh"
	}

	runCtx := ctx
	var cancel context.CancelFunc
	if target.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, target.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(runCtx, shell, "-c", target.Script)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	startedAt := time.Now().UTC()
	runErr := cmd.Run()
	finishedAt := time.Now().UTC()

	combined := stdout.String()
	if se := stderr.String(); se != "" {
		if combined != "" {
			combined += "\n"
		}
		combined += se
	}

	if runErr != nil {
		return executor.Result{
			Status:     jobrundomain.StatusFailed,
			Message:    runErr.Error(),
			Output:     combined,
			StartedAt:  startedAt,
			FinishedAt: finishedAt,
		}, nil
	}

	return executor.Result{
		Status:     jobrundomain.StatusSucceeded,
		Message:    "shell script completed",
		Output:     combined,
		StartedAt:  startedAt,
		FinishedAt: finishedAt,
	}, nil
}
