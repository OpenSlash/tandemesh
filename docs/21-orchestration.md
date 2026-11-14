# 编排模型：Dispatch、消息与决策门

这一层把任务从“一次性 Run”升级为可追踪、可恢复、可串联的执行模型。

## 核心概念

### Task

Task 是长期存在的工作项，保留既有字段：

- `status`：`open | in_progress | done`
- `execution_state`：`pending | ready | dispatched | completed | failed | blocked`

`execution_state` 只描述执行编排状态，不替代外部平台状态。

### Dispatch

一次 Dispatch 是某个 Task 的一次执行尝试。

```json
{
  "id": "dsp_...",
  "taskId": "task_...",
  "runId": "run_...",
  "agentId": "claude",
  "attempt": 1,
  "status": "pending | running | succeeded | failed | cancelled | circuit_broken",
  "outcome": "succeeded | failed",
  "result": "summary",
  "filesModified": ["src/a.ts"],
  "failureCount": 0
}
```

规则：

- Task 可以有多个 Dispatch；Run 只是 Dispatch 的执行载体
- 完成时必须带 `taskId` 和 `dispatchId`，防止过期重试完成错的任务
- 连续 3 次失败后禁止继续自动派发（熔断）
- `run.finished` 事件会自动结束当前 active dispatch，避免遗漏上报

### 消息

`agent_messages` 是 Task 内的 Agent 邮箱。

支持地址：

- 具体 run id
- `@all`：同一 Task 下的其他 Run
- `@idle`：非运行中的 Run
- `@agent:<agentId>`：指定 Agent 的 Run

接口：

```
GET  /api/v1/tasks/{id}/messages
POST /api/v1/tasks/{id}/messages
POST /api/v1/tasks/{id}/messages/check?runId=...
POST /api/v1/messages/{id}/ack
```

`check` 按 FIFO 取一条未读消息并标记已处理，Agent 可用它替代 sleep+poll。

### 决策门

`decision_gates` 用于阻塞任务，直到人工记录决定。

```
GET  /api/v1/tasks/{id}/gates
POST /api/v1/tasks/{id}/gates
POST /api/v1/gates/{id}/resolve
```

存在 open gate 时，Dispatch 无法完成，接口返回 `409 E_GATE`。

## Task DAG

依赖表沿用现有 `task_dependencies`。

- 新增依赖后，若上游未完成，任务 `execution_state = pending`
- 上游完成后，满足全部依赖的下游自动提升为 `ready`
- 手动把任务改为 `done`、平台状态映射为 `done`、Dispatch 成功完成，都会触发下游提升

接口：

```
GET    /api/v1/tasks/{id}/dependencies
POST   /api/v1/tasks/{id}/dependencies
DELETE /api/v1/tasks/{id}/dependencies/{dep}
```

## Dispatch 接口

```
GET  /api/v1/tasks/{id}/dispatches
POST /api/v1/tasks/{id}/dispatches
GET  /api/v1/dispatches/{id}
POST /api/v1/dispatches/{id}/start
POST /api/v1/dispatches/{id}/complete
```

完成请求：

```json
{
  "taskId": "task_...",
  "outcome": "succeeded",
  "result": "fixed footer overlap",
  "filesModified": ["src/app.tsx"]
}
```

## Diff 批注回传

`diff_annotations` 保存行级评审意见：

```
GET  /api/v1/tasks/{id}/annotations
POST /api/v1/tasks/{id}/annotations
POST /api/v1/tasks/{id}/annotations/send
POST /api/v1/annotations/{id}/resolve
```

发送时把所有 open 批注合并成一个 prompt，作为 `review` 类型消息投递给目标 Run。
未解决批注会保留，下一次发送可继续带上。

## 前端

任务详情页新增「执行与编排」面板：

- Dispatch 列表与状态
- 决策门创建 / 解决
- Diff 批注创建 / 批量发送 / 解决
- Agent 消息列表与发送