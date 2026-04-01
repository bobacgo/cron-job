CREATE TABLE IF NOT EXISTS jobs (
    id                                 VARCHAR(64)    NOT NULL                          COMMENT '作业唯一 ID',
    name                               VARCHAR(255)   NOT NULL                          COMMENT '作业名称',
    description                        TEXT           NOT NULL                          COMMENT '作业描述',
    enabled                            TINYINT(1)     NOT NULL DEFAULT 1                COMMENT '是否启用：1 启用，0 禁用',
    -- 调度配置
    schedule_cron                      VARCHAR(128)   NOT NULL DEFAULT ''               COMMENT 'Cron 表达式，与 interval 二选一',
    schedule_interval_seconds          BIGINT         NOT NULL DEFAULT 0                COMMENT '固定间隔调度（秒），与 cron 二选一',
    schedule_time_zone                 VARCHAR(64)    NOT NULL DEFAULT ''               COMMENT '调度时区，如 Asia/Shanghai',
    schedule_starting_deadline_seconds INT            NOT NULL DEFAULT 0                COMMENT '错过调度后的补偿截止窗口（秒）',
    -- 执行器配置
    executor_kind                      VARCHAR(32)    NOT NULL                          COMMENT '执行器类型：sdk / binary / shell',
    sdk_protocol                       VARCHAR(32)    NOT NULL DEFAULT ''               COMMENT 'SDK 协议：http / grpc',
    sdk_url                            VARCHAR(512)   NOT NULL DEFAULT ''               COMMENT 'SDK 服务地址',
    sdk_method                         VARCHAR(128)   NOT NULL DEFAULT ''               COMMENT 'SDK 调用方法',
    sdk_timeout_seconds                BIGINT         NOT NULL DEFAULT 0                COMMENT 'SDK 超时时间（秒）',
    binary_command                     VARCHAR(512)   NOT NULL DEFAULT ''               COMMENT '二进制可执行文件路径',
    binary_args_json                   TEXT           NOT NULL                          COMMENT '二进制参数列表（JSON 数组）',
    binary_timeout_seconds             BIGINT         NOT NULL DEFAULT 0                COMMENT '二进制执行超时（秒）',
    shell_script                       TEXT           NOT NULL                          COMMENT 'Shell 内联脚本内容',
    shell_shell                        VARCHAR(128)   NOT NULL DEFAULT ''               COMMENT 'Shell 解释器路径，默认 /bin/sh',
    shell_timeout_seconds              BIGINT         NOT NULL DEFAULT 0                COMMENT 'Shell 执行超时（秒）',
    -- 重试策略
    retry_max_retries                  INT            NOT NULL DEFAULT 0                COMMENT '最大重试次数',
    retry_initial_backoff_seconds      BIGINT         NOT NULL DEFAULT 0                COMMENT '首次重试退避时间（秒）',
    retry_max_backoff_seconds          BIGINT         NOT NULL DEFAULT 0                COMMENT '最大退避时间（秒）',
    retry_backoff_multiple             DOUBLE         NOT NULL DEFAULT 1                COMMENT '退避倍数',
    -- 其他
    concurrency_policy                 VARCHAR(32)    NOT NULL DEFAULT 'Allow'          COMMENT '并发策略：Allow / Forbid / Replace',
    next_run_at                        DATETIME(3)    NOT NULL DEFAULT '0001-01-01 00:00:00.000' COMMENT '下次预计执行时间',
    last_run_at                        DATETIME(3)    NOT NULL DEFAULT '0001-01-01 00:00:00.000' COMMENT '上次执行时间',
    last_success_at                    DATETIME(3)    NOT NULL DEFAULT '0001-01-01 00:00:00.000' COMMENT '上次成功执行时间',
    created_at                         DATETIME(3)    NOT NULL DEFAULT '0001-01-01 00:00:00.000' COMMENT '创建时间',
    updated_at                         DATETIME(3)    NOT NULL DEFAULT '0001-01-01 00:00:00.000' COMMENT '更新时间',
    PRIMARY KEY (id),
    INDEX idx_jobs_enabled (enabled),
    INDEX idx_jobs_next_run_at (next_run_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='作业定义表';

CREATE TABLE IF NOT EXISTS job_runs (
    id           VARCHAR(64)  NOT NULL                                   COMMENT '运行记录唯一 ID',
    job_id       VARCHAR(64)  NOT NULL                                   COMMENT '所属作业 ID',
    scheduled_at DATETIME(3)  NOT NULL DEFAULT '0001-01-01 00:00:00.000' COMMENT '计划执行时间',
    started_at   DATETIME(3)  NOT NULL DEFAULT '0001-01-01 00:00:00.000' COMMENT '实际开始时间',
    finished_at  DATETIME(3)  NOT NULL DEFAULT '0001-01-01 00:00:00.000' COMMENT '结束时间',
    status       VARCHAR(32)  NOT NULL                                   COMMENT '执行状态：Pending / Blocked / Ready / Running / Succeeded / Failed / TimedOut / Canceled / Skipped',
    attempt      INT          NOT NULL DEFAULT 0                         COMMENT '当前重试次数（从 0 开始）',
    trigger_type VARCHAR(64)  NOT NULL DEFAULT ''                        COMMENT '触发类型：scheduler / manual 等',
    message      TEXT         NOT NULL                                   COMMENT '执行结果或错误信息',
    created_at   DATETIME(3)  NOT NULL DEFAULT '0001-01-01 00:00:00.000' COMMENT '创建时间',
    updated_at   DATETIME(3)  NOT NULL DEFAULT '0001-01-01 00:00:00.000' COMMENT '更新时间',
    dedup_key    VARCHAR(128) NOT NULL DEFAULT ''                        COMMENT '幂等键，格式：job_id/scheduled_at，防止重复调度',
    PRIMARY KEY (id),
    UNIQUE INDEX idx_job_runs_dedup_key (dedup_key),
    INDEX idx_job_runs_job_id (job_id),
    INDEX idx_job_runs_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='作业运行记录表';

CREATE TABLE IF NOT EXISTS dependencies (
    job_id            VARCHAR(64) NOT NULL COMMENT '依赖方作业 ID',
    depends_on_job_id VARCHAR(64) NOT NULL COMMENT '被依赖的作业 ID',
    PRIMARY KEY (job_id, depends_on_job_id),
    INDEX idx_dependencies_job_id (job_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='作业依赖关系表';