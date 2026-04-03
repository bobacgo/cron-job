<template>
  <div class="job-center">
    <t-card title="任务中心" :bordered="false" class="panel-card">
      <template #actions>
        <t-space class="toolbar" size="12">
          <t-input v-model="filters.keyword" class="toolbar-search" clearable placeholder="搜索任务名 / 描述">
            <template #suffix-icon>
              <search-icon size="16px" />
            </template>
          </t-input>

          <t-select v-model="filters.executor" clearable style="width: 140px" placeholder="执行器">
            <t-option value="sdk" label="SDK" />
            <t-option value="binary" label="Binary" />
            <t-option value="shell" label="Shell" />
          </t-select>

          <t-select v-model="filters.status" clearable style="width: 140px" placeholder="运行状态">
            <t-option value="Running" label="Running" />
            <t-option value="Succeeded" label="Succeeded" />
            <t-option value="Failed" label="Failed" />
            <t-option value="Ready" label="Ready" />
            <t-option value="Blocked" label="Blocked" />
          </t-select>

          <t-space size="8">
            <span class="switch-text">自动刷新</span>
            <t-switch v-model="autoRefresh" />
          </t-space>

          <t-button variant="outline" :loading="loading" @click="fetchAllData(true)">刷新</t-button>
          <t-button theme="primary" @click="openCreateDialog">新建任务</t-button>
        </t-space>
      </template>

      <t-row :gutter="[16, 16]" class="summary-row">
        <t-col v-for="card in summaryCards" :key="card.label" :xs="6" :xl="3">
          <div class="summary-item">
            <div class="summary-label">{{ card.label }}</div>
            <div class="summary-value">{{ card.value }}</div>
            <div class="summary-hint">{{ card.hint }}</div>
          </div>
        </t-col>
      </t-row>

      <t-table
        :data="filteredRows"
        :columns="jobColumns"
        row-key="ID"
        size="small"
        :loading="loading"
        :pagination="{ defaultPageSize: 8, total: filteredRows.length }"
      >
        <template #enabled="{ row }">
          <t-tag :theme="row.Enabled ? 'success' : 'warning'" variant="light">
            {{ row.Enabled ? '启用' : '暂停' }}
          </t-tag>
        </template>

        <template #executor="{ row }">
          <t-tag theme="primary" variant="light">{{ row.Executor.Kind }}</t-tag>
        </template>

        <template #latestStatus="{ row }">
          <t-tag :theme="getStatusTheme(row.latestStatus)" variant="light">
            {{ row.latestStatus }}
          </t-tag>
        </template>

        <template #nextRunAt="{ row }">
          {{ formatUnixTime(row.NextRunAt) }}
        </template>

        <template #op="{ row }">
          <t-space size="8">
            <t-link theme="primary" @click="openDetail(row)">详情</t-link>
            <t-link theme="primary" :disabled="!row.Enabled" @click="handleTrigger(row)">立即执行</t-link>
            <t-link theme="warning" @click="handleToggle(row)">{{ row.Enabled ? '暂停' : '恢复' }}</t-link>
          </t-space>
        </template>
      </t-table>
    </t-card>

    <t-dialog v-model:visible="createVisible" header="新建任务" width="760px" :footer="false" destroy-on-close>
      <div class="create-form">
        <t-row :gutter="[16, 16]">
          <t-col :xs="12" :md="6">
            <div class="field-label">任务名</div>
            <t-input v-model="createForm.name" placeholder="例如：cleanup-tmp" />
          </t-col>
          <t-col :xs="12" :md="6">
            <div class="field-label">时区</div>
            <t-input v-model="createForm.time_zone" placeholder="默认 Asia/Shanghai" />
          </t-col>
          <t-col :xs="12">
            <div class="field-label">描述</div>
            <t-textarea v-model="createForm.description" :autosize="{ minRows: 2, maxRows: 4 }" />
          </t-col>
          <t-col :xs="12" :md="6">
            <div class="field-label">Cron 表达式</div>
            <t-input v-model="createForm.cron" placeholder="留空则走 interval" />
          </t-col>
          <t-col :xs="12" :md="6">
            <div class="field-label">间隔秒数</div>
            <t-input-number v-model="createForm.interval_seconds" theme="column" :min="0" />
          </t-col>
          <t-col :xs="12" :md="6">
            <div class="field-label">执行器类型</div>
            <t-select v-model="createForm.executor_type">
              <t-option value="shell" label="Shell" />
              <t-option value="binary" label="Binary" />
              <t-option value="sdk" label="SDK" />
            </t-select>
          </t-col>
          <t-col :xs="12" :md="6">
            <div class="field-label">并发策略</div>
            <t-select v-model="createForm.concurrency_policy">
              <t-option value="Forbid" label="Forbid" />
              <t-option value="Allow" label="Allow" />
              <t-option value="Replace" label="Replace" />
            </t-select>
          </t-col>
          <t-col :xs="12" :md="6">
            <div class="field-label">最大重试次数</div>
            <t-input-number v-model="createForm.max_retries" theme="column" :min="0" />
          </t-col>
          <t-col :xs="12" :md="6">
            <div class="field-label">依赖任务</div>
            <t-select v-model="createForm.dependency_ids" multiple clearable placeholder="可选，支持多选">
              <t-option v-for="job in jobs" :key="job.ID" :value="job.ID" :label="job.Name" />
            </t-select>
          </t-col>

          <template v-if="createForm.executor_type === 'sdk'">
            <t-col :xs="12" :md="4">
              <div class="field-label">协议</div>
              <t-select v-model="createForm.sdk_protocol">
                <t-option value="http" label="http" />
                <t-option value="grpc" label="grpc" />
              </t-select>
            </t-col>
            <t-col :xs="12" :md="4">
              <div class="field-label">HTTP Method</div>
              <t-select v-model="createForm.sdk_method">
                <t-option value="POST" label="POST" />
                <t-option value="GET" label="GET" />
              </t-select>
            </t-col>
            <t-col :xs="12" :md="4">
              <div class="field-label">超时（秒）</div>
              <t-input-number v-model="createForm.sdk_timeout_seconds" theme="column" :min="1" />
            </t-col>
            <t-col :xs="12">
              <div class="field-label">SDK 地址</div>
              <t-input v-model="createForm.sdk_url" placeholder="例如：http://127.0.0.1:8081/execute" />
            </t-col>
          </template>

          <template v-if="createForm.executor_type === 'binary'">
            <t-col :xs="12" :md="8">
              <div class="field-label">命令</div>
              <t-input v-model="createForm.binary_command" placeholder="例如：/usr/bin/python3" />
            </t-col>
            <t-col :xs="12" :md="4">
              <div class="field-label">超时（秒）</div>
              <t-input-number v-model="createForm.binary_timeout_seconds" theme="column" :min="1" />
            </t-col>
            <t-col :xs="12">
              <div class="field-label">参数列表</div>
              <t-input v-model="binaryArgsText" placeholder="用逗号分隔，例如：script.py,--env=prod" />
            </t-col>
          </template>

          <template v-if="createForm.executor_type === 'shell'">
            <t-col :xs="12" :md="4">
              <div class="field-label">Shell</div>
              <t-input v-model="createForm.shell_shell" placeholder="默认 /bin/sh" />
            </t-col>
            <t-col :xs="12" :md="4">
              <div class="field-label">超时（秒）</div>
              <t-input-number v-model="createForm.shell_timeout_seconds" theme="column" :min="1" />
            </t-col>
            <t-col :xs="12" :md="4">
              <div class="field-label">是否启用</div>
              <div class="switch-row">
                <t-switch v-model="createForm.enabled" />
                <span>{{ createForm.enabled ? '启用' : '暂停' }}</span>
              </div>
            </t-col>
            <t-col :xs="12">
              <div class="field-label">Shell 脚本</div>
              <t-textarea v-model="createForm.shell_script" :autosize="{ minRows: 6, maxRows: 10 }" />
            </t-col>
          </template>
        </t-row>
      </div>

      <template #footer>
        <t-space>
          <t-button variant="outline" @click="createVisible = false">取消</t-button>
          <t-button theme="primary" :loading="submitting" @click="handleCreate">创建任务</t-button>
        </t-space>
      </template>
    </t-dialog>

    <t-drawer v-model:visible="detailVisible" size="720px" header="任务详情">
      <div v-if="selectedDetail" class="detail-panel">
        <t-descriptions bordered :column="1" size="small">
          <t-descriptions-item label="任务名">{{ selectedDetail.Job.Name }}</t-descriptions-item>
          <t-descriptions-item label="描述">{{ selectedDetail.Job.Description || '-' }}</t-descriptions-item>
          <t-descriptions-item label="调度策略">{{ formatSchedule(selectedDetail.Job) }}</t-descriptions-item>
          <t-descriptions-item label="执行器">{{ selectedDetail.Job.Executor.Kind }}</t-descriptions-item>
          <t-descriptions-item label="状态">
            <t-tag :theme="selectedDetail.Job.Enabled ? 'success' : 'warning'" variant="light">
              {{ selectedDetail.Job.Enabled ? '启用' : '暂停' }}
            </t-tag>
          </t-descriptions-item>
          <t-descriptions-item label="下次执行">{{ formatUnixTime(selectedDetail.Job.NextRunAt) }}</t-descriptions-item>
          <t-descriptions-item label="依赖任务">
            {{ selectedDetail.DependencyJobs.map((item) => item.Name).join('，') || '无' }}
          </t-descriptions-item>
        </t-descriptions>

        <div class="detail-section">
          <div class="section-title">最近运行记录</div>
          <t-table
            :data="selectedDetail.Runs"
            :columns="runColumns"
            row-key="ID"
            size="small"
            :pagination="{ defaultPageSize: 6, total: selectedDetail.Runs.length }"
          >
            <template #status="{ row }">
              <t-tag :theme="getStatusTheme(row.Status)" variant="light">{{ row.Status }}</t-tag>
            </template>
            <template #scheduledAt="{ row }">{{ formatUnixTime(row.ScheduledAt) }}</template>
            <template #op="{ row }">
              <t-space size="8">
                <t-link theme="primary" @click="openRunLog(row)">日志</t-link>
                <t-link v-if="canRetryRun(row)" theme="primary" @click="handleRetryRun(row)">重试</t-link>
                <t-link v-if="canCancelRun(row)" theme="danger" @click="handleCancelRun(row)">取消</t-link>
              </t-space>
            </template>
          </t-table>
        </div>
      </div>
      <t-empty v-else description="请选择任务" />
    </t-drawer>

    <t-dialog v-model:visible="logVisible" header="运行日志" width="960px" :footer="false">
      <div class="log-toolbar">
        <span>Run ID：{{ currentRunId || '-' }}</span>
        <t-space>
          <span>日志流</span>
          <t-select v-model="logStream" style="width: 120px" @change="loadRunLog">
            <t-option value="" label="全部" />
            <t-option value="stdout" label="stdout" />
            <t-option value="stderr" label="stderr" />
          </t-select>
        </t-space>
      </div>
      <t-loading :loading="logLoading">
        <pre class="log-box">{{ logContent || '暂无日志内容' }}</pre>
      </t-loading>
    </t-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { SearchIcon } from 'tdesign-icons-vue-next';
import { MessagePlugin, type PrimaryTableCol } from 'tdesign-vue-next';

import {
  canCancelRun,
  canRetryRun,
  cancelRun,
  createJob,
  formatSchedule,
  formatUnixTime,
  getJobDetail,
  getLatestRun,
  getRunLog,
  getStatusTheme,
  listJobs,
  pauseJob,
  resumeJob,
  retryRun,
  triggerJob,
  type CreateJobPayload,
  type Job,
  type JobDetail,
  type JobRun,
} from '@/api/cron';

defineOptions({
  name: 'DashboardDetail',
});

interface JobRow extends Job {
  latestStatus: string;
}

const loading = ref(false);
const submitting = ref(false);
const autoRefresh = ref(true);
const jobs = ref<Job[]>([]);
const detailMap = ref<Record<string, JobDetail>>({});

const filters = ref({
  keyword: '',
  executor: '',
  status: '',
});

const createVisible = ref(false);
const detailVisible = ref(false);
const selectedJobID = ref('');
const binaryArgsText = ref('');

const logVisible = ref(false);
const logLoading = ref(false);
const logContent = ref('');
const logStream = ref('');
const currentRunId = ref('');

const jobColumns: PrimaryTableCol[] = [
  { title: '任务名', colKey: 'Name', width: 180, ellipsis: true },
  { title: '描述', colKey: 'Description', ellipsis: true },
  { title: '执行器', colKey: 'executor', width: 100 },
  { title: '调度策略', colKey: 'Schedule', width: 180, cell: (_, row) => formatSchedule(row as Job) },
  { title: '最近状态', colKey: 'latestStatus', width: 110 },
  { title: '启用状态', colKey: 'enabled', width: 100 },
  { title: '下次执行', colKey: 'nextRunAt', width: 180 },
  { title: '操作', colKey: 'op', width: 220, fixed: 'right' },
];

const runColumns: PrimaryTableCol[] = [
  { title: 'Run ID', colKey: 'ID', ellipsis: true },
  { title: '状态', colKey: 'status', width: 110 },
  { title: '触发方式', colKey: 'TriggerType', width: 120 },
  { title: '调度时间', colKey: 'scheduledAt', width: 180 },
  { title: '操作', colKey: 'op', width: 160, fixed: 'right' },
];

function defaultCreateForm(): CreateJobPayload {
  return {
    name: '',
    description: '',
    enabled: true,
    cron: '',
    time_zone: 'Asia/Shanghai',
    interval_seconds: 300,
    executor_type: 'shell',
    concurrency_policy: 'Forbid',
    max_retries: 0,
    initial_backoff_seconds: 30,
    max_backoff_seconds: 300,
    backoff_multiple: 2,
    sdk_protocol: 'http',
    sdk_url: '',
    sdk_method: 'POST',
    sdk_timeout_seconds: 15,
    binary_command: '',
    binary_args: [],
    binary_timeout_seconds: 60,
    dependency_ids: [],
    shell_script: 'echo "hello cron job"',
    shell_shell: '/bin/sh',
    shell_timeout_seconds: 60,
  };
}

const createForm = ref<CreateJobPayload>(defaultCreateForm());

const allRows = computed<JobRow[]>(() => {
  return jobs.value.map((job) => ({
    ...job,
    latestStatus: getLatestRun(detailMap.value[job.ID])?.Status || 'Pending',
  }));
});

const filteredRows = computed(() => {
  const keyword = filters.value.keyword.trim().toLowerCase();
  return allRows.value.filter((row) => {
    if (keyword) {
      const text = `${row.Name} ${row.Description} ${row.ID}`.toLowerCase();
      if (!text.includes(keyword)) {
        return false;
      }
    }
    if (filters.value.executor && row.Executor.Kind !== filters.value.executor) {
      return false;
    }
    if (filters.value.status && row.latestStatus !== filters.value.status) {
      return false;
    }
    return true;
  });
});

const summaryCards = computed(() => {
  const total = jobs.value.length;
  const enabled = jobs.value.filter((job) => job.Enabled).length;
  const running = allRows.value.filter((row) => row.latestStatus === 'Running').length;
  const unhealthy = allRows.value.filter((row) => ['Failed', 'TimedOut', 'Canceled'].includes(row.latestStatus)).length;

  return [
    { label: '总任务数', value: total, hint: '统一接入调度与执行' },
    { label: '启用中', value: enabled, hint: `暂停 ${total - enabled} 个` },
    { label: '运行中', value: running, hint: '当前正在执行' },
    { label: '异常任务', value: unhealthy, hint: '需重点关注' },
  ];
});

const selectedDetail = computed(() => {
  if (!selectedJobID.value) {
    return undefined;
  }
  return detailMap.value[selectedJobID.value];
});

async function fetchAllData(showSuccess = false) {
  if (loading.value) {
    return;
  }

  loading.value = true;
  try {
    const jobList = await listJobs();
    jobs.value = jobList;

    // 并发补齐每个任务的详情，保证详情抽屉和状态统计都是真实数据。
    const detailEntries = await Promise.allSettled(
      jobList.map(async (job) => ({
        id: job.ID,
        detail: await getJobDetail(job.ID),
      })),
    );

    const nextMap: Record<string, JobDetail> = {};
    detailEntries.forEach((item) => {
      if (item.status === 'fulfilled') {
        nextMap[item.value.id] = item.value.detail;
      }
    });
    detailMap.value = nextMap;

    if (showSuccess) {
      MessagePlugin.success('任务数据已刷新');
    }
  } catch (error) {
    console.error(error);
    MessagePlugin.error((error as Error).message || '加载任务失败');
  } finally {
    loading.value = false;
  }
}

function openCreateDialog() {
  createForm.value = defaultCreateForm();
  binaryArgsText.value = '';
  createVisible.value = true;
}

function validateCreateForm() {
  if (!createForm.value.name.trim()) {
    MessagePlugin.warning('请输入任务名');
    return false;
  }
  if (!createForm.value.cron.trim() && createForm.value.interval_seconds <= 0) {
    MessagePlugin.warning('请填写 Cron 或 interval_seconds');
    return false;
  }
  if (createForm.value.executor_type === 'sdk' && !createForm.value.sdk_url.trim()) {
    MessagePlugin.warning('请填写 SDK 地址');
    return false;
  }
  if (createForm.value.executor_type === 'binary' && !createForm.value.binary_command.trim()) {
    MessagePlugin.warning('请填写 binary 命令');
    return false;
  }
  if (createForm.value.executor_type === 'shell' && !createForm.value.shell_script.trim()) {
    MessagePlugin.warning('请填写 shell 脚本');
    return false;
  }
  return true;
}

async function handleCreate() {
  if (!validateCreateForm()) {
    return;
  }

  submitting.value = true;
  try {
    const payload: CreateJobPayload = {
      ...createForm.value,
      name: createForm.value.name.trim(),
      description: createForm.value.description.trim(),
      sdk_url: createForm.value.sdk_url.trim(),
      binary_command: createForm.value.binary_command.trim(),
      shell_script: createForm.value.shell_script.trim(),
      binary_args: binaryArgsText.value
        .split(',')
        .map((item) => item.trim())
        .filter(Boolean),
    };

    await createJob(payload);
    createVisible.value = false;
    MessagePlugin.success('任务创建成功');
    await fetchAllData();
  } catch (error) {
    console.error(error);
    MessagePlugin.error((error as Error).message || '创建任务失败');
  } finally {
    submitting.value = false;
  }
}

async function openDetail(row: Job) {
  selectedJobID.value = row.ID;
  detailVisible.value = true;
  if (!detailMap.value[row.ID]) {
    try {
      detailMap.value[row.ID] = await getJobDetail(row.ID);
    } catch (error) {
      console.error(error);
      MessagePlugin.error((error as Error).message || '加载详情失败');
    }
  }
}

async function handleTrigger(row: Job) {
  try {
    await triggerJob(row.ID);
    MessagePlugin.success(`已触发任务：${row.Name}`);
    await fetchAllData();
  } catch (error) {
    console.error(error);
    MessagePlugin.error((error as Error).message || '触发任务失败');
  }
}

async function handleToggle(row: Job) {
  try {
    if (row.Enabled) {
      await pauseJob(row.ID);
      MessagePlugin.success(`已暂停：${row.Name}`);
    } else {
      await resumeJob(row.ID);
      MessagePlugin.success(`已恢复：${row.Name}`);
    }
    await fetchAllData();
  } catch (error) {
    console.error(error);
    MessagePlugin.error((error as Error).message || '更新任务状态失败');
  }
}

async function handleRetryRun(run: JobRun) {
  try {
    await retryRun(run.ID);
    MessagePlugin.success('已提交重试');
    await fetchAllData();
  } catch (error) {
    console.error(error);
    MessagePlugin.error((error as Error).message || '重试失败');
  }
}

async function handleCancelRun(run: JobRun) {
  try {
    await cancelRun(run.ID);
    MessagePlugin.success('运行已取消');
    await fetchAllData();
  } catch (error) {
    console.error(error);
    MessagePlugin.error((error as Error).message || '取消运行失败');
  }
}

async function openRunLog(run: JobRun) {
  currentRunId.value = run.ID;
  logStream.value = '';
  logVisible.value = true;
  await loadRunLog();
}

async function loadRunLog() {
  if (!currentRunId.value) {
    return;
  }

  logLoading.value = true;
  try {
    const result = await getRunLog(currentRunId.value, logStream.value);
    logContent.value = result.content || '';
  } catch (error) {
    console.error(error);
    MessagePlugin.error((error as Error).message || '读取日志失败');
  } finally {
    logLoading.value = false;
  }
}

let refreshTimer: number | undefined;

onMounted(() => {
  fetchAllData();
  refreshTimer = window.setInterval(() => {
    if (autoRefresh.value) {
      fetchAllData();
    }
  }, 15000);
});

onUnmounted(() => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer);
  }
});
</script>

<style scoped lang="less">
.job-center {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.panel-card {
  :deep(.t-card__body) {
    padding-top: 12px;
  }
}

.toolbar {
  flex-wrap: wrap;
  justify-content: flex-end;
}

.toolbar-search {
  width: 240px;
}

.switch-text {
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.summary-row {
  margin-bottom: 12px;
}

.summary-item {
  padding: 12px 14px;
  border-radius: var(--td-radius-medium);
  background: var(--td-bg-color-secondarycontainer);
}

.summary-label {
  font-size: 13px;
  color: var(--td-text-color-secondary);
}

.summary-value {
  margin-top: 8px;
  font-size: 26px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.summary-hint {
  margin-top: 8px;
  font-size: 12px;
  color: var(--td-text-color-placeholder);
}

.create-form {
  max-height: 65vh;
  overflow: auto;
}

.field-label {
  margin-bottom: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.switch-row {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 32px;
}

.detail-panel {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.detail-section {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.log-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}

.log-box {
  min-height: 360px;
  max-height: 60vh;
  overflow: auto;
  margin: 0;
  padding: 16px;
  white-space: pre-wrap;
  word-break: break-word;
  border-radius: var(--td-radius-medium);
  background: #0f172a;
  color: #dbeafe;
}
</style>
