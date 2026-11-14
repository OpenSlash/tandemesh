# 真实终端会话

Terminal 模式的 AgentSession 现在可以启动一个真实 PTY，并在网页里直接接管 Shell。

## 架构

```text
Browser ── input/output polling ── Control Plane ── agentd polling ── PTY
```

控制面保存终端会话、输出块和输入队列，执行机上的 AgentDaemon 负责真正启动 PTY。

## 数据

`terminal_sessions`：

| 字段 | 说明 |
| --- | --- |
| `id` | `term_...` |
| `workspace_id` | 工作区 |
| `agent_session_id` | 关联的 terminal 模式 AgentSession |
| `machine_id` | 分配的执行机 |
| `shell` | 空则由 daemon 选择 `pwsh / powershell / cmd / $SHELL` |
| `cwd` | 起始目录，空则 daemon 工作目录 |
| `cols / rows` | PTY 尺寸 |
| `status` | `pending / running / exited / closed` |
| `exit_code` | 退出码 |

`terminal_events` 保存输出块，`terminal_inputs` 保存输入队列。

## API

```http
GET  /api/v1/terminal-sessions?workspace=ws_default
POST /api/v1/terminal-sessions
GET  /api/v1/terminal-sessions/{id}
POST /api/v1/terminal-sessions/{id}/input   {"data":"ls\r"}
GET  /api/v1/terminal-sessions/{id}/output?after=42
POST /api/v1/terminal-sessions/{id}/close
```

Daemon 侧：

```http
GET  /api/v1/agentd/terminals?machineId=mch_...
GET  /api/v1/agentd/terminals/{id}/input
POST /api/v1/agentd/terminals/{id}/output   {"data":"..."}
POST /api/v1/agentd/terminals/{id}/state    {"status":"running"}
```

## PTY

`internal/pty`：

- Windows 使用 ConPTY：`CreatePseudoConsole`、进程线程属性、`ResizePseudoConsole`
- 非 Windows 使用 `sh -c` 加管道作为兼容实现
- 支持读写、Resize、退出码

AgentDaemon 每 2 秒领取 pending 终端，启动后：

- 读取 PTY 输出并上报
- 每 300ms 拉取输入队列写入 PTY
- 进程退出后上报 `exited + exitCode`

## 前端

Session 详情页在 `mode=terminal` 时渲染终端面板：

- 自动创建并绑定终端
- 350ms 轮询输出增量
- 点击终端后直接输入
- 支持 Enter / Backspace / Tab / 方向键 / Ctrl+C / 粘贴
- 支持重启和关闭

组件：`web/src/components/terminal-panel.tsx`

## 限制

- 控制面使用轮询而非 WebSocket，低延迟但非零
- 非 Windows daemon 当前是管道兼容模式，不是内核 PTY