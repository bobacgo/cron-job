export type ExecutorKind = 'sdk' | 'binary' | 'shell';

const API_PREFIX = '/api/v1';

export interface JobSchedule {
  Cron: string;
  Interval: number;
  TimeZone: string;
  StartingDeadlineSeconds?: number;
}

export interface RetryPolicy {
  MaxRetries: number;
  InitialBackoff: number;
  MaxBackoff: number;
  BackoffMultiple: number;
}

export interface SDKTarget {
  Protocol: string;
  URL: string;
  Method: string;
  Timeout: number;
}

export interface BinaryTarget {
  Command: string;
  Args: string[];
  Timeout: number;
}

export interface ShellTarget {
  Script: string;
  Shell: string;
  Timeout: number;
}

export interface ExecutorSpec {
  Kind: ExecutorKind;
  SDK?: SDKTarget;
  Binary?: BinaryTarget;
  Shell?: ShellTarget;
}

export interface Job {
  ID: string;
  Name: string;
  Description: string;
  Enabled: boolean;
  Schedule: JobSchedule;
  Executor: ExecutorSpec;
  RetryPolicy: RetryPolicy;
  ConcurrencyPolicy: string;
  NextRunAt: number;
  LastRunAt: number;
  LastSuccessAt: number;
  CreatedAt: number;
  UpdatedAt: number;
}

export interface DependencyEdge {
  JobID: string;
  DependsOnJobID: string;
}

export interface JobRun {
  ID: string;
  JobID: string;
  ScheduledAt: number;
  StartedAt: number;
  FinishedAt: number;
  Status: string;
  Attempt: number;
  TriggerType: string;
  Message: string;
  CreatedAt: number;
  UpdatedAt: number;
}

export interface JobDetail {
  Job: Job;
  Dependencies: DependencyEdge[];
  DependencyJobs: Job[];
  Runs: JobRun[];
}

export interface RunLogResponse {
  run_id: string;
  stream: string;
  content: string;
}

export interface CreateJobPayload {
  name: string;
  description: string;
  enabled: boolean;
  cron: string;
  time_zone: string;
  interval_seconds: number;
  executor_type: ExecutorKind;
  concurrency_policy: string;
  max_retries: number;
  initial_backoff_seconds: number;
  max_backoff_seconds: number;
  backoff_multiple: number;
  sdk_protocol: string;
  sdk_url: string;
  sdk_method: string;
  sdk_timeout_seconds: number;
  binary_command: string;
  binary_args: string[];
  binary_timeout_seconds: number;
  dependency_ids: string[];
  shell_script: string;
  shell_shell: string;
  shell_timeout_seconds: number;
}

async function apiRequest<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_PREFIX}${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(init?.headers || {}),
    },
    ...init,
  });

  const contentType = response.headers.get('content-type') || '';
  const isJSON = contentType.includes('application/json');
  const body = isJSON ? await response.json() : await response.text();

  if (!response.ok) {
    const message = typeof body === 'string' ? body : body?.message || JSON.stringify(body);
    throw new Error(message || `请求失败: ${response.status}`);
  }

  return body as T;
}

export function listJobs() {
  return apiRequest<Job[]>('/jobs');
}

export function getJobDetail(id: string) {
  return apiRequest<JobDetail>(`/jobs/${id}`);
}

export function createJob(payload: CreateJobPayload) {
  return apiRequest<Job>('/jobs', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export function triggerJob(id: string) {
  return apiRequest<JobRun>(`/jobs/${id}/trigger`, { method: 'POST' });
}

export function pauseJob(id: string) {
  return apiRequest<Job>(`/jobs/${id}/pause`, { method: 'POST' });
}

export function resumeJob(id: string) {
  return apiRequest<Job>(`/jobs/${id}/resume`, { method: 'POST' });
}

export function cancelRun(id: string) {
  return apiRequest<JobRun>(`/job-runs/${id}/cancel`, { method: 'POST' });
}

export function retryRun(id: string) {
  return apiRequest<JobRun>(`/job-runs/${id}/retry`, { method: 'POST' });
}

export function getRunLog(id: string, stream = '') {
  const query = stream ? `?stream=${encodeURIComponent(stream)}` : '';
  return apiRequest<RunLogResponse>(`/job-runs/${id}/logs${query}`);
}

export function getLatestRun(detail?: JobDetail) {
  return detail?.Runs?.[0];
}

export function formatUnixTime(value?: number) {
  if (!value) {
    return '-';
  }
  return new Date(value * 1000).toLocaleString('zh-CN', { hour12: false });
}

function normalizeDurationSeconds(raw?: number) {
  if (!raw) {
    return 0;
  }
  // Go 的 time.Duration JSON 默认是纳秒，这里统一换算成秒。
  if (raw > 1e6) {
    return Math.round(raw / 1e9);
  }
  return Math.round(raw);
}

export function formatDuration(raw?: number) {
  const seconds = normalizeDurationSeconds(raw);
  if (!seconds) {
    return '-';
  }
  if (seconds < 60) {
    return `${seconds}s`;
  }
  if (seconds < 3600) {
    return `${Math.round(seconds / 60)}m`;
  }
  if (seconds < 86400) {
    return `${Math.round(seconds / 3600)}h`;
  }
  return `${Math.round(seconds / 86400)}d`;
}

export function formatSchedule(job: Job) {
  if (job.Schedule?.Cron) {
    return `Cron · ${job.Schedule.Cron}`;
  }
  return `Interval · ${formatDuration(job.Schedule?.Interval)}`;
}

export function getStatusTheme(status?: string) {
  switch (status) {
    case 'Succeeded':
      return 'success';
    case 'Running':
      return 'primary';
    case 'Ready':
    case 'Pending':
    case 'Blocked':
      return 'warning';
    case 'Failed':
    case 'TimedOut':
    case 'Canceled':
      return 'danger';
    default:
      return 'default';
  }
}

export function canRetryRun(run: JobRun) {
  return ['Failed', 'TimedOut', 'Canceled'].includes(run.Status);
}

export function canCancelRun(run: JobRun) {
  return ['Pending', 'Blocked', 'Ready', 'Running'].includes(run.Status);
}
