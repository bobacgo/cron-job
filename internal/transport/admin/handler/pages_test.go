package handler

import (
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	jobdomain "github.com/bobacgo/cron-job/internal/domain/job"
)

func TestMustParseTemplates(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../../../../"))
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd() error = %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(wd); chdirErr != nil {
			t.Fatalf("restore working directory error = %v", chdirErr)
		}
	}()
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir(%q) error = %v", root, err)
	}

	if templates := mustParseTemplates(); templates == nil {
		t.Fatal("mustParseTemplates() returned nil")
	}
}

func TestBuildExecutorSpecShell(t *testing.T) {
	form := url.Values{}
	form.Set("executor_type", string(jobdomain.ExecutorKindShell))
	form.Set("shell_script", "echo hello")
	form.Set("shell_shell", "/bin/bash")
	form.Set("shell_timeout_seconds", "15")

	req := httptest.NewRequest("POST", "/jobs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatalf("ParseForm() error = %v", err)
	}

	spec := buildExecutorSpec(req)
	if spec.Kind != jobdomain.ExecutorKindShell {
		t.Fatalf("Kind = %q, want %q", spec.Kind, jobdomain.ExecutorKindShell)
	}
	if spec.Shell == nil {
		t.Fatal("Shell target is nil")
	}
	if spec.Shell.Script != "echo hello" {
		t.Fatalf("Script = %q, want %q", spec.Shell.Script, "echo hello")
	}
	if spec.Shell.Shell != "/bin/bash" {
		t.Fatalf("Shell = %q, want %q", spec.Shell.Shell, "/bin/bash")
	}
	if spec.Shell.Timeout != 15*time.Second {
		t.Fatalf("Timeout = %v, want %v", spec.Shell.Timeout, 15*time.Second)
	}
}

func TestBuildExecutorSpecShellDefaultsInterpreter(t *testing.T) {
	form := url.Values{}
	form.Set("executor_type", string(jobdomain.ExecutorKindShell))
	form.Set("shell_script", "echo hello")

	req := httptest.NewRequest("POST", "/jobs", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatalf("ParseForm() error = %v", err)
	}

	spec := buildExecutorSpec(req)
	if spec.Shell == nil {
		t.Fatal("Shell target is nil")
	}
	if spec.Shell.Shell != "/bin/sh" {
		t.Fatalf("Shell = %q, want %q", spec.Shell.Shell, "/bin/sh")
	}
}

func TestFilterJobsIncludesShell(t *testing.T) {
	jobs := []jobdomain.Job{
		{Name: "sdk", Executor: jobdomain.ExecutorSpec{Kind: jobdomain.ExecutorKindSDK}},
		{Name: "binary", Executor: jobdomain.ExecutorSpec{Kind: jobdomain.ExecutorKindBinary}},
		{Name: "shell", Executor: jobdomain.ExecutorSpec{Kind: jobdomain.ExecutorKindShell}},
	}

	filtered := filterJobs(jobs, "", "", "shell")
	if len(filtered) != 1 {
		t.Fatalf("len(filtered) = %d, want 1", len(filtered))
	}
	if filtered[0].Executor.Kind != jobdomain.ExecutorKindShell {
		t.Fatalf("filtered executor kind = %q, want %q", filtered[0].Executor.Kind, jobdomain.ExecutorKindShell)
	}
}

func TestSummarizeJobsIncludesShell(t *testing.T) {
	jobs := []jobdomain.Job{
		{Enabled: true, Executor: jobdomain.ExecutorSpec{Kind: jobdomain.ExecutorKindSDK}},
		{Enabled: false, Executor: jobdomain.ExecutorSpec{Kind: jobdomain.ExecutorKindBinary}},
		{Enabled: true, Executor: jobdomain.ExecutorSpec{Kind: jobdomain.ExecutorKindShell}},
	}

	enabled, disabled, sdk, binary, shell := summarizeJobs(jobs)
	if enabled != 2 || disabled != 1 || sdk != 1 || binary != 1 || shell != 1 {
		t.Fatalf("summarizeJobs() = (%d, %d, %d, %d, %d), want (2, 1, 1, 1, 1)", enabled, disabled, sdk, binary, shell)
	}
}

