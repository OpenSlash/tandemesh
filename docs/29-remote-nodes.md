# 远程 Agent 节点增强

本模块增强多执行机调度：节点标签选择、负载感知和健康退避。

## 节点标签

执行机注册时的 `labels`（AgentDaemon `--labels`）是调度标签：

```bash
tandemesh-agentd --labels=os=linux,gpu,zone=cn
```

Agent 的 `capabilities` 中带 `node:` 前缀的项被解释为**必需节点标签**：

```json
{ "capabilities": ["review", "node:gpu", "node:zone=cn"] }
```

调度器只把 Run 放到标签完全覆盖 `node:*` 选择器的节点上；普通 capabilities 不影响调度。

## 负载感知

`pickMachine` 不再简单取第一个可用节点，而是：

1. 过滤离线、心跳过期、容量已满的节点
2. 过滤不支持该 Agent backend 的节点
3. 过滤缺少必需标签的节点
4. 在剩余节点中选择 `busy / concurrency` 最低的
5. 负载相同则选择当前运行数更少的

这样可以把新任务分散到空闲节点。

## 健康退避

Scheduler 每 500ms 检查：

- `status = online`
- `last_seen` 超过 45 秒

命中后：

1. 节点标记为 `offline`
2. 该节点上的 `leased / preparing / running / waiting_approval` Run 重新入队
3. 其他健康节点可以所有这些 Run

节点恢复心跳后自动回到 `online`。

## 手动状态

```http
POST /api/v1/machines/{id}/status
{"status": "draining"}
```

支持：

- `online`
- `offline`
- `draining`

`draining` 不接收新 Run，但已运行任务继续执行。Machines 页面提供 Drain / Resume 按钮，并展示节点标签。

## 相关字段

- `Machine.Labels` 与 `Agent.Capabilities` 中的 `node:*` 选择器共同决定标签匹配
- `run.MachineID` 记录实际节点
- `machines.status` 不再只由心跳改，也可手动操作

## 注意

- 标签是精确字符串匹配，大小写敏感
- 心跳超时固定 45 秒，可用 `--heartbeat` 调整 AgentDaemon 频率
- 被标记 offline 的节点重新上线后不会自动恢复之前被重新入队的 Run，它们已经由其他节点处理