# Skills Registry

Skill 是工作区级的 SKILL.md 剧本，被 Agent 引用后注入到提示词。Registry 负责安装、版本、来源、启停和导出。

## 数据

`skills` 表新增：

| 字段 | 说明 |
| --- | --- |
| `version` | 版本号 |
| `source` | `manual / url / file / repo` |
| `source_url` | 安装来源 |
| `tags_json` | 标签 |
| `enabled` | 是否注入提示词 |
| `checksum` | 安装内容的 sha256 |

`body` 只保存正文；frontmatter 中的 `name`、`description`、`version`、`tags` 会拆到独立字段。

## 安装

```http
POST /api/v1/skills/install
{
  "workspaceId": "ws_default",
  "sourceUrl": "https://example.com/review/SKILL.md"
}
```

或直接提交正文：

```json
{
  "workspaceId": "ws_default",
  "body": "---\nname: review-style\nversion: 1.2.0\n---\n\n..."
}
```

行为：

- URL 仅允许 `http(s)`，超时 15s，最大 1MB
- 自动解析 frontmatter，缺失 `name` 时要求请求体提供
- 计算 sha256 作为 `checksum`
- 同名技能按 workspace + name 覆盖更新

## 其他接口

```
GET  /api/v1/skills?workspace=ws_default
GET  /api/v1/skills/{id}
PUT  /api/v1/skills/{id}
DELETE /api/v1/skills/{id}
GET  /api/v1/skills/{id}/raw
POST /api/v1/skills/{id}/enabled   {"enabled": false}
GET  /api/v1/skills/export?workspace=ws_default
```

`/raw` 返回完整 SKILL.md，`/export` 返回每个技能的 markdown，用于备份或同步到仓库。

## Agent 注入

Agent 的 `skills` 列表引用技能名。控制面组装任务提示词时：

1. 按名称读取技能
2. 跳过 `enabled=false`
3. 用 `skills.Prompt` 截断到 32KB
4. 拼接到任务 prompt 前

## CLI

```bash
orca skills list --workspace=ws_default
orca skills get --name=review-style
orca skills install --file=./SKILL.md
orca skills install --url=https://example.com/SKILL.md
orca skills export --dir=./skills
```

`export` 生成 `skills/<name>/SKILL.md`，可直接提交到仓库供 AgentDaemon 的本地 skills 扫描使用。

## 前端

Skills 页面支持：

- 手工创建（名称、描述、版本、标签、正文）
- 从 URL 安装
- 展示 source / version / tags / enabled
- 启用、禁用、删除