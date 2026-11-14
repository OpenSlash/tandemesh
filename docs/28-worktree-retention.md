# Worktree 物理清理与 retention

控制面只保存 Worktree 投影；磁盘目录在执行机上。本模块通过清理队列让执行机真正删除目录，并提供保留策略。

## 数据

`worktree_cleanups`：

| 字段 | 说明 |
| --- | --- |
| `worktree_id` | 对应 worktrees 记录 |
| `run_id` | 用于 `workspace.Manager.Cleanup` |
| `path` | 目录路径，mgr 不可用时直接删除 |
| `machine_id` | 目标执行机，取自 Run |
| `status` | `pending / done / failed` |
| `attempts` | 执行次数 |
| `error` | 失败原因 |

同一 worktree 只允许一个 pending 清理请求（部分唯一索引）。

## 触发清理

1. **归档时自动入队**：`POST /worktrees/{id}/archive`
2. **手动入队**：`POST /worktrees/{id}/cleanup`
3. **保留策略**：`POST /worktrees/retention/apply`

```json
{ "workspaceId": "ws_default", "olderThanDays": 7 }
```

保留策略会：

- 找出创建时间早于 cutoff 且未归档的 Worktree
- 逐个归档
- 为有执行机的记录入队物理清理

候选查询：

```http
GET /api/v1/worktrees/retention?workspaceId=ws_default&olderThanDays=7
```

## 执行机

AgentDaemon 每 5 秒拉取：

```http
GET /api/v1/agentd/worktree-cleanups?machineId=mch_...
```

执行：

- 有 workspace manager：`mgr.Cleanup(runId)`，注销 git worktree 并删除目录
- 否则：`os.RemoveAll(path)`
- 回报：

```http
POST /api/v1/agentd/worktree-cleanups/{id}
{"status":"done"}
```

成功后控制面把 Worktree 状态置为 `removed`。

## 定期 GC

AgentDaemon 启动时执行一次 `workspace.Manager.GC`，之后每 6 小时执行一次：

- 删除超过 `--retain`（默认 7 天）的 run 目录
- 对每个 repo cache 执行 `git worktree prune`

命令行：

```bash
orca-agentd --workspace --retain=168h --data-dir=~/.orca
```

## 前端

任务执行面板的 Worktree 卡片：

- 归档后出现「从磁盘删除」按钮
- 清理完成后状态变为 `removed`

## 注意

- 清理请求绑定创建 Run 的执行机；换机后旧目录需要原机器在线
- `retention/apply` 不会删除未分配执行机的记录
- 磁盘删除是异步的，API 返回 `202` 只表示已入队