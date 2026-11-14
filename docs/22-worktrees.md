# Worktree 生命周期

Orca 为每次 Run 准备独立 git worktree，避免并行执行互相覆盖文件。控制面保存投影，便于 UI、CLI 与审计使用。

## 生命周期

```
creating → ready → archived
         ↘ failed
```

- `creating`：创建 Run 时记录 worktree 意图（repo/base/branch）
- `ready`：AgentDaemon 完成 `git worktree add` 并上报 `workspace.ready`
- `failed`：准备失败，Run 以 `E_WORKSPACE` 结束
- `archived`：不再使用，可人工归档

Run 的完成不会自动删除 worktree，便于复查、续跑和 checkpoint。

## 数据

`worktrees` 表：

| 字段 | 说明 |
| --- | --- |
| `task_id` / `run_id` | 归属 |
| `repo` | clone URL 或本地路径 |
| `path` | 实际工作目录 |
| `branch` | 运行分支，默认 `orca/<id>` |
| `base_ref` / `start_ref` | 起点分支 |
| `status` | 生命周期状态 |
| `checkpoint_comment` | Agent 或人工写入的进度说明 |
| `workspace_status` | `todo / in-progress / in-review / completed` |

## 仓库约定

### `orca.yaml`

支持 `scripts.setup`：

```yaml
scripts:
  setup: |
    npm ci
    npm run build
```

AgentDaemon 在 worktree 创建后执行，超时 10 分钟。失败时 Run 报 `E_WORKSPACE`。

也支持 `.orca.yaml`。

### `.worktreeinclude`

逐行列出需要从仓库缓存复制到 worktree 的未跟踪文件或目录：

```
.env.local
config/*.local.json
```

支持 glob，复制时保留相对路径。

## API

```
GET  /api/v1/worktrees?workspaceId=&taskId=&status=
GET  /api/v1/worktrees/{id}
GET  /api/v1/runs/{id}/worktree
POST /api/v1/worktrees/{id}/checkpoint
POST /api/v1/runs/{id}/worktree/checkpoint
POST /api/v1/worktrees/{id}/archive
```

Checkpoint：

```json
{
  "comment": "已完成路由拆分，等待测试",
  "workspaceStatus": "in-review"
}
```

## CLI

```bash
orca worktree current --run=<runId>
orca worktree set --run=<runId> --comment="halfway" --workspace-status=in-progress
```

`current` 默认输出可读摘要；加 `--json` 输出原始 JSON。

## 前端

任务详情页「执行与编排」面板会展示最新 worktree：

- 分支、路径、状态
- 写入进度 checkpoint
- 标记归档