# Diff 批阅

Diff 批阅把一次 Run 的代码变更和行级批注放在同一视图，批注可直接发回 Agent。

## 采集

AgentDaemon 在 Run 结束、`Finish` 之前执行：

```text
git merge-base HEAD <base>
git diff --no-color <baseRef>...HEAD
git diff --no-color
```

- 同时覆盖已提交和未提交改动
- 仅在 worktree 是 git 仓库时采集
- 统一 diff 最大 1 MB，超出标记 `truncated`
- 通过事件 `workspace.diff` 上报，payload：

```json
{ "diff": "diff --git ...", "truncated": false }
```

## 读取

```http
GET /api/v1/runs/{id}/diff
```

返回：

```json
{ "runId": "run_...", "diff": "...", "truncated": false }
```

控制面从该 Run 的事件尾部查找最新 `workspace.diff`。

## 批注

沿用现有接口：

```
GET  /api/v1/tasks/{id}/annotations?runId=...
POST /api/v1/tasks/{id}/annotations
POST /api/v1/annotations/{id}/resolve
POST /api/v1/tasks/{id}/annotations/send
```

批注锚定 `filePath + line + side`：

- 新增行：`side=new`，行号为新文件行号
- 删除行：`side=old`，行号为旧文件行号

`send` 会把所有 open 批注聚合成一条 `review` AgentMessage，逐条列出文件、行号和意见。

## 前端

任务详情新增 Diff 面板：

- 按文件渲染 unified diff，保留双行号
- 点击任意行即可写批注
- 行下展示已有批注，可标记 resolved
- 顶部可一键发送全部 open 批注给 Agent
- Run 运行中每 5 秒刷新 diff 和批注

组件：`web/src/components/task-diff-panel.tsx`