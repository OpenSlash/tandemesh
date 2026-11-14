# 移动端技术栈调研

> 状态：调研结论，暂不进入实现。
> 目标：一套代码同时覆盖 iOS、Android、HarmonyOS，并适配平板与折叠屏。

## 1. 先明确「一次构建」

移动端不存在真正意义上的「一个二进制跑三端」：

- iOS 需要 Xcode 签名，产出 `.ipa`
- Android 需要 Gradle 签名，产出 `.apk` / `.aab`
- HarmonyOS 需要 DevEco / hvigor 签名，产出 `.hap` / `.app`

因此本文的「一次构建」定义为：

> **一套业务代码 + 一套设计系统 + 一条 CI 构建矩阵**，分别产出三端安装包。

这仍然是当前工程上最接近目标的做法，但不能承诺同一个安装包跨系统运行。

## 2. 候选框架

### 2.1 Flutter + Flutter-OH

- iOS / Android：官方稳定支持
- HarmonyOS：通过 OpenHarmony SIG 的 Flutter-OH 适配，已发布 3.41 版本
- 语言：Dart
- UI：自绘引擎，三端视觉一致性最好

参考：
- [Flutter 3.41 OpenHarmony Release](https://developer.huawei.com.cn/consumer/cn/forum/topic/0201223234884554700?fid=0109140870620153026)
- [Flutter-OH 3.41 发布：内存负载优化](https://developer.huawei.com/consumer/cn/forum/topic/0204225021093038138)

优点：

- 一套 UI 代码三端一致
- 性能稳定，动画和复杂列表表现好
- 平板/折叠屏可通过尺寸断点统一适配
- 社区大，组件生态成熟

风险：

- Orca 现有 Web 是 React/TypeScript，Flutter 无法复用页面、组件和状态逻辑
- Flutter-OH 的插件生态与官方 Flutter 不完全同步，原生能力需要自行桥接
- 团队需要额外维护 Dart 技术栈

### 2.2 React Native + RNOH

- iOS / Android：React Native 官方支持
- HarmonyOS：CPF-RN 社区的 React Native for OpenHarmony（RNOH）
- 语言：TypeScript / React
- UI：原生组件渲染

参考：
- [React Native for OpenHarmony 当前主仓（CPF-RN）](https://gitcode.com/CPF-RN/ohos_react_native)
- [RNOH 文档目录](https://gitcode.com/CPF-RN/ohos_react_native/tree/main/docs/zh-cn)
- [RNOH Tags](https://gitcode.com/CPF-RN/ohos_react_native/tags)

> 注意：旧地址 `gitee.com/openharmony-sig/ohos_react_native` 已停止更新（最后提交 2025-02），当前活跃主仓是 `CPF-RN/ohos_react_native`，`gitcode.com` 会跳转到 AtomGit。

优点：

- 与 Orca Web 同为 React + TypeScript，API 客户端、类型、状态逻辑、设计 token 可大量复用
- 一套代码覆盖 iOS / Android / HarmonyOS
- 原生组件渲染，平台交互自然
- 现有 React 团队迁移成本最低

风险：

- RNOH 提供 0.72 / 0.77 / 0.82 / 0.84 / 0.86 多条分支；必须锁定 RN 版本，三方库仍需逐个验证
- HarmonyOS 上的原生模块、推送、文件、深链等能力需要 RNOH 生态或自研桥接
- 折叠屏/多窗口能力需要自己封装原生模块

### 2.3 Kuikly（腾讯，Kotlin Multiplatform）

- 平台：Android、iOS、HarmonyOS、H5、微信小程序、macOS
- 语言：Kotlin / Compose DSL
- UI：复用平台原生组件，逻辑与 UI 跨端

参考：
- [Kuikly 架构介绍](https://kuikly.tds.qq.com/Introduction/arch.html)
- [Kuikly Roadmap 2025](https://kuikly.tds.qq.com/Blog/roadmap2025.html)
- [Kuikly Roadmap 2026](https://raw.githubusercontent.com/tencent-tds/kuiklyui/eddfda292aca82b879158859046aa85808e0eca0/docs/Blog/Kuikly%20Roadmap%EF%BC%882026%EF%BC%89.md)

优点：

- 原生渲染，性能和包体积控制较好
- 鸿蒙支持已开源，且有 Kotlin/Native 鸿蒙工具链
- 适合中国区多端产品，社区组件市场在增长
- 对折叠屏、平板有专门适配实践

风险：

- 团队需要 Kotlin / KMP 能力，和现有 TypeScript 技术栈不连续
- 生态比 Flutter / RN 年轻，三方库和排障资料较少
- 鸿蒙调试脚手架仍在完善

### 2.4 uni-app x

- 平台：Web、Android、iOS、HarmonyOS Next、小程序
- 语言：Vue / UTS
- HarmonyOS：4.61+ 支持纯血鸿蒙，编译为 ArkTS 运行

参考：
- [uni-app x HarmonyOS 开发指南](https://doc.dcloud.net.cn/uni-app-x/app-harmony/)
- [uni-app x 介绍](https://doc.dcloud.net.cn/uni-app-x/)

优点：

- Vue 语法，Web 团队上手快
- 一套代码覆盖 App / H5 / 小程序 / 鸿蒙
- 国内文档和中文生态完善
- 宽屏适配有官方方案

风险：

- 强依赖 DCloud 工具链（HBuilderX）
- 原生插件扩展和复杂交互能力不如 RN/Flutter 灵活
- 大型工程的可维护性、类型安全不如纯 TS/Kotlin 方案

### 2.5 Tauri 2

- iOS / Android：Tauri 2 官方支持
- HarmonyOS：社区实验性方案，未形成稳定生产路径

参考：
- [Tauri HarmonyOS 社区讨论](https://bbs.itying.com/topic/69e26ad5c504c50058fd6dde#69e26af8dc652800528fdf9f6)

结论：**不建议**作为三端生产方案。Tauri 的强项是桌面和 Web 技术复用，鸿蒙支持不成熟，移动端原生能力仍需大量自研。

## 3. 对比表

| 维度 | Flutter-OH | React Native + RNOH | Kuikly | uni-app x | Tauri 2 |
| --- | --- | --- | --- | --- | --- |
| iOS | 稳定 | 稳定 | 稳定 | 稳定 | 稳定 |
| Android | 稳定 | 稳定 | 稳定 | 稳定 | 稳定 |
| HarmonyOS | Flutter-OH 适配 | RNOH 适配 | 已开源 | 4.61+ 支持 | 实验性 |
| 语言 | Dart | TypeScript / React | Kotlin | Vue / UTS | Rust + Web |
| 与现有 Web 复用 | 低 | **高** | 低 | 中 | 高 |
| UI 一致性 | 最高 | 高（原生组件） | 高（原生组件） | 高 | 高 |
| 平板/折叠屏 | 断点自适配 | 需原生桥接 | 有实践 | 官方宽屏方案 | 需自研 |
| 生态成熟度 | 高 | 中高 | 中 | 中高（国内） | 低（鸿蒙） |
| 团队学习成本 | 中 | **低** | 高 | 中 | 中 |
| 推荐度 | 备选 | **首选** | 特定场景 | 备选 | 不推荐 |

## 4. 推荐结论

### 首选：React Native + RNOH

理由：

1. Orca 已有 React 19 + TypeScript + TanStack Query + 设计 token，迁移到 RN 的复用率最高。
2. 一套代码覆盖三端，符合「一次开发、多端构建」目标。
3. 团队无需新增 Dart/Kotlin 能力，交付风险最低。
4. RNOH 由 CPF-RN 社区持续维护，2026-11 仍在每日提交，并提供 0.72–0.86 多分支。

必须接受的约束：

- RNOH 已提供多 RN 版本分支，选库时要锁定对应版本，优先纯 JS/TS 库，减少原生依赖。
- 鸿蒙原生能力（推送、文件、深链、折叠状态）需要抽象成自有桥接层。
- 复杂长列表、终端渲染、Diff 大文本需要专门性能验证。

### 备选：Flutter + Flutter-OH

如果团队愿意投入 Dart，并更看重三端像素级一致和渲染性能，选 Flutter。它的鸿蒙适配和宽屏方案更成熟，但无法复用现有 Web 代码。

### 特定场景：Kuikly

如果未来需要同时覆盖鸿蒙、微信小程序，且团队有 Kotlin/KMP 能力，Kuikly 值得评估。

## 5. 推荐架构

```text
orca/
  apps/
    web/                 # 现有 React Web
    mobile/              # React Native + RNOH
  packages/
    core/                # 纯 TS：API client、类型、查询 key、领域模型
    tokens/              # 设计 token：颜色、间距、字号、断点
    ui-web/              # Web UI 组件
    ui-mobile/           # RN UI 组件
```

关键原则：

- `packages/core` 不依赖 DOM / React Native，只放纯 TypeScript
- Web 与 Mobile 共享 API 类型、请求封装、业务规则
- UI 不强行共享，按平台实现，避免抽象泄漏
- 移动端先做「只读优先」：任务列表、任务详情、运行状态、Diff 查看
- 写操作（评论、审批、发送批注、终端）第二阶段开放

## 6. 平板与折叠屏适配

### 6.1 断点

统一采用三档：

| 断点 | 宽度 | 布局 |
| --- | --- | --- |
| Compact | < 600dp | 底部 Tab，单列 |
| Medium | 600–839dp | 导航栏 + 单列内容 |
| Expanded | >= 840dp | 导航栏 + 列表/详情双栏 |

### 6.2 页面策略

- 任务页：Compact 列表→详情堆叠；Expanded 左侧任务列表、右侧详情
- Diff：Expanded 双栏（文件树 + diff），Compact 单栏切换
- 终端：Expanded 固定高度面板；Compact 全屏终端
- 表单：限制最大宽度，避免平板上输入框被拉得过长

### 6.3 折叠屏

需要抽象 `useFoldPosture()`：

```ts
type FoldPosture = 'flat' | 'half-opened' | 'closed' | 'unknown'
```

平台实现：

- Android：封装 Jetpack WindowManager 的 `WindowInfoTracker` + `FoldingFeature`
- HarmonyOS：使用窗口/显示相关 API，监听窗口尺寸与折叠状态
- iOS：iPadOS 使用 `UIScene` size classes 与 Stage Manager 多窗口

布局规则：

- `half-opened` 时避免内容跨过折痕
- 双栏布局在折叠态自动退化为单栏
- 监听窗口尺寸变化而不是只读启动尺寸

参考：
- [HarmonyOS 多窗口/宽屏适配](https://developer.huawei.com/consumer/en/doc/best-practices-V14/bpta-multi-window-V14)
- [HarmonyOS 折叠屏布局自适应问答](https://developer.huawei.com/consumer/cn/forum/topic/0204224332693110462?fid=0109140870620153026)
- [Kuikly 折叠屏适配实践](https://kuikly.tds.qq.com/home/blog/386)

## 7. 构建与发布

### 7.1 一套代码

- `apps/mobile` 单一代码库
- 环境配置通过 build profile 注入：`dev / staging / prod`
- API 地址、OAuth client、推送配置按环境区分

### 7.2 CI 矩阵

| 平台 | 工具 | 产物 |
| --- | --- | --- |
| iOS | Xcode + Fastlane | `.ipa` |
| Android | Gradle | `.apk` / `.aab` |
| HarmonyOS | DevEco + hvigor | `.hap` / `.app` |

流水线：

1. `pnpm install`
2. `pnpm test`
3. `pnpm --filter mobile typecheck`
4. 三端并行构建
5. 上传制品与符号表
6. 内测分发

### 7.3 版本策略

- 三端共用语义化版本号
- 原生壳版本与 JS bundle 版本分开记录
- 鸿蒙与 RNOH 版本锁定，升级需回归

## 8. 测试与验收

### 8.1 功能测试

- 登录 / 工作区切换
- 任务列表与筛选
- 任务详情与运行状态
- Diff 查看与批注
- 审批与通知
- 终端只读或交互（视阶段）

### 8.2 设备矩阵

- 手机：iOS 小屏/大屏、Android 小屏/大屏
- 平板：iPad、Android 平板、HarmonyOS 平板
- 折叠屏：展开/半折/合上
- 横竖屏切换
- 深色模式、字体放大

### 8.3 自动化

- `packages/core` 单测
- RN 组件快照测试
- E2E：Detox（iOS/Android），鸿蒙侧优先手工 + 关键路径脚本
- 截图回归覆盖三档断点

## 9. 风险与缓解

| 风险 | 影响 | 缓解 |
| --- | --- | --- |
| RNOH 版本落后 | 三方库不兼容 | 优先纯 TS 库；建立兼容清单 |
| 鸿蒙原生能力缺失 | 推送/文件/深链不可用 | 自研薄桥接层，按需实现 |
| 折叠屏 API 差异 | 布局错乱 | 统一 `useFoldPosture` + 断点 |
| 大 Diff 性能 | 卡顿 | 虚拟列表、分片渲染 |
| 终端交互复杂 | 移动端体验差 | 第一阶段只读，第二阶段再开放 |
| 三端构建环境 | CI 维护成本 | 容器化 + 缓存 + 版本锁定 |

## 10. 实施阶段

### 阶段 0：Web 功能测试与打磨（前置）

先把 Web 端任务、Diff、审批、通知、终端流程测稳，因为移动端要复用同一套 API 和业务规则。

### 阶段 1：共享核心

- 抽出 `packages/core` 与 `packages/tokens`
- Web 改用共享包
- 建立 API 类型契约测试

### 阶段 2：移动端只读 MVP

- 登录 / 工作区
- 任务列表 / 详情
- 运行状态 / Diff 查看
- 平板与折叠屏断点

### 阶段 3：写操作

- 评论、审批、批注发送
- 通知
- 终端（先只读后交互）

### 阶段 4：三端发布

- iOS / Android / HarmonyOS 签名与内测
- 崩溃监控、性能监控
- 灰度与回滚

## 11. 最终建议

1. **移动端选 React Native + RNOH**，最大化复用现有 React/TS 资产。
2. **不承诺一个安装包跑三端**，承诺一套代码 + 三端 CI 产物。
3. **先完成 Web 功能测试与打磨**，再抽共享核心，最后进入移动端实现。
4. **平板/折叠屏从第一天就按断点设计**，不要等页面写完再补。
5. **鸿蒙能力按薄桥接层设计**，避免业务代码直接依赖 RNOH 细节。

## 参考资料

> 选定 RN + RNOH 后的具体架构、版本策略、认证改造和实施阶段见 [31-mobile-rnoh-implementation.md](./31-mobile-rnoh-implementation.md)。

- [React Native for OpenHarmony 当前主仓（CPF-RN）](https://gitcode.com/CPF-RN/ohos_react_native)
- [RNOH 文档目录](https://gitcode.com/CPF-RN/ohos_react_native/tree/main/docs/zh-cn)
- [RNOH Tags](https://gitcode.com/CPF-RN/ohos_react_native/tags)
- [Flutter 3.41 OpenHarmony Release](https://developer.huawei.com.cn/consumer/cn/forum/topic/0201223234884554700?fid=0109140870620153026)
- [Flutter-OH 3.41 发布](https://developer.huawei.com/consumer/cn/forum/topic/0204225021093038138)
- [Kuikly 架构](https://kuikly.tds.qq.com/Introduction/arch.html)
- [Kuikly Roadmap 2025](https://kuikly.tds.qq.com/Blog/roadmap2025.html)
- [Kuikly Roadmap 2026](https://raw.githubusercontent.com/tencent-tds/kuiklyui/eddfda292aca82b879158859046aa85808e0eca0/docs/Blog/Kuikly%20Roadmap%EF%BC%882026%EF%BC%89.md)
- [uni-app x HarmonyOS 开发指南](https://doc.dcloud.net.cn/uni-app-x/app-harmony/)
- [uni-app x 介绍](https://doc.dcloud.net.cn/uni-app-x/)
- [HarmonyOS 多窗口/宽屏适配](https://developer.huawei.com/consumer/en/doc/best-practices-V14/bpta-multi-window-V14)
- [HarmonyOS 折叠屏布局自适应问答](https://developer.huawei.com/consumer/cn/forum/topic/0204224332693110462?fid=0109140870620153026)
- [Kuikly 折叠屏适配](https://kuikly.tds.qq.com/home/blog/386)
- [Tauri HarmonyOS 社区讨论](https://bbs.itying.com/topic/69e26ad5c504c50058fd6dde#69e26af8dc652800528fdf9f6)
