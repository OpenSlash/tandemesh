# 移动端 React Native + RNOH 实施方案

> 状态：设计文档，待评审后进入实现。
> 前置调研：[30-mobile-tech-stack.md](./30-mobile-tech-stack.md)
> 目标：一套 React Native 代码，构建 iOS、Android、HarmonyOS 三端应用，并适配平板与折叠屏。

## 1. 决策

### 1.1 技术选型

| 项 | 选择 |
| --- | --- |
| 跨端框架 | React Native + RNOH |
| 语言 | TypeScript |
| RN 版本 | 0.82.x |
| RNOH 版本 | `@react-native-oh/react-native-harmony@0.82-stable` |
| 状态管理 | TanStack Query v5 |
| 导航 | React Navigation 7 |
| 列表 | FlashList |
| 安全存储 | iOS Keychain / Android Keystore / HarmonyOS 安全存储 |
| 构建 | pnpm workspace + 三端 CI 矩阵 |

### 1.2 为什么锁定 0.82-stable

RNOH 官方 README 明确说明：

- `main` 是版本演进分支，**不保证稳定性，生产环境不要直接依赖**
- 生产接入应使用 npm dist-tag `0.82-stable`
- 当前官方推荐新项目使用 `0.82-stable`（最新 GA）

虽然仓库已有 0.84、0.86 分支和 tag，但首版不追新。0.84/0.86 只做 PoC，等 0.82 稳定交付后再评估升级。

参考：
- [RNOH 当前主仓](https://gitcode.com/CPF-RN/ohos_react_native)
- [RNOH README（版本策略）](https://gitcode.com/CPF-RN/ohos_react_native/blob/main/README.md)
- [RNOH 稳定版本识别与安装](https://gitcode.com/CPF-RN/ohos_react_native/blob/main/docs/zh-cn/05-%E8%BF%90%E7%BB%B4/%E7%A8%B3%E5%AE%9A%E7%89%88%E6%9C%AC%E8%AF%86%E5%88%AB%E4%B8%8E%E5%AE%89%E8%A3%85.md)

### 1.3 非目标

首版不做：

- 完整离线编辑
- 移动端终端写操作
- 移动端插件市场
- HarmonyOS 以外的国产系统适配
- 一套安装包跨三端
- 绕过应用商店的 OTA 热更新

## 2. 总体架构

```text
┌──────────────────────────────────────────────┐
│ apps/mobile                                  │
│  ├─ app/            导航、Providers、主题      │
│  ├─ features/       tasks/runs/diff/approvals │
│  ├─ components/     RN UI 组件                │
│  ├─ platform/       iOS/Android/Harmony 桥接  │
│  └─ native/         ios/android/harmony 工程  │
└──────────────────────┬───────────────────────┘
                       │ 只依赖纯 TS 接口
┌──────────────────────▼───────────────────────┐
│ packages/core                                │
│  ├─ api/            OrcaClient、DTO、错误码    │
│  ├─ domain/         任务/运行/审批规则         │
│  ├─ queries/        query keys、hooks 契约     │
│  └─ utils/          时间、状态、格式化         │
└──────────────────────┬───────────────────────┘
                       │
┌──────────────────────▼───────────────────────┐
│ packages/tokens                              │
│  颜色、间距、字号、断点、圆角、阴影            │
└──────────────────────────────────────────────┘
```

### 2.1 目录

```text
orca/
  apps/
    web/
    mobile/
      app/
      features/
      components/
      platform/
      ios/
      android/
      harmony/
  packages/
    core/
    tokens/
  pnpm-workspace.yaml
  package.json
```

### 2.2 分层规则

- `packages/core` 不依赖 DOM、React、React Native
- `packages/tokens` 只导出常量与类型
- `apps/mobile` 可以依赖 core/tokens
- `apps/web` 后续也改为依赖 core/tokens
- 平台能力只能通过 `platform/` 接口进入业务层
- 禁止业务组件直接 import RNOH 或 Harmony 原生模块

## 3. 共享核心

### 3.1 先从 Web 抽什么

优先级从高到低：

1. DTO 类型：`Task`、`Run`、`Worktree`、`Diff`、`Notification`
2. API 错误结构：`{ code, message }`
3. API 路径与请求参数
4. 状态映射：任务状态、运行状态、审批状态
5. 时间格式化与相对时间
6. 权限判断：viewer 只读、owner/admin 可管理
7. 查询 key 规范

### 3.2 不要共享什么

- Web 的 Tailwind 组件
- Radix UI
- 路由实现
- 浏览器存储
- 任何依赖 `window`、`document`、`localStorage` 的代码

### 3.3 OrcaClient

```ts
export interface OrcaClientOptions {
  baseUrl: string
  getToken: () => Promise<string | null>
  getWorkspace: () => string | null
  fetch?: typeof fetch
}

export class OrcaClient {
  get<T>(path: string, query?: Record<string, string | number | boolean>): Promise<T>
  post<T>(path: string, body?: unknown): Promise<T>
  patch<T>(path: string, body?: unknown): Promise<T>
  delete<T>(path: string): Promise<T>
}
```

职责：

- 拼接 `baseUrl` 与 `/api/v1`
- 注入 `Authorization: Bearer <token>`
- 注入 `X-Orca-Workspace`
- 统一解析 `E_*` 错误
- 401 时触发一次刷新或登出
- 超时与重试只对幂等 GET 生效

### 3.4 查询 key

```ts
export const qk = {
  me: () => ['me'] as const,
  workspaces: () => ['workspaces'] as const,
  tasks: (workspace: string, filter: string) => ['tasks', workspace, filter] as const,
  task: (id: string) => ['task', id] as const,
  taskActivity: (id: string) => ['task', id, 'activity'] as const,
  taskComments: (id: string) => ['task', id, 'comments'] as const,
  taskRuns: (id: string) => ['task', id, 'runs'] as const,
  run: (id: string) => ['run', id] as const,
  runDiff: (id: string) => ['run', id, 'diff'] as const,
  notifications: () => ['notifications'] as const,
}
```

## 4. 认证与安全

### 4.1 现状问题

Orca 当前 Web 使用 `orca_session` Cookie。移动端不能可靠复用浏览器 Cookie：

- WebView 登录体验差，且 Cookie 难同步到原生请求
- 系统浏览器登录后无法直接读取 HttpOnly Cookie
- 把 Cookie 塞进 RN 网络层会带来安全和兼容性问题

因此移动端必须新增 **Bearer Token** 认证。

### 4.2 后端新增

表：

```sql
mobile_auth_codes (
  code_hash TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  workspace_id TEXT,
  expires_at_ms INTEGER NOT NULL,
  used_at_ms INTEGER,
  created_at_ms INTEGER NOT NULL
)

mobile_tokens (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL,
  token_hash TEXT NOT NULL,
  device_name TEXT,
  platform TEXT,
  created_at_ms INTEGER NOT NULL,
  last_used_ms INTEGER,
  revoked_at_ms INTEGER
)
```

接口：

```http
GET  /auth/mobile/login?redirect=orca://auth
GET  /auth/mobile/callback?code=...&state=...
POST /auth/mobile/token
POST /auth/mobile/logout
GET  /auth/me
```

流程：

1. App 打开系统浏览器：`/auth/mobile/login?redirect=orca://auth`
2. 服务端复用现有 OIDC 登录
3. 登录成功后生成一次性 code，重定向 `orca://auth?code=...&state=...`
4. App 校验 state，调用 `POST /auth/mobile/token` 换取 Bearer Token
5. Token 存入 Keychain/Keystore/安全存储
6. 后续请求带 `Authorization: Bearer <token>`
7. 退出登录调用 `/auth/mobile/logout` 并清除本地 Token

约束：

- 一次性 code TTL 60 秒
- Token 只存哈希，不存明文
- Token 默认 30 天，滑动续期
- 支持按设备撤销
- 深链参数禁止打印到日志
- Token 不写入 AsyncStorage

### 4.3 开发模式

本地开发保留 `X-Orca-User` 或静态 dev token，但必须由环境变量显式开启，生产禁用。

## 5. 页面与信息架构

### 5.1 一级导航

Compact：

```text
底部 Tab：收件箱 | 任务 | 审批 | 通知 | 我的
```

Medium/Expanded：

```text
左侧 NavigationRail：收件箱 | 任务 | 审批 | 通知 | 我的
右侧内容区
```

### 5.2 页面清单

| 页面 | 优先级 | 说明 |
| --- | --- | --- |
| 登录 | P0 | 系统浏览器 + 深链回调 |
| 工作区选择 | P0 | 多工作区切换 |
| 收件箱 | P0 | 待处理、提及、分配给我 |
| 任务列表 | P0 | 状态/负责人/迭代筛选 |
| 任务详情 | P0 | 元数据、描述、活动、运行 |
| 运行详情 | P0 | 状态、日志摘要、Diff 入口 |
| Diff 查看 | P0 | 文件树 + 行级 diff |
| 审批 | P0 | 待审批列表、通过/拒绝 |
| 通知 | P1 | 列表、已读 |
| 评论 | P1 | 查看 + 发送 |
| 文档 | P2 | 只读 |
| 终端 | P2 | 只读轮询，后续再开放输入 |
| 设置 | P1 | 账号、工作区、退出、版本 |

### 5.3 任务详情布局

Compact：

```text
标题
状态 / 负责人 / 迭代
Tab: 概览 | 活动 | 运行 | Diff
```

Expanded：

```text
左：任务列表（可折叠）
中：任务详情
右：运行/Diff 辅助面板
```

Medium 使用两栏：任务详情 + 运行列表。

## 6. 自适应布局

### 6.1 断点

| 名称 | 宽度 | 行为 |
| --- | --- | --- |
| compact | < 600 | 单列、底部 Tab |
| medium | 600–839 | 单列 + NavigationRail |
| expanded | >= 840 | NavigationRail + 列表/详情双栏 |

```ts
export type LayoutSize = 'compact' | 'medium' | 'expanded'

export function layoutFor(width: number): LayoutSize {
  if (width >= 840) return 'expanded'
  if (width >= 600) return 'medium'
  return 'compact'
}
```

### 6.2 平板

- 表单最大宽度 640，避免输入框被拉长
- Diff 在 expanded 下显示文件树，compact 下用底部抽屉
- 列表与详情使用双栏，不把移动端单列简单放大
- 支持横竖屏切换，不重建导航栈
- 支持键盘快捷键（Phase 3）：`j/k` 切换任务、`Esc` 返回

### 6.3 折叠屏

统一抽象：

```ts
export type FoldPosture = 'flat' | 'half-opened' | 'closed' | 'unknown'

export interface FoldInfo {
  posture: FoldPosture
  separating: boolean
  foldWidth: number
}
```

平台实现：

- Android：Jetpack WindowManager `WindowInfoTracker` + `FoldingFeature`
- HarmonyOS：窗口/显示 API，监听窗口尺寸和折叠状态
- iOS：`UIScene` size classes；iPadOS Stage Manager 多窗口

布局规则：

- `half-opened` 且 `separating` 时，双栏内容不跨折痕
- 折叠态自动退化为单栏
- 监听尺寸变化，不缓存启动时宽度
- 折叠/展开后保持当前任务与滚动位置

## 7. 数据与状态

### 7.1 查询策略

| 数据 | 策略 |
| --- | --- |
| 当前用户 | 启动时一次，失败登出 |
| 工作区 | 启动时一次，缓存 5 分钟 |
| 任务列表 | 30 秒 stale，前台每 10 秒轮询 |
| 任务详情 | 10 秒 stale，前台每 5 秒轮询 |
| 运行详情 | 5 秒 stale，运行中每 3 秒轮询 |
| Diff | 运行结束后按需加载，缓存 5 分钟 |
| 通知 | 前台每 30 秒轮询 |
| 审批 | 前台每 10 秒轮询 |

### 7.2 前后台

- `AppState` 进入后台：停止轮询，记录时间
- 回到前台：立即刷新当前页，恢复轮询
- 超过 5 分钟：清空运行中数据的 stale 状态
- 弱网：指数退避，最大 30 秒

### 7.3 离线

MVP 只做只读缓存：

- TanStack Query persist 到 AsyncStorage
- 只缓存任务列表、任务详情、通知
- 不缓存 Diff 和终端
- 离线时显示 banner，禁止写操作
- 重新联网后按当前页面刷新

### 7.4 乐观更新

MVP 不做乐观更新。评论、审批先采用：

1. 提交
2. 按钮 loading
3. 成功刷新
4. 失败 toast + 保留输入

Phase 3 再对评论做乐观插入。

## 8. Diff 与终端

### 8.1 Diff

数据源：`GET /api/v1/runs/{id}/diff`

渲染：

- 文件列表 + 文件内行列表
- 使用 FlashList 虚拟化
- 行号、增删标记、旧/新行背景色
- 等宽字体，横向滚动按行独立
- 大文件折叠，默认只渲染前 200 行
- 1000 行以上默认关闭语法高亮

不做：

- WebView 渲染 diff
- 移动端编辑代码
- 行内评论（Phase 3 再做）

### 8.2 终端

Phase 2 只做只读：

- 复用 `terminalOutput` 轮询
- 等宽文本、自动滚底、暂停滚动
- 不发送输入

Phase 3 再评估输入：

- 软键盘快捷键条
- Ctrl/Cmd 组合键
- 粘贴确认

## 9. 平台桥接

### 9.1 接口

```ts
export interface Platform {
  secureStorage: {
    get(key: string): Promise<string | null>
    set(key: string, value: string): Promise<void>
    delete(key: string): Promise<void>
  }
  auth: {
    openLogin(url: string): Promise<{ code: string; state: string }>
  }
  share: {
    text(value: string): Promise<void>
  }
  fold: {
    current(): Promise<FoldInfo>
    subscribe(listener: (info: FoldInfo) => void): () => void
  }
  lifecycle: {
    onForeground(listener: () => void): () => void
  }
}
```

### 9.2 HarmonyOS

- 使用 RNOH 0.82-stable
- 原生模块用 ArkTS + TurboModule
- 安全存储使用 HarmonyOS 系统能力
- 深链在 `module.json5` 注册 `orca://`
- 折叠状态通过窗口 API 读取
- 所有 Harmony 特有代码放在 `platform/harmony`

### 9.3 Android

- `react-native-keychain` 使用 Keystore
- `WindowInfoTracker` 封装折叠状态
- 深链 `intent-filter`
- 返回键处理交给 React Navigation

### 9.4 iOS

- Keychain 存储 Token
- Universal Links + URL Scheme 双通道
- iPadOS 使用 size classes
- 安全区域使用 `react-native-safe-area-context`

## 10. 构建与发布

### 10.1 版本矩阵

| 组件 | 版本 |
| --- | --- |
| React Native | 0.82.x |
| RNOH | 0.82-stable |
| react-native-harmony-cli | 与 RNOH 0.82 匹配 |
| Hermes | RN 0.82 默认 |
| DevEco Studio | 按 RNOH 0.82 环境文档 |
| HarmonyOS SDK | 按 RNOH 0.82 环境文档 |
| Xcode | 当前 App Store 要求 |
| Android Gradle Plugin | RN 0.82 模板 |

版本锁定文件：

- `package.json` 精确版本，不用 `^`
- `pnpm-lock.yaml` 提交
- `harmony/oh-package.json5` 锁定 RNOH
- CI 缓存按 lockfile hash

### 10.2 CI 矩阵

| Job | Runner | 产物 |
| --- | --- | --- |
| lint + typecheck + unit | Linux | 无 |
| iOS | macOS | `.ipa` |
| Android | Linux/macOS | `.apk` / `.aab` |
| HarmonyOS | Windows/macOS + DevEco | `.app` |

步骤：

```bash
pnpm install --frozen-lockfile
pnpm -r typecheck
pnpm -r test
pnpm --filter @orca/mobile build:ios
pnpm --filter @orca/mobile build:android
pnpm --filter @orca/mobile build:harmony
```

### 10.3 环境

| 环境 | API | 用途 |
| --- | --- | --- |
| dev | `http://10.0.2.2:8080` / 局域网 | 本地 |
| staging | `https://staging.orca.example` | 内测 |
| prod | `https://orca.example` | 正式 |

构建时注入：

- `ORCA_API_URL`
- `ORCA_OIDC_CLIENT_ID`
- `ORCA_DEEP_LINK_SCHEME`

### 10.4 发布

- iOS：TestFlight → App Store
- Android：内测轨道 → 正式轨道
- HarmonyOS：AppGallery Connect 内测 → 正式
- 三端版本号统一，构建号独立递增
- 首版不做 OTA 热更新

## 11. 测试

### 11.1 单元测试

- `packages/core`：API 参数、错误映射、状态转换
- `packages/tokens`：断点函数
- 业务 hooks：mock client

### 11.2 组件测试

- React Native Testing Library
- 关键组件：TaskCard、StatusBadge、DiffLine、ApprovalCard
- 覆盖空态、加载、错误、只读

### 11.3 E2E

- iOS/Android：Detox
- HarmonyOS：首版手工 + 关键路径脚本
- 路径：
  1. 登录
  2. 切换工作区
  3. 打开任务
  4. 查看运行
  5. 查看 Diff
  6. 审批
  7. 退出

### 11.4 设备矩阵

| 类型 | 设备 |
| --- | --- |
| 手机 | iPhone SE / iPhone 16 / Pixel 小屏 / Pixel 大屏 |
| 平板 | iPad 10.9 / iPad Pro 13 / Android 平板 / HarmonyOS 平板 |
| 折叠 | 华为折叠屏 / 三星 Fold / 小米 Fold |
| 系统 | iOS 当前-1 / Android 当前-2 / HarmonyOS NEXT |

### 11.5 性能预算

| 指标 | 目标 |
| --- | --- |
| 冷启动到任务列表 | < 2.5s（中端机） |
| 任务列表滚动 | 55–60fps |
| 1000 行 Diff 首屏 | < 500ms |
| 页面切换 | < 300ms |
| 内存峰值 | < 250MB |
| 安装包 | Android < 40MB，iOS < 60MB（首版目标） |

### 11.6 视觉回归

- 320 / 390 / 600 / 840 / 1200 宽度截图
- 深色/浅色
- 字体放大 1.3x
- 折叠/展开两态

## 12. 可观测性

- 崩溃：iOS/Android 接 Sentry RN；HarmonyOS 先用 DevEco 崩溃日志，确认 Sentry 支持后再接
- 性能：React DevTools Profiler + DevEco Profiler
- 日志：分级，生产只保留 warn/error
- 埋点：页面曝光、登录成功、任务打开、Diff 打开、审批提交
- 不上报 Token、评论正文、Diff 内容

## 13. 实施阶段

### 阶段 0：Web 稳定 + 后端认证（1–2 人周）

- 完成 Web 功能测试与打磨
- 抽 `packages/core` / `packages/tokens`
- 新增移动端 Bearer Token 认证
- 新增设备 token 撤销接口

退出标准：

- Web 关键流程稳定
- 移动端可用 curl 完成登录换 token
- Token 可撤销

### 阶段 1：RN 壳（2 人周）

- 初始化 `apps/mobile`
- iOS/Android/Harmony 三端空壳可启动
- 导航、主题、Token 存储
- CI 三端构建跑通

退出标准：

- 三端安装包可安装
- 登录换 token 成功
- 工作区列表可拉取

### 阶段 2：只读 MVP（4–6 人周）

- 收件箱
- 任务列表/详情
- 运行详情
- Diff 查看
- 审批列表
- 通知
- 平板/折叠断点

退出标准：

- 三端核心只读流程可用
- 三档断点无布局错乱
- P0 崩溃为 0

### 阶段 3：写操作（3–4 人周）

- 评论
- 审批通过/拒绝
- 任务状态变更
- 批注发送
- 乐观更新与失败回滚

退出标准：

- viewer 角色只读
- 写操作有 loading/错误/重试
- 冲突有明确提示

### 阶段 4：Harmony 打磨与发布（3–4 人周）

- Harmony 深链、安全存储、折叠状态
- 推送评估
- 性能优化
- 三端内测
- 商店素材与隐私合规

退出标准：

- 三端内测通过
- 崩溃率 < 1%
- 关键路径性能达标

总估算：**13–18 人周**，2 人并行约 **7–9 周**，不含商店审核等待。

## 14. 风险

| 风险 | 影响 | 缓解 |
| --- | --- | --- |
| RNOH 0.82 与三方库不兼容 | 功能缺失 | 优先纯 TS 库；建立兼容清单；PoC 验证 |
| 移动端认证改造 | 阻塞登录 | 阶段 0 先做，接口先行 |
| Harmony 原生能力不足 | 推送/深链/折叠不可用 | 薄桥接层；按需实现；不阻塞 MVP |
| Diff 大文件卡顿 | 体验差 | 虚拟化、折叠、关闭高亮 |
| 折叠屏 API 差异 | 布局错乱 | 统一 FoldInfo 抽象；真机矩阵 |
| CI 缺 macOS/Harmony 环境 | 无法出包 | 提前准备 runner；本地兜底 |
| 三端 UI 漂移 | 维护成本 | 共享 token；平台组件分叉；截图回归 |
| Token 泄露 | 安全问题 | Keychain/Keystore；不落日志；可撤销 |

## 15. 验收标准

功能：

- 登录、切换工作区、任务列表、任务详情、运行详情、Diff、审批、通知可用
- viewer 只读
- 评论发送成功并可刷新

适配：

- 320、390、600、840、1200 宽度无溢出
- 折叠/展开布局正确
- 横竖屏切换不丢状态
- 深色模式可用

质量：

- `typecheck`、unit、E2E 通过
- 三端构建成功
- P0/P1 崩溃为 0
- 冷启动、列表、Diff 达到性能预算

安全：

- Token 存安全存储
- 退出登录后 Token 失效
- 日志无敏感信息

## 16. 待确认

1. 是否接受阶段 0 先改后端认证？
2. HarmonyOS 最低支持版本？
3. 首版是否需要评论/审批写操作？
4. 是否需要企业内部分发（不上架）？
5. 推送是否进入首版？

## 参考资料

- [RNOH 当前主仓](https://gitcode.com/CPF-RN/ohos_react_native)
- [RNOH README](https://gitcode.com/CPF-RN/ohos_react_native/blob/main/README.md)
- [RNOH 稳定版本识别与安装](https://gitcode.com/CPF-RN/ohos_react_native/blob/main/docs/zh-cn/05-%E8%BF%90%E7%BB%B4/%E7%A8%B3%E5%AE%9A%E7%89%88%E6%9C%AC%E8%AF%86%E5%88%AB%E4%B8%8E%E5%AE%89%E8%A3%85.md)
- [RNOH 环境搭建](https://gitcode.com/CPF-RN/ohos_react_native/blob/main/docs/zh-cn/02-%E5%BC%80%E5%8F%91/01-%E7%8E%AF%E5%A2%83%E4%B8%8E%E5%B7%A5%E5%85%B7/%E7%8E%AF%E5%A2%83%E6%90%AD%E5%BB%BA.md)
- [RNOH 编译构建指南](https://gitcode.com/CPF-RN/ohos_react_native/blob/main/docs/zh-cn/02-%E5%BC%80%E5%8F%91/02-%E5%BC%80%E5%8F%91%E6%8C%87%E5%8D%97/%E7%BC%96%E8%AF%91%E6%9E%84%E5%BB%BA/%E7%BC%96%E8%AF%91%E6%9E%84%E5%BB%BA%E6%8C%87%E5%8D%97.md)
- [React Native 官方文档](https://reactnative.dev/docs/getting-started)
- [React Navigation](https://reactnavigation.org/docs/getting-started)
- [TanStack Query](https://tanstack.com/query/latest/docs/framework/react/overview)
