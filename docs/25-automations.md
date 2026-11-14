# Automations 增强

Automation 是按 Cron 计划启动 Run 的无事件触发器。增强后支持预检、错过宽限和会话复用。

## 字段

`automations` 新增：

| 字段 | 说明 |
| --- | --- |
| `precheck` | 启动前执行的 shell 命令 |
| `missed_grace_secs` | 允许迟到多少秒仍然启动；0 表示不限制 |
| `session_id` | 复用会话 ID |
| `reuse_session` | 是否跨运行复用同一会话 |
| `last_status` | `started / precheck_failed / skipped_missed / invalid_schedule / start_failed` |
| `last_error` | 最近一次失败原因 |

## Precheck

配置：

```json
{
  "precheck": "npm run precheck --silent"
}
```

行为：

- 60 秒超时
- 注入 `TANDEMESH_AUTOMATION_ID` / `TANDEMESH_AUTOMATION_NAME`
- 非 0 退出码：不创建 Run，记录 `precheck_failed`，并推进到下一周期
- 输出最多保留 2000 字符写入 `last_error`

手动 `POST /automations/{id}/run` 预检失败返回 `422 E_PRECHECK`。

## Missed grace

```json
{ "missedGraceSeconds": 300 }
```

Scheduler 在启动前检查：

```text
now - nextRun > missedGraceSeconds  →  跳过本轮
```

跳过时记录 `skipped_missed`，并把 `nextRun` 推进到下一个周期，不补跑历史错过的所有周期。

`0` 表示关闭该限制，迟到多久都会启动。

## 复用会话

```json
{ "reuseSession": true }
```

首次运行：

1. 创建一条 `mode=chat` 的 AgentSession
2. 把 Session ID 写回 Automation
3. Run 关联该 Session

后续运行复用同一 `session_id`，Daemon 通过 ACP `session/load` 恢复上下文。

复用会话时 Run 不创建 per-run worktree，而是使用稳定会话目录 `~/.orca/sessions/<sessionId>`；需要独立仓库工作树的自动化请关闭 `reuseSession`。

## 状态

列表与详情返回 `lastStatus` / `lastError`，前端在自动化名称下方显示。

## API

```
GET  /api/v1/automations?workspace=ws_default
POST /api/v1/automations
PUT  /api/v1/automations/{id}
POST /api/v1/automations/{id}/run
```

创建/更新支持 `precheck`、`missedGraceSeconds`、`reuseSession`。