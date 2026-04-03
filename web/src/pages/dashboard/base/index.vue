<template>
  <div class="overview-page">
    <t-row :gutter="[16, 16]">
      <t-col v-for="card in summaryCards" :key="card.label" :xs="6" :xl="3">
        <t-card :bordered="false" class="metric-card">
          <div class="metric-label">{{ card.label }}</div>
          <div class="metric-value">{{ card.value }}</div>
          <div class="metric-hint">{{ card.hint }}</div>
        </t-card>
      </t-col>
    </t-row>

    <t-row class="section-gap" :gutter="[16, 16]">
      <t-col :xs="12" :xl="7">
        <t-card title="任务总览" :bordered="false" class="panel-card">
          <template #actions>
            <t-space>
              <t-button variant="outline" :loading="loading" @click="refreshData(true)">刷新</t-button>
              <t-button theme="primary" @click="router.push('/dashboard/detail')">进入任务中心</t-button>
            </t-space>
          </template>

          <t-table
            :data="jobRows"
            :columns="jobColumns"
            row-key="ID"
            size="small"
            :loading="loading"
            :pagination="{ defaultPageSize: 6, total: jobRows.length }"
          >
            <template #enabled="{ row }">
              <t-tag :theme="row.Enabled ? 'success' : 'warning'" variant="light">
                {{ row.Enabled ? '启用' : '暂停' }}
              </t-tag>
            </template>
            <template #latestStatus="{ row }">
              <t-tag :theme="getStatusTheme(row.latestStatus)" variant="light">
                {{ row.latestStatus }}
              </t-tag>
            </template>
            <template #nextRunAt="{ row }">
              {{ formatUnixTime(row.NextRunAt) }}
            </template>
            <template #op>
              <t-link theme="primary" @click="router.push('/dashboard/detail')">管理</t-link>
            </template>
          </t-table>
        </t-card>
      </t-col>

      <t-col :xs="12" :xl="5">
        <t-card title="最近运行" :bordered="false" class="panel-card">
          <div v-if="recentRuns.length" class="recent-run-list">
            <div v-for="run in recentRuns" :key="run.ID" class="recent-run-item">
              <div>
                <div class="run-title">{{ run.jobName }}</div>
                <div class="run-meta">
                  {{ formatUnixTime(run.ScheduledAt) }} · {{ run.TriggerType || 'auto' }}
                </div>
              </div>
              <div class="run-actions">
                <t-tag :theme="getStatusTheme(run.Status)" variant="light">{{ run.Status }}</t-tag>
                <t-button variant="text" theme="primary" @click="openRunLog(run)">日志</t-button>
              </div>
            </div>
          </div>
          <t-empty v-else description="暂无运行记录" />
        </t-card>
      </t-col>
    </t-row>

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
import { computed, onMounted, ref } from 'vue';
import { useRouter } from 'vue-router';
import { MessagePlugin, type PrimaryTableCol } from 'tdesign-vue-next';

import {
  formatSchedule,
  formatUnixTime,
  getJobDetail,
  getLatestRun,
  getRunLog,
  getStatusTheme,
  listJobs,
  type Job,
  type JobDetail,
  type JobRun,
} from '@/api/cron';

defineOptions({
  name: 'DashboardBase',
});

interface OverviewJobRow extends Job {
  scheduleText: string;
  latestStatus: string;
}

interface RecentRunRow extends JobRun {
  jobName: string;
}

const router = useRouter();
const loading = ref(false);
const jobs = ref<Job[]>([]);
const detailMap = ref<Record<string, JobDetail>>({});

const logVisible = ref(false);
const logLoading = ref(false);
const logContent = ref('');
const logStream = ref('');
const currentRunId = ref('');

const jobColumns: PrimaryTableCol[] = [
  { title: '任务名', colKey: 'Name', ellipsis: true },
  { title: '调度策略', colKey: 'scheduleText', ellipsis: true },
  { title: '最近状态', colKey: 'latestStatus', width: 110 },
  { title: '开关', colKey: 'enabled', width: 90 },
  { title: '下次执行', colKey: 'nextRunAt', width: 180 },
  { title: '操作', colKey: 'op', width: 80, align: 'center' },
];

const jobRows = computed<OverviewJobRow[]>(() => {
  return jobs.value.map((job) => ({
    ...job,
    scheduleText: formatSchedule(job),
    latestStatus: getLatestRun(detailMap.value[job.ID])?.Status || 'Pending',
  }));
});

const recentRuns = computed<RecentRunRow[]>(() => {
  return Object.values(detailMap.value)
    .flatMap((detail) =>
      detail.Runs.map((run) => ({
        ...run,
        jobName: detail.Job.Name,
      })),
    )
    .sort((a, b) => (b.ScheduledAt || 0) - (a.ScheduledAt || 0))
    .slice(0, 8);
});

const summaryCards = computed(() => {
  const total = jobs.value.length;
  const enabled = jobs.value.filter((job) => job.Enabled).length;
  const paused = total - enabled;
  const running = recentRuns.value.filter((run) => run.Status === 'Running').length;
  const failed = recentRuns.value.filter((run) => ['Failed', 'TimedOut', 'Canceled'].includes(run.Status)).length;

  return [
    { label: '总任务数', value: total, hint: '当前已接入的任务规模' },
    { label: '启用中', value: enabled, hint: `暂停 ${paused} 个` },
    { label: '运行中', value: running, hint: '实时执行中的任务数' },
    { label: '异常运行', value: failed, hint: '最近运行失败 / 超时 / 取消' },
  ];
});

async function refreshData(showSuccess = false) {
  if (loading.value) {
    return;
  }

  loading.value = true;
  try {
    const jobList = await listJobs();
    jobs.value = jobList;

    // 把详情并发拉取下来，页面统计和最近运行都直接基于真实数据展示。
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
      MessagePlugin.success('数据已刷新');
    }
  } catch (error) {
    console.error(error);
    MessagePlugin.error((error as Error).message || '加载任务数据失败');
  } finally {
    loading.value = false;
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

onMounted(() => {
  refreshData();
});
</script>

<style scoped lang="less">
.overview-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section-gap {
  margin-top: 0;
}

.metric-card {
  padding: 8px;

  :deep(.t-card__body) {
    padding: 0;
  }
}

.metric-label {
  color: var(--td-text-color-secondary);
  font-size: 13px;
}

.metric-value {
  margin-top: 8px;
  font-size: 28px;
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.metric-hint {
  margin-top: 8px;
  color: var(--td-text-color-placeholder);
  font-size: 12px;
}

.panel-card {
  :deep(.t-card__body) {
    padding-top: 12px;
  }
}

.recent-run-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.recent-run-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  border-radius: var(--td-radius-medium);
  background: var(--td-bg-color-secondarycontainer);
}

.run-title {
  font-weight: 600;
  color: var(--td-text-color-primary);
}

.run-meta {
  margin-top: 4px;
  font-size: 12px;
  color: var(--td-text-color-secondary);
}

.run-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
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
