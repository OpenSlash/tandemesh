# Agent Session

Agent Session 是跨多次 Run 的持续对话。任务仍然是一次性执行，Session 把用户与 Agent 的上下文串起来。

## 模型

```
AgentSession 1 ── n Run ── n Dispatch
```

Session 保存：

- `agent_id` / `backend`
- `mode`：`chat | terminal`
- `status`：`active | closed`
- `backend_session_id`：ACP 后端会话 ID，用于 `session/load` 恢复
- `last_run_id`

Run 新增 `session_id`，一条消息对应一次 Run，避免长连接依赖。

## 创建

```http
POST /api/v1/sessions
{
  "workspaceId": "ws_default",
  "taskId": "task_...",       // 可选
  "agentId": "claude",
  "mode": "chat",
  "title": "Refactor chat"
}
```

任务下的 Session 默认继承任务标题与 assignee。

## 发消息

```http
POST /api/v1/sessions/{id}/runs
{ "prompt": "继续实现剩余部分" }
```

控制面创建一条 `role=chat` 的 Run，并关联 `session_id`。Daemon 处理时：

1. 从 Session 读取 `backend_session_id`
2. lease 中携带 `sessionId` 与 `sessionRecordId`
3. ACP 先调用 `session/load`，失败则 `session/new`
4. 首条消息后把后端 session ID 回写到 Session

无 repo 的 Session 使用稳定目录 `~/.orca/sessions/<sessionId>`，保证续聊 cwd 不变。

## 消息

```http
GET /api/v1/sessions/{id}/messages
```

按 Run 顺序聚合：

- `role=user`：Run 的 prompt
- `role=assistant`：Agent 文本与工具事件
- `role=system`：状态与错误

## 其他接口

```
GET  /api/v1/sessions?workspaceId=&taskId=&status=
GET  /api/v1/sessions/{id}
GET  /api/v1/sessions/{id}/runs
POST /api/v1/sessions/{id}/close
```

## CLI

```bash
orca session list --workspace=ws_default
orca session new --title="Refactor" --agent=claude --mode=chat
orca session send --session=<id> --prompt="继续"
orca session close --session=<id>
```

## 前端

- `/sessions`：会话列表与创建
- `/sessions/:id`：对话记录、发送消息、关闭会话

## Terminal 模式

当前 `mode=terminal` 与 chat 共用 ACP 通道，记录为终端型会话；真正的 PTY 流式终端仍待接入，后续会在协议中增加 `terminal.*` 事件。