# Cron Job

一个用 Go 编写的定时任务配置管理系统骨架，当前实现采用“模块化单体 + 控制循环内核”的路线，支持：

- 任务配置管理
- `interval` 和 `cron` 两种调度方式
- SDK 任务与二进制任务两类执行模型
- HTTP transport 的 SDK 执行
- gRPC transport 的 SDK 执行
- 任务依赖 DAG 基础能力
- 手动触发
- 暂停和恢复任务
- 运行记录和状态跟踪
- 运行日志分流落盘（stdout/stderr）和检索
- Go template + Web Components 后台页面
- 登录认证（会话 Cookie）

## 当前能力

当前版本已经实现一个可运行的 MVP：

- `Job` 和 `JobRun` 分离建模
- `ScheduleLoop` 根据调度规则生成 `JobRun`
- `DependencyLoop` 处理被依赖阻塞的任务
- `RunLoop` 从 ready queue 取任务并调用执行器
- SQLite 版 `Repository`、内存版 `Queue`/`Lease`
- 文件版任务日志存储
- 日志流过滤和关键字检索
- JSON API
- 后台任务列表、创建表单、任务详情页、手动触发、暂停恢复、日志页、登录/退出

依赖语义目前是一个务实版实现：

- 下游任务创建或手动触发时，如果存在依赖，会先进入 `Blocked`
- 当所有上游任务的最新一次运行状态为 `Succeeded` 时，下游会被释放到 `Ready`
- 当前没有做复杂 trigger rule，也没有按“同一调度窗口”精确关联上下游 run

## 目录结构

```text
cron-job/
├── cmd/server                  # 服务入口
├── internal/app                # 用例编排
├── internal/domain             # 领域模型和状态机
├── internal/scheduler          # 调度和依赖释放循环
├── internal/dispatcher         # ready queue、lease、run loop
├── internal/executor           # SDK/Binary 执行器
├── internal/repository         # 存储接口和内存实现
├── internal/transport          # HTTP API 和后台页面
├── web/templates               # Go template 页面
├── migrations                  # 迁移脚本占位
└── README.md
```

## 启动方式

要求：

- Go 1.26+

启动：

```bash
go run ./cmd/server
```

默认监听：

```bash
:8080
```

可选：启动一个本地 gRPC SDK worker（用于 `sdk_protocol=grpc` 任务联调）：

```bash
go run ./cmd/sdk-worker -addr :50051
```

可通过环境变量覆盖：

```bash
HTTP_ADDR=:9090 go run ./cmd/server
```

日志目录也可覆盖：

```bash
LOG_DIR=./tmp/logs go run ./cmd/server
```

SQLite 数据库路径也可覆盖：

```bash
DB_PATH=./data/cron-job.db go run ./cmd/server
```

后台登录账号也可通过环境变量覆盖：

```bash
ADMIN_USER=admin ADMIN_PASSWORD=admin123 go run ./cmd/server
```

## 已验证命令

```bash
gofmt -w ./cmd ./internal && go test ./...
```

## API

### 健康检查

```http
GET /api/v1/healthz
```

### 任务列表

```http
GET /api/v1/jobs
```

### 创建任务

```http
POST /api/v1/jobs
Content-Type: application/json
```

SDK 任务示例：

```json
{
  "name": "sync-user-cache",
  "description": "refresh user cache every minute",
  "enabled": true,
  "cron": "*/1 * * * *",
  "time_zone": "UTC",
  "executor_type": "sdk",
  "sdk_protocol": "http",
  "sdk_url": "http://127.0.0.1:9000/task/run",
  "sdk_method": "POST",
  "sdk_timeout_seconds": 10
}
```

gRPC SDK 任务示例：

```json
{
  "name": "grpc-sync-job",
  "description": "invoke grpc sdk endpoint",
  "enabled": true,
  "interval_seconds": 300,
  "executor_type": "sdk",
  "sdk_protocol": "grpc",
  "sdk_url": "127.0.0.1:50051",
  "sdk_method": "/cronjob.v1.Executor/Run"
}
```

当前 gRPC transport 采用 JSON codec 调用固定 RPC 方法，适合作为 SDK transport 骨架；正式接入时建议把请求响应模型和 proto 一起固化下来。

二进制任务示例：

```json
{
  "name": "backup-job",
  "description": "local backup script",
  "enabled": true,
  "interval_seconds": 3600,
  "executor_type": "binary",
  "binary_command": "/bin/echo",
  "binary_args": ["backup finished"]
}
```

带依赖的任务示例：

```json
{
  "name": "downstream-job",
  "description": "run after upstream succeeds",
  "enabled": true,
  "interval_seconds": 3600,
  "executor_type": "binary",
  "binary_command": "/bin/echo",
  "binary_args": ["downstream"],
  "dependency_ids": ["upstream-job-id"]
}
```

### 查询任务详情

```http
GET /api/v1/jobs/{jobID}
```

会返回：

- 任务定义
- 依赖边
- 依赖任务摘要
- 任务运行记录

### 手动触发

```http
POST /api/v1/jobs/{jobID}/trigger
```

### 暂停任务

```http
POST /api/v1/jobs/{jobID}/pause
```

### 恢复任务

```http
POST /api/v1/jobs/{jobID}/resume
```

### 读取运行日志

```http
GET /api/v1/job-runs/{runID}/logs
```

可按流读取：

```http
GET /api/v1/job-runs/{runID}/logs?stream=stdout
GET /api/v1/job-runs/{runID}/logs?stream=stderr
```

### 日志检索

```http
GET /api/v1/logs/search?q=timeout&stream=stderr&run_id={runID}&limit=100
```

### 取消运行

```http
POST /api/v1/job-runs/{runID}/cancel
```

### 重试运行

```http
POST /api/v1/job-runs/{runID}/retry
```

## 后台页面

- `/login`：登录页面
- `/`：仪表盘
- `/jobs`：任务列表 + 创建表单
- `/jobs/{jobID}`：任务详情 + 最近运行记录 + 手动触发 + 暂停恢复
- `/job-runs/{runID}/logs`：运行日志查看

## 当前限制

当前实现仍然是骨架阶段，下面这些能力还没有完成：

- 更完整的重试和退避策略
- 依赖图可视化页面
- 更细粒度的 RBAC
- 按调度窗口关联的依赖编排
- 分布式 worker 和高可用调度

## 建议的下一步

1. 补充 PostgreSQL 仓储实现，并增加迁移版本管理。
2. 为取消动作增加运行中任务的协作式中断能力。
3. 完善 gRPC SDK 协议（proto 固化、版本协商、错误码语义）。
4. 扩展测试覆盖到 dispatcher 重试链路与 API handler。
5. 增加依赖图可视化和运维审计页。
