# 一卡通、free/plus/pro 池组设计方案

本文用于设计 new-api 中“一张 API Key 调用多家大模型”的一卡通能力，并结合 free、plus、pro 三类账号池进行资源隔离、资源池选择、路由调度和统一计费。

## 1. 目标

最终目标：用户只需要一个 API Key，就可以调用项目支持的多种大模型，例如 GPT、Claude、Gemini、DeepSeek 等。

| 目标 | 说明 |
|---|---|
| 一个 API Key | 用户侧只保存一个 `sk-xxx` |
| 多模型调用 | 用户通过 `model` 字段选择 GPT、Claude、Gemini 等模型 |
| 自动路由 | 系统根据 `API Key group + model` 自动选择对应渠道池 |
| 资源池隔离 | API Key 可选择 `free`、`plus`、`pro` 资源池；账号进入哪个池由管理员手动决定 |
| 统一计费 | 无论上游是 GPT、Claude 还是 Gemini，都走统一额度/订阅/日志体系 |
| 统一体验 | 用户不需要知道背后使用的是 OpenAI、Codex、Claude、Gemini 还是 Vertex |

一句话总结：

| 核心原则 | 说明 |
|---|---|
| 一卡通不是让一个渠道支持所有模型 | 而是让一个 API Key 通过真实模型名自动路由到不同渠道池 |

## 2. 当前项目已有基础

当前 new-api 已经具备实现一卡通的主要基础设施。

| 现有能力 | 当前状态 | 可复用方式 |
|---|---:|---|
| 用户 API Key | 已有 | `tokens` 表 |
| API Key 分组 | 已有 | `tokens.group` |
| 渠道分组 | 已有 | `channels.group` |
| 渠道能力索引 | 已有 | `abilities(group, model, channel_id)` |
| 按模型选渠道 | 已有 | `model + group` 查候选渠道 |
| 权重/优先级 | 已有 | `priority`、`weight` |
| 池组倍率配置 | 已有 | `GroupRatio`、`GroupGroupRatio` |
| 多 provider adaptor | 已有 | OpenAI、Claude、Gemini、Codex 等 |
| 模型映射 | 已有 | `model_mapping` |
| 统一计费 | 基本已有 | 官方模型价格、分组倍率、订阅、钱包额度 |
| 日志 | 已有 | 记录 user/token/model/channel/group |
| 订阅升级 | 基本已有 | 订阅计划可绑定升级分组 |

因此本方案不新建独立“账号池系统”，固定复用：

| 能力 | 设计用法 |
|---|---|
| `token.group` | 决定用户 API Key 使用哪个资源池 |
| `channel.group` | 决定渠道属于哪个资源池 |
| `abilities` | 决定某分组下某模型有哪些可用渠道 |

核心判断：**new-api 现有的 token、channel、group、ability、model_mapping 设计已经能表达一卡通的主要诉求**。因此 onecard 不应该重做账号池、渠道池、路由表，而应该作为“策略增强层”挂在现有流程上。

| 不新建的东西 | 复用的现有设计 | 原因 |
|---|---|---|
| 独立账号池表 | `channels` | 一个上游账号一个 channel，已经能禁用、统计、调度 |
| 独立池组表 | `channel.group`、`token.group` | 当前 group 已能表达 free/plus/pro 资源池 |
| 独立路由表 | `abilities` | 已经是 `group + model -> channel` 的路由索引 |
| 独立模型表 | `model_mapping`、官方模型价格配置 | 已经能处理真实模型名和上游 provider 特定名称差异 |
| 独立 API Key 系统 | `tokens` | 用户 API Key、额度、模型限制、分组都已有 |

更准确地说：**不是不做账号池能力，而是不新建一套账号池数据系统**。当前项目已有能力已经可以承担账号池系统的大部分核心功能。

| 账号池系统需要的能力 | 当前项目是否已有 | 现有实现承载方式 | 是否适合复用 |
|---|---:|---|---:|
| 池组划分 | 有 | `channel.group` 配置 `free`、`plus`、`pro` | 是 |
| 用户 API Key 绑定池组 | 有 | `token.group` 决定该 key 使用哪个资源池 | 是 |
| 用户选择池组并按池计费 | 有 | `tokens.group`、`GroupRatio`、`GroupGroupRatio` | 是 |
| 模型到账号池路由 | 有 | `abilities(group, model, channel_id)` | 是 |
| 一个池内多个账号轮询 | 有 | 多个 channel 绑定同一个 group 和 model | 是 |
| 权重/优先级调度 | 有 | `priority`、`weight` | 是 |
| 账号启用/禁用 | 有 | `channel.status`、`UpdateChannelStatus` | 是 |
| 异常自动禁用 | 有 | `auto_ban`、`DisableChannel` | 是 |
| 按池计费 | 有 | 官方模型价格、group ratio、日志 group | 是 |
| 按账号统计用量 | 有 | `channel.used_quota`、日志 `channel_id` | 是 |
| 按池查看模型 | 有 | `GetGroupEnabledModels(group)` | 是 |
| 批量导入账号 | 部分有 | 有批量插入 channel 能力，但缺 provider 友好的导入入口 | 复用底座，增强入口 |
| 账号池组归属 | 已有基础 | 账号导入或编辑时由管理员指定 `free`、`plus`、`pro` 池组，不自动识别账号等级 | 复用 `channel.group` |
| 池健康评分 | 部分有 | 有禁用、余额、响应时间字段，但缺统一健康评分策略 | 复用数据，增强算法 |
| 池级可视化运营 | 部分有 | 现有渠道列表、日志、标签可辅助；缺专门池视图 | 复用数据，增强页面 |

所以实现重点不是“造账号池”，而是在当前架构上补三件事：

| 需要补的能力 | 为什么补 | 落地点 |
|---|---|---|
| 池组策略解释 | 把 `free`、`plus`、`pro` 的业务规则封装起来 | `pkg/onecard/pool` |
| Provider 账号导入 | 把 Codex/OpenAI/Claude/Gemini 的凭证差异封装起来 | `pkg/onecard/provider`、`pkg/onecard/importer` |
| 特殊接口说明 | 保持各渠道当前支持接口边界，Codex 仍按现状只支持 Responses 相关入口 | `pkg/onecard/compat` 可仅做能力说明和错误提示 |

换句话说，当前项目已经有“账号池的骨架”，onecard 只需要补“产品化肌肉”和“业务规则大脑”。这样既能实现账号池效果，又不会重做一套和主项目并行的数据体系。

## 3. 总体架构

### 3.1 用户侧体验

用户只拿一个 API Key：

```text
Authorization: Bearer sk-user-one-card
```

调用不同模型：

```json
{
  "model": "gpt-5",
  "messages": [
    {
      "role": "user",
      "content": "你好"
    }
  ]
}
```

或：

```json
{
  "model": "claude-sonnet",
  "messages": [
    {
      "role": "user",
      "content": "你好"
    }
  ]
}
```

用户不需要知道背后实际使用哪个上游账号。

### 3.2 内部架构

| 层 | 作用 | 示例 |
|---|---|---|
| 用户 API Key | 一卡通入口 | `sk-user-xxx` |
| Token Group | 用户 API Key 选择的资源池 | `free`、`plus`、`pro` |
| Model Name | 用户选择的模型能力 | `gpt-5`、`claude-sonnet`、`gemini-pro` |
| Ability 表 | 路由索引 | `group + model -> channel_id` |
| Channel Pool | 真实上游账号池 | Codex、OpenAI、Claude、Gemini、Vertex |
| Adaptor | 协议转换 | OpenAI/Claude/Gemini/Codex adaptor |
| Billing | 统一计费 | 官方模型价格、分组倍率、订阅额度 |

## 4. free/plus/pro 池组设计

### 4.1 资源池定义

| 池组 | group | 账号放置规则 | 资源/计费定位 |
|---|---|---|---|
| free 池 | `free` | 管理员指定进入 free 池的任意账号/API Key | 低倍率资源池 |
| plus 池 | `plus` | 管理员指定进入 plus 池的任意账号/API Key | 更高倍率资源池 |
| pro 池 | `pro` | 管理员指定进入 pro 池的任意账号/API Key | 最高倍率资源池 |

> 决策：正式池组统一为 `free`、`plus`、`pro`。`free` 是新建资源池组，和项目原有 `default` 分组没有任何继承、别名或迁移关系。

> 重要约束：账号放入哪个池组完全由管理员决定，不根据账号自身类型、账号套餐、账号持有者用户类型自动判断。比如管理员可以把一个 plus 账号放进 `free` 池给选择 free 池的 API Key 使用，也可以把普通账号放进 `plus` 或 `pro` 池；系统只按该账号所在的 `channel.group` 做路由和计费。

### 4.2 各池放置内容

| 池组 | GPT 资源 | Claude 资源 | Gemini 资源 | 其他资源 |
|---|---|---|---|---|
| free | 管理员放入 free 池的 GPT/Codex/OpenAI 账号 | 管理员放入 free 池的 Claude 账号 | 管理员放入 free 池的 Gemini 账号 | 低倍率资源 |
| plus | 管理员放入 plus 池的 GPT/Codex/OpenAI 账号 | 管理员放入 plus 池的 Claude 账号 | 管理员放入 plus 池的 Gemini 账号 | 中倍率资源 |
| pro | 管理员放入 pro 池的 GPT/Codex/OpenAI 账号 | 管理员放入 pro 池的 Claude 账号 | 管理员放入 pro 池的 Gemini 账号 | 高倍率资源 |

### 4.3 渠道组织方式

固定采用：**一个上游账号 = 一个 new-api 渠道**。

这样可以直接复用现有 channel 的启用、禁用、刷新、查用量、定位异常、统计成本、按权重调度等能力。账号属于哪个资源池，由该渠道的 `channel.group` 决定。

示例：

| 池组 | 账号数量 | 渠道数量 | 渠道 group |
|---|---:|---:|---|
| free | 1000 | 1000 | `free` |
| plus | 1000 | 1000 | `plus` |
| pro | 1000 | 1000 | `pro` |

## 5. 数据模型设计

### 5.1 Channel

| 字段 | 用途 | 示例 |
|---|---|---|
| `type` | 渠道类型 | Codex 为 `57`，Claude/Gemini/OpenAI 使用各自类型 |
| `key` | 上游凭证 | API Key 或 Codex OAuth JSON |
| `base_url` | 上游地址 | `https://chatgpt.com`、`https://api.openai.com` |
| `models` | 该渠道支持的模型列表 | `gpt-5,gpt-5-codex` |
| `group` | 所属资源池 | `free`、`plus`、`pro` |
| `priority` | 优先级 | 同池分层 |
| `weight` | 权重 | 同优先级内随机权重 |
| `tag` | 管理标签 | `codex-free`、`claude-plus` |
| `status` | 是否启用 | enabled/disabled |
| `auto_ban` | 异常自动禁用 | 必须开启 |

### 5.2 Token

| 字段 | 用途 | 示例 |
|---|---|---|
| `key` | 用户 API Key | `sk-xxx` |
| `group` | 该 key 使用的资源池 | `free`、`plus`、`pro` |
| `model_limits` | 限制可调用模型 | 可用于套餐内模型限制 |
| `remain_quota` | 余额额度 | 钱包/预付费场景 |
| `unlimited_quota` | 是否无限额度 | 订阅/管理员场景 |
| `cross_group_retry` | 跨分组重试 | 一卡通默认不依赖该字段；`auto` 的池组访问顺序由 `AutoGroups` 控制 |

### 5.3 Ability

`Ability` 是路由索引，决定某个 group 下某个 model 可以使用哪些 channel。

| 字段 | 用途 |
|---|---|
| `group` | 资源池分组 |
| `model` | 用户请求模型 |
| `channel_id` | 可用渠道 |
| `enabled` | 是否启用 |
| `priority` | 优先级 |
| `weight` | 权重 |
| `tag` | 渠道标签 |

当渠道创建或更新时，会根据 `channel.group` 和 `channel.models` 自动维护 abilities。

## 6. 路由调用流程

用户请求进入系统后的核心流程：

| 步骤 | 处理 | 结果 |
|---|---|---|
| 1 | `TokenAuth` 校验 API Key | 得到 token、user |
| 2 | 读取 `token.group` | 确定使用资源池 |
| 3 | 校验 `token.group` 是否为合法资源池 | 确保 group 属于 `free`、`plus`、`pro` |
| 4 | `Distribute` 读取请求模型 | 得到 `model` |
| 5 | 根据 `group + model` 查候选渠道 | 查询缓存或 DB 中的 abilities |
| 6 | 按 priority/weight 选择渠道 | 得到具体上游账号 |
| 7 | 进入对应 adaptor | OpenAI/Claude/Gemini/Codex |
| 8 | 协议转换并请求上游 | 发起真实上游调用 |
| 9 | 返回统一响应 | 用户无感知 |
| 10 | 统一计费与日志 | 记录 user/token/group/model/channel |

简化公式：

```text
API Key -> token.group -> 请求 model -> abilities(group, model) -> channel -> adaptor -> upstream
```

## 7. 模型命名与模型目录设计

一卡通最重要的是“管理端和用户端看到同一套真实模型名”。不再额外设计一套营销别名，用户请求什么真实模型名，后台也按同一个模型名展示、定价和统计。

### 7.1 用户请求模型名

| 真实模型名 | 说明 | 可能路由渠道 |
|---|---|---|
| `gpt-5` | GPT 模型 | Codex / OpenAI |
| `gpt-5-codex` | Codex 模型 | Codex |
| `claude-sonnet-4-5` | Claude Sonnet | Anthropic / Vertex Claude |
| `claude-opus-4-1` | Claude Opus | Anthropic 高倍率池 |
| `gemini-2.5-pro` | Gemini Pro | Gemini / Vertex |
| `deepseek-chat` | DeepSeek | DeepSeek |
| `text-embedding-3-large` | 向量模型 | OpenAI / Jina / Cohere |

### 7.2 内部模型映射

| 方式 | 用途 |
|---|---|
| `channels.models` | 声明渠道可处理哪些真实模型名 |
| `model_mapping` | 仅在上游同一模型存在 provider 特定名称时使用，默认不做营销别名映射 |
| 官方模型价格配置 | 按真实模型名维护官方标准定价 |
| 模型展示配置 | 控制前端展示哪些真实模型名 |

示例：

| 展示/请求模型 | plus 池渠道 | 上游模型 |
|---|---|---|
| `gpt-5` | Codex Plus 渠道 | `gpt-5` |
| `claude-sonnet-4-5` | Claude Plus 渠道 | `claude-sonnet-4-5` |
| `gemini-2.5-pro` | Gemini Plus 渠道 | `gemini-2.5-pro` |

## 8. 资源池选择规则设计

### 8.1 用户类型和资源池组解耦

不要把“用户身份”和“资源池”混成同一个概念。用户类型仍然可以使用现有 `{ "default": 1, "svip": 1, "vip": 1 }`；账号池组固定为 `free`、`plus`、`pro`，本质是不同扣费倍率和资源路由分组。

| 类型 | 示例 | 作用 |
|---|---|---|
| 用户类型 | `default`、`vip`、`svip` | 表示用户身份、后台运营标签 |
| 资源池组 | `free`、`plus`、`pro` | 表示账号池/渠道池 |
| Token 使用组 | `free`、`plus`、`pro`、`auto` | 表示该 API Key 走固定池或 new-api 原生 auto 池 |

### 8.2 API Key 池组选择规则

当前业务决策：不同类型用户创建 API Key 时，都可以根据自己的预算和使用需求选择 `free`、`plus`、`pro`、`auto` 中任意 group。`free`、`plus`、`pro` 是实体资源池；`auto` 复用 new-api 原生自动分组能力。

| 用户类型 | 可选择 API Key group | 说明 |
|---|---|---|
| `default` | `free`、`plus`、`pro`、`auto` | 只要用户有权益卡或永久余额，就可以选择任意池组 |
| `vip` | `free`、`plus`、`pro`、`auto` | 用户自行选择成本和质量 |
| `svip` | `free`、`plus`、`pro`、`auto` | 用户自行选择成本和质量 |

`auto` 不是实体账号池。用户选择该 group 时，系统按 `AutoGroups = ["free", "plus", "pro"]` 的顺序选择账号池；实际调用哪个池组，就按哪个池组倍率计费。

## 9. 计费设计

一卡通必须统一计费，避免上游成本和扣费口径混乱。计费规则固定为：**官方标准模型价格 × 实际调用账号池倍率 × 实际 usage**。

| 计费维度 | 规则 |
|---|---|
| 模型基础价格 | 采用对应模型的官方标准定价，管理端和用户端展示一致 |
| 账号池倍率 | `free`、`plus`、`pro` 分别配置不同倍率 |
| 实际计费池组 | 按最终实际调用的 `using_group` 计费；发生 fallback 时按 fallback 后的账号池倍率计费 |
| 订阅额度 | 日卡、周卡、月卡按订阅周期发放额度 |
| 超额策略 | 余额扣费或拒绝请求 |
| 日志统计 | 记录真实模型、渠道、`requested_group`、`using_group`、token、实际 usage |
| 成本分析 | 后台按渠道 tag / group 汇总成本 |

计费示例：

| 请求模型 | 官方标准价格 | API Key 选择池组 | 实际调用池组 | 最终扣费 |
|---|---|---|---|---|
| `gpt-5` | 按官方 `gpt-5` 定价 | `free` | `free` | 官方价格 × `free` 池倍率 × usage |
| `claude-sonnet-4-5` | 按官方 Claude Sonnet 定价 | `auto` | `plus` | 官方价格 × `plus` 池倍率 × usage |
| `gemini-2.5-pro` | 按官方 Gemini Pro 定价 | `pro` | `pro` | 官方价格 × `pro` 池倍率 × usage |

核心原则：模型价格只跟真实模型的官方标准定价有关，账号池价格差异只通过 `free`、`plus`、`pro` 的池组倍率体现。

官方标准模型价格由后台配置维护。后台以真实模型名为 key 维护官方价格、币种、单位和生效时间；计费模块只读取已生效的后台配置，不在请求链路里联网查询价格。

价格缺失处理固定为硬失败：onecard 请求的真实模型名没有已生效官方价格配置时，直接拒绝请求并返回“模型价格未配置”错误，不允许按 0 价格、默认价格或旧倍率逻辑继续扣费。

## 10. Codex 特殊处理

Codex 当前是 Responses-only 渠道，不直接支持 `/v1/chat/completions`。

| 项目 | 当前行为 |
|---|---|
| 支持 | `/v1/responses`、`/v1/responses/compact` |
| 默认不支持 | `/v1/chat/completions` |
| 上游路径 | `/backend-api/codex/responses` |

一卡通不新增 Codex Chat Completions 兼容转换，保持当前项目现状：Codex 渠道只支持 `/v1/responses` 和 `/v1/responses/compact`。如果用户使用 Codex 池，需要客户端直接调用 Responses 相关入口。

### 10.1 产品规则

| 场景 | 行为 |
|---|---|
| 用户请求 `/v1/responses` 且命中 Codex 渠道 | 正常转发到 Codex 后端 |
| 用户请求 `/v1/responses/compact` 且命中 Codex 渠道 | 正常转发到 Codex compact 后端 |
| 用户请求 `/v1/chat/completions` 且命中 Codex 渠道 | 保持现状，返回接口不支持 |
| 用户希望使用 Chat Completions | 应选择支持 Chat Completions 的 OpenAI-compatible 渠道池 |

### 10.2 固定接口策略

| 策略 | 决策 |
|---|---|
| 代码内置 Codex 自动 Chat -> Responses 转换 | 不做，保持现状 |
| 管理端配置 Chat -> Responses 兼容策略 | 不作为一卡通默认能力 |
| UI 提示 | 必须提示 Codex 是 Responses-only 渠道，避免用户误用 |

## 11. 账号池导入与管理

3000+ 账号不适合手动录入，必须做导入能力。

| 导入入口 | 用途 | 规则 |
|---|---|---|
| 后台上传 JSON/CSV | 管理员可视化导入 | 作为主要导入入口 |
| CLI 脚本调用 API | 初始化批量导入 | 调用同一套导入 API，不绕过业务校验 |
| 直接写数据库 | 数据修复或迁移 | 不作为常规入口，必须同步 abilities 和缓存 |
| 手动添加 | 少量调试 | 不用于大规模账号导入 |

固定导入格式：

```json
[
  {
    "pool": "free",
    "provider": "codex",
    "email": "free001@example.com",
    "credential": {
      "access_token": "...",
      "refresh_token": "...",
      "account_id": "...",
      "email": "free001@example.com",
      "type": "codex",
      "expired": "..."
    }
  },
  {
    "pool": "plus",
    "provider": "claude",
    "email": "plus-claude-001@example.com",
    "credential": {
      "api_key": "..."
    }
  }
]
```

导入生成规则：

| 字段 | 生成方式 |
|---|---|
| `name` | `{provider}-{pool}-{email/account_id}` |
| `type` | 根据 provider 映射渠道类型 |
| `group` | `pool` |
| `key` | credential 转为渠道 key |
| `models` | 根据 provider + pool 默认模型列表 |
| `base_url` | 根据 provider 默认值 |
| `tag` | `{provider}-{pool}` |
| `status` | 默认启用 |

## 12. 池健康与风控

一卡通产品化后，池管理比接口转换更重要。

| 风险 | 方案 |
|---|---|
| 单账号被打爆 | 按账号 usage/失败率自动降权或禁用 |
| 上游账号失效 | 自动刷新失败后禁用渠道 |
| 池容量不足 | 看板展示每个池可用渠道数量 |
| 池组误用 | 明确展示 API Key group、实际调用 group 和计费倍率 |
| 某模型无可用渠道 | 明确返回模型/套餐不可用 |
| 同 IP 大量账号风控 | 支持代理、分散出口、限速 |
| 用户滥用 | token 额度、模型限制、IP 限制、速率限制 |
| 成本失控 | 按 group/model/channel/tag 汇总成本 |

必须增加池健康看板：

| 指标 | 说明 |
|---|---|
| 总账号数 | 每个池总渠道数 |
| 可用账号数 | `status=enabled` 且近期测试通过 |
| 失败率 | 按渠道、模型、池统计 |
| 平均延迟 | 用于动态路由 |
| 剩余额度/usage | 支持的上游尽量拉取 |
| 自动禁用数量 | 用于告警 |

## 13. 调度策略

调度策略使用当前项目已有能力。固定池组直接走对应实体池；`auto` 复用 new-api 原生自动分组能力，并通过 `AutoGroups = ["free", "plus", "pro"]` 固定访问顺序。

| 规则 | 当前支持 |
|---|---:|
| 同池随机 | 支持 |
| 优先级 | 支持 |
| 权重 | 支持 |
| 同池重试 | 支持 |
| 跨池 fallback | 复用 new-api 原生 `auto` 分组；只有 API Key group 为 `auto` 时执行 |

默认调度参数：

| 池 | priority | weight |
|---|---:|---:|
| free | 0 | 0 |
| plus | 0 | 0 |
| pro | 0 | 0 |

跨池 fallback 策略：

| API Key group | 资源池选择顺序 |
|---|---|
| `free` | `free` |
| `plus` | `plus` |
| `pro` | `pro` |
| `auto` | `free -> plus -> pro` |

增强能力：

| 增强项 | 说明 |
|---|---|
| 根据失败率降权 | 失败越多权重越低 |
| 根据剩余额度调度 | 额度低的账号减少使用 |
| 根据延迟调度 | 低延迟渠道优先 |
| 根据成本调度 | 同模型多上游时优先低成本 |

## 14. 独立功能模块与低侵入实现方案

这一节是本方案的实现约束：一卡通能力必须封装成独立功能模块，尽量少改 new-api 主流程。后续从主干合代码时，最好像插一块乐高，而不是在老代码里撒芝麻。

### 14.1 核心原则

| 原则 | 说明 |
|---|---|
| 独立模块 | 新增 `pkg/onecard` 或 `service/onecard`，一卡通策略、池组选择、provider 差异、导入逻辑都收敛在模块内 |
| 单点接入 | 现有主流程只调用一个 Facade，例如 `onecard.Resolve(...)`，不要在多个文件里散落 free/plus/pro 或 Codex 判断 |
| 默认不改变旧行为 | 增加功能开关，例如 `onecard.enabled`；关闭时完全回落到当前 `token.group + model -> channel` 逻辑 |
| 复用现有设计 | 优先复用 `token.group`、`channel.group`、`abilities`、`model_mapping`、`channels`、`tokens`，默认不新增数据库表 |
| 面向对象封装 | 用父接口定义能力，用基础结构体承载通用逻辑，用不同子类实现具体业务差异 |
| 合主干友好 | 对 `middleware`、`service/channel_select.go`、`relay/compatible_handler.go` 只留薄 hook，降低冲突概率 |

Go 没有 Java/C++ 那种传统继承，本方案统一使用 `interface + struct embedding` 实现面向对象里的“父类定义方法、子类重写实现、多态调用”。

### 14.1.1 面向对象设计约束

这个约束不是只针对 Provider 账号导入，而是一卡通模块的全局设计原则：只要出现“同一类能力 + 多种业务差异”的场景，都必须采用父接口、基础父类、子类实现、注册表/工厂多态分发的方式设计。

主流程只能依赖父接口和 Facade，不允许在主流程里散落大量 `if provider == "codex"`、`if pool == "plus"`、`if cardType == "month"` 这类判断。差异逻辑要关进对应子类里，像把不同味道的火锅底料分锅装好，不要全倒进一口锅里硬煮。

| 适用场景 | 父接口/抽象能力 | 基础父类 | 子类实现示例 | 必须隔离的差异 |
|---|---|---|---|---|
| 池组策略 | `PoolStrategy` | `BasePoolStrategy` | `FreePoolStrategy`、`PlusPoolStrategy`、`ProPoolStrategy`、`AutoPoolStrategy` | 固定池解析、原生 auto 配置校验、实际计费池组 |
| Provider 能力 | `ProviderAdapter` | `BaseProvider` | `CodexProvider`、`OpenAIProvider`、`ClaudeProvider`、`GeminiProvider` | 凭证格式、接口能力、模型映射、健康检查 |
| 接口能力说明 | `EndpointPolicy` | `BaseEndpointPolicy` | `CodexEndpointPolicy`、`OpenAIEndpointPolicy`、`ClaudeEndpointPolicy` | Chat、Responses、Messages 等接口差异 |
| 账号导入格式 | `ImportParser` | `BaseImportParser` | `Sub2APIParser`、`JSONParser`、`CSVParser` | 字段解析、凭证校验、错误提示 |
| 扣费资金来源 | `FundingSource` | `BaseFundingSource` | `DayCardSource`、`WeekCardSource`、`MonthCardSource`、`PermanentBalanceSource` | 重置周期、可用额度、扣费顺序、余额消耗 |
| 跨池 fallback | `FallbackStrategy` | `BaseFallbackStrategy` | `AutoFallbackStrategy` | 复用 new-api 原生 `auto`，按 `AutoGroups = ["free","plus","pro"]` 执行、禁止降级 |
| 健康检查 | `HealthChecker` | `BaseHealthChecker` | `CodexHealthChecker`、`OpenAIHealthChecker`、`ClaudeHealthChecker` | 测试接口、成功条件、异常归类 |

统一代码形态如下，后续新增任何同类能力都按这个模板落地：

```go
type OneCardStrategy interface {
    Name() string
}

type BaseStrategy struct {
    name string
}

func (s *BaseStrategy) Name() string {
    return s.name
}

type StrategyRegistry struct {
    strategies map[string]OneCardStrategy
}

func (r *StrategyRegistry) Register(strategy OneCardStrategy) {
    r.strategies[strategy.Name()] = strategy
}

func (r *StrategyRegistry) Get(name string) OneCardStrategy {
    return r.strategies[name]
}
```

落地要求很简单：新增类型时优先新增子类和注册项，不优先修改主流程。只有 Facade、注册表、基础父类允许知道“有哪些子类”，业务主流程只负责调用接口。

### 14.2 目录结构

| 路径 | 职责 |
|---|---|
| `pkg/onecard` | 一卡通策略增强模块，对外暴露 Facade，不替代现有账号池/渠道池 |
| `pkg/onecard/pool` | free、plus、pro 池组策略 |
| `pkg/onecard/provider` | Codex、OpenAI、Claude、Gemini 等 provider 封装 |
| `pkg/onecard/router` | 模型解析、池组解析、渠道选择策略 |
| `pkg/onecard/compat` | Chat Completions、Responses、Codex 等接口兼容策略 |
| `pkg/onecard/importer` | 账号批量导入、凭证校验、渠道生成 |
| `pkg/onecard/health` | 账号健康、失败率、额度、风控状态 |

### 14.3 模块边界

| 模块 | 对外暴露 | 内部封装 |
|---|---|---|
| OneCard Facade | `Resolve`、`ImportAccounts`、`CheckAccess` | 编排池组、provider、兼容、选择器，并复用现有 token/channel/ability |
| Pool Strategy | `ResolvePool`、`ValidateAccess`、`BuildChannelQuery` | free/plus/pro 的固定资源池合法性、auto 配置校验；实际池组仍落在现有 group 字段 |
| Provider Adapter | `NormalizeCredential`、`BuildChannel`、`SupportedInterfaces` | Codex OAuth JSON、OpenAI Key、Claude Key 等差异；最终生成现有 channel 数据 |
| Endpoint Policy | `ValidateEndpoint`、`SupportedInterfaces` | Codex Responses-only、OpenAI Chat/Responses、Claude Messages 等接口能力说明 |
| Channel Selector | `Select` | 在现有 channel candidates 上叠加权重、优先级、健康分、成本策略 |
| Importer | `Parse`、`Validate`、`CreateChannels` | JSON/CSV/sub2api 等格式解析，落库仍走现有渠道创建逻辑 |

### 14.4 复用边界

onecard 模块只做“解释、约束、编排”，不重新拥有数据。这是基于当前代码结构得出的正式复用边界。

| 领域 | 固定做法 | onecard 的角色 |
|---|---|---|
| 用户 API Key | 继续使用 `tokens` | 读取 token group，做资源池策略解释 |
| 资源池 | 继续使用 `channel.group` | 将 free/plus/pro 映射为现有 group |
| 账号凭证 | 继续放在 `channels.key` | provider 子类只负责校验和标准化 |
| 路由索引 | 继续使用 `abilities` | 只构造查询条件或补充策略过滤 |
| 模型映射 | 继续使用 `model_mapping` | 只处理 provider 特定上游名称差异，默认使用真实模型名 |
| 计费 | 继续使用当前倍率、额度、日志 | onecard 只补充 pool/provider/decision 上下文 |
| 渠道管理 | 继续使用当前 channel CRUD | 批量导入最终仍创建普通 channel |

一句话：onecard 是“大脑皮层”，现有 `tokens/channels/abilities` 还是“骨架和肌肉”。这样比较不容易把系统改成一锅热闹但难合并的粥。

### 14.5 父类接口与子类策略设计

父接口定义“所有池组都必须具备的行为”：

```go
type PoolStrategy interface {
    Name() string
    ResolvePool(ctx *RequestContext) (*PoolDecision, error)
    ValidateAccess(ctx *RequestContext, decision *PoolDecision) error
    BuildChannelQuery(ctx *RequestContext, decision *PoolDecision) (*ChannelQuery, error)
}
```

基础父类承载公共逻辑，子类通过嵌入它来复用能力：

```go
type BasePoolStrategy struct {
    config Config
}

func (s *BasePoolStrategy) ValidateAccess(ctx *RequestContext, decision *PoolDecision) error {
    // 通用校验：token group 是否是 free/plus/pro/auto，auto 是否配置了合法访问顺序
    return nil
}
```

不同业务子类只实现自己的差异：

```go
type FreePoolStrategy struct {
    BasePoolStrategy
}

func (s *FreePoolStrategy) Name() string {
    return "free"
}

func (s *FreePoolStrategy) ResolvePool(ctx *RequestContext) (*PoolDecision, error) {
    // free 池：固定只使用 free
    return &PoolDecision{Pool: "free"}, nil
}

type PlusPoolStrategy struct {
    BasePoolStrategy
}

func (s *PlusPoolStrategy) ResolvePool(ctx *RequestContext) (*PoolDecision, error) {
    // plus 池：固定只使用 plus
    return &PoolDecision{Pool: "plus"}, nil
}

type ProPoolStrategy struct {
    BasePoolStrategy
}

func (s *ProPoolStrategy) ResolvePool(ctx *RequestContext) (*PoolDecision, error) {
    // pro 池：优先高质量账号，可配置专属池和更严格健康检查
    return &PoolDecision{Pool: "pro"}, nil
}

type AutoPoolStrategy struct {
    BasePoolStrategy
}

func (s *AutoPoolStrategy) Name() string {
    return "auto"
}

func (s *AutoPoolStrategy) ResolvePool(ctx *RequestContext) (*PoolDecision, error) {
    // auto：不产出最终计费池，只校验 AutoGroups 并声明候选顺序。
    // 最终 using_group 由 new-api 原生 auto 选中 channel 后写入 ContextKeyAutoGroup / relayInfo.UsingGroup。
    return &PoolDecision{Pool: "auto", FallbackPools: config.GetAutoGroups()}, nil
}
```

示例代码省略 import 和配置对象初始化，实际实现从系统配置层读取 `AutoGroups`。

策略工厂负责多态分发：

```go
type StrategyFactory interface {
    GetPoolStrategy(group string) PoolStrategy
}
```

调用方只面向 `PoolStrategy` 接口，不关心实际是 free、plus 还是 pro。新增 `enterprise` 池时，只要增加 `EnterprisePoolStrategy`，主流程不用长出一堆 `if group == "enterprise"`。

### 14.6 Provider 子类设计

不同 provider 的凭证格式、接口类型、模型映射、健康检查都不同，必须封装到 provider 子类里。

Provider 账号导入必须采用面向对象设计模式：父接口定义统一能力，基础父类封装通用流程，不同 provider 子类重写差异逻辑，通过工厂/注册表进行多态调用。主流程只依赖父接口，不直接判断 `Codex/OpenAI/Claude/Gemini`。

```go
type ProviderAdapter interface {
    Type() int
    Name() string
    SupportedInterfaces() []InterfaceType
    NormalizeCredential(raw []byte) (*Credential, error)
    BuildChannel(input *AccountImportItem, pool string) (*ChannelDraft, error)
    HealthCheck(channelID int) (*HealthResult, error)
}
```

父类接口职责：

| 方法 | 父接口定义 | 子类重写点 |
|---|---|---|
| `Type()` | 返回 new-api 渠道类型 | 各 provider 返回自己的 channel type |
| `Name()` | 返回 provider 名称 | `codex`、`openai`、`claude`、`gemini` |
| `SupportedInterfaces()` | 返回支持的接口能力 | Codex 只返回 Responses，OpenAI 返回 Chat/Responses/Image 等 |
| `NormalizeCredential()` | 把导入凭证标准化 | 子类解析不同凭证格式 |
| `BuildChannel()` | 构造可落库的 channel 草稿 | 子类决定 key、base_url、models、tag、extra |
| `HealthCheck()` | 测试账号是否可用 | 子类调用对应 provider 的测试方式 |

基础父类封装通用逻辑：

```go
type BaseProvider struct {
    channelType int
    name        string
}

func (p *BaseProvider) Type() int {
    return p.channelType
}

func (p *BaseProvider) Name() string {
    return p.name
}

func (p *BaseProvider) BuildBaseChannel(input *AccountImportItem, pool string) *ChannelDraft {
    return &ChannelDraft{
        Name:     input.DisplayName,
        Group:    pool,
        Tag:      p.name + "-" + pool,
        Priority: input.Priority,
        Weight:   input.Weight,
        Status:   "enabled",
    }
}
```

子类只实现差异：

```go
type CodexProvider struct {
    BaseProvider
}

func (p *CodexProvider) SupportedInterfaces() []InterfaceType {
    return []InterfaceType{InterfaceResponses, InterfaceResponsesCompact}
}

func (p *CodexProvider) NormalizeCredential(raw []byte) (*Credential, error) {
    // 解析 sub2api / OAuth JSON，校验 access_token，提取 account_id、refresh_token 等字段
    return normalizeCodexOAuthCredential(raw)
}

func (p *CodexProvider) BuildChannel(input *AccountImportItem, pool string) (*ChannelDraft, error) {
    ch := p.BuildBaseChannel(input, pool)
    ch.Type = p.Type()
    ch.Key = input.Credential.ToChannelKeyJSON()
    ch.BaseURL = "https://chatgpt.com"
    ch.Models = input.Models
    return ch, nil
}
```

| 子类 | 负责逻辑 | 不应该外泄到主流程的判断 |
|---|---|---|
| `CodexProvider` | OAuth JSON 校验、Responses-only 能力声明、按导入参数生成目标池组 channel | `ChannelTypeCodex == 57` 后怎么转换 |
| `OpenAIProvider` | API Key 校验、OpenAI-compatible 模型映射 | OpenAI 是否支持 chat/responses/images |
| `ClaudeProvider` | Claude Key/代理渠道配置、Claude 模型能力 | Claude 消息格式差异 |
| `GeminiProvider` | Gemini Key/Vertex 配置、模型映射 | Gemini API 形态差异 |

Provider 工厂：

```go
type ProviderFactory interface {
    GetProvider(channelType int) ProviderAdapter
    GetProviderByName(name string) ProviderAdapter
}
```

注册表示例：

```go
type ProviderRegistry struct {
    byType map[int]ProviderAdapter
    byName map[string]ProviderAdapter
}

func (r *ProviderRegistry) Register(provider ProviderAdapter) {
    r.byType[provider.Type()] = provider
    r.byName[provider.Name()] = provider
}

func NewProviderRegistry() *ProviderRegistry {
    r := &ProviderRegistry{
        byType: map[int]ProviderAdapter{},
        byName: map[string]ProviderAdapter{},
    }
    r.Register(NewCodexProvider())
    r.Register(NewOpenAIProvider())
    r.Register(NewClaudeProvider())
    r.Register(NewGeminiProvider())
    return r
}
```

导入器只面向父接口：

```go
func (i *AccountImporter) Import(ctx context.Context, req ImportRequest) (*ImportResult, error) {
    provider := i.providers.GetProviderByName(req.Provider)
    if provider == nil {
        return nil, ErrUnsupportedProvider
    }

    normalized, err := provider.NormalizeCredential(req.RawCredential)
    if err != nil {
        return nil, err
    }

    item := &AccountImportItem{
        DisplayName: req.DisplayName,
        Credential:  normalized,
        Models:      req.Models,
        Priority:    req.Priority,
        Weight:      req.Weight,
    }

    draft, err := provider.BuildChannel(item, req.Pool)
    if err != nil {
        return nil, err
    }

    return i.channelWriter.CreateFromDraft(ctx, draft)
}
```

这个结构保证：

| 设计目标 | 实现方式 |
|---|---|
| 封装 | provider 差异留在各自子类里 |
| 继承 | Go 使用 `BaseProvider` struct embedding 复用通用逻辑 |
| 多态 | importer 只依赖 `ProviderAdapter` 接口 |
| 隔离 | Codex/OpenAI/Claude/Gemini 的凭证解析、模型默认值、健康检查互不污染 |
| 低侵入 | 最终仍创建普通 `channel`，不改现有 relay/adaptor 主流程 |

### 14.7 接口能力说明策略设计

Codex 保持当前接口能力边界，不新增 Chat Completions -> Responses 自动转换。onecard 模块只需要把“该渠道支持哪些入口、不支持哪些入口”集中说明，避免在 relay 主流程里散落 provider 判断。

```go
type EndpointPolicy interface {
    Name() string
    Match(ctx *RequestContext, channel *ChannelInfo) bool
    ValidateEndpoint(ctx *RequestContext, channel *ChannelInfo) error
}
```

| 策略子类 | 匹配条件 | 行为 |
|---|---|---|
| `OpenAICompatiblePolicy` | OpenAI-compatible 渠道 | 声明支持 Chat/Responses 等现有入口 |
| `CodexEndpointPolicy` | Codex 渠道 | 只允许 Responses 相关入口；Chat Completions 返回明确错误 |
| `ClaudeMessagesPolicy` | Claude 渠道 | 声明 Claude messages 能力 |
| `NoopPolicy` | 未命中任何特殊策略 | 不额外处理，交给现有 adaptor |

这样主流程只做：

```go
policy := compatRegistry.Match(ctx, channel)
err := policy.ValidateEndpoint(ctx, channel)
```

具体“Codex 为什么不支持 Chat Completions、应该使用哪个入口”的说明全部藏在 `CodexEndpointPolicy` 内部。

### 14.8 Facade 单点调用

现有代码不要直接组合各类策略，而是只调用一个入口：

```go
type Facade interface {
    Resolve(ctx *RequestContext) (*RouteDecision, error)
    CheckAccess(ctx *RequestContext) error
    ImportAccounts(ctx context.Context, input *ImportRequest) (*ImportResult, error)
}
```

固定调用流程：

| 步骤 | Facade 内部动作 |
|---|---|
| 1 | 根据 token group 获取 `PoolStrategy` |
| 2 | 校验 token group 是否是合法资源池，并检查 fallback 方向是否合规 |
| 3 | 根据真实模型名解析模型目录 |
| 4 | 根据 provider 能力匹配候选渠道 |
| 5 | 根据健康、权重、优先级选择 channel |
| 6 | 匹配接口兼容策略 |
| 7 | 返回 `RouteDecision` 给现有 relay 流程 |

`RouteDecision` 必须包含：

| 字段 | 说明 |
|---|---|
| `RequestedPool` | 用户 API Key 原始选择的资源池，例如 `free`、`plus`、`pro` |
| `Pool` | 最终实际调用和计费的资源池，例如 `free`、`plus`、`pro` |
| `UserModel` | 用户请求模型名 |
| `UpstreamModel` | 上游真实模型名 |
| `ChannelID` | 选中的渠道 |
| `ChannelType` | provider 类型 |
| `InterfaceType` | chat、responses、images、embeddings 等 |
| `EndpointPolicy` | 命中的接口能力策略名称 |
| `FallbackUsed` | 是否发生跨池 fallback |

### 14.9 对现有项目的最小侵入接入点

| 现有位置 | 固定改动 | 侵入程度 | 回退方式 |
|---|---|---:|---|
| `middleware/auth.go` | 不按用户类型限制资源池；只确认 token group 合法，并把 group 写入上下文 | 低 | 沿用当前逻辑 |
| `middleware/distributor.go` | 不接管分发主链路，继续按 `ContextKeyUsingGroup + model` 分发 | 低 | 沿用当前逻辑 |
| `service/channel_select.go` | 不改主选择器，继续按 abilities、priority、weight 选渠道 | 低 | 沿用当前逻辑 |
| `relay/compatible_handler.go` | 不新增 Codex Chat 兼容转换；只补接口能力提示 | 低 | 没有策略时走原逻辑 |
| `controller/channel.go` | 增加批量导入入口，最终仍创建普通 channel | 低 | 不影响现有渠道 CRUD |
| `model` | 不改核心表，复用 channel/token/ability/model_mapping | 低 | 完整闭环表达不了时再加 onecard 专属表 |

最重要的一条：onecard 不接管 `TokenAuth -> Distribute -> ChannelSelect` 主链路。原因不是“先凑合”，而是当前主链路已经天然表达了 `API Key -> group -> model -> channel -> adaptor`，重做一套反而会重复鉴权、路由、计费、日志和重试逻辑。

### 14.10 模块落地顺序

| 阶段 | 目标 | 要点 |
|---|---|---|
| M1 | 分组体系落地 | 用 `GroupRatio`、`tokens.group`、`channels.group` 表达 free/plus/pro |
| M2 | 路由体系落地 | 通过 channel 的 `models/group/model_mapping` 自动生成 abilities，形成 `group + model -> channel` |
| M3 | 调度体系复用 | 复用 `priority`、`weight`、retry、auto group，不改 `service/channel_select.go` 主算法 |
| M4 | 薄 onecard 模块 | 建 `pkg/onecard`，封装池组常量、默认模型、导入校验、策略接口，不接管主路由 |
| M5 | Provider 导入增强 | 实现 `CodexProvider` 等导入器，最终仍创建普通 channel |
| M6 | 接口能力提示 | 只在必要位置接入 `CodexEndpointPolicy`，避免在 relay 主流程散落 Codex 判断 |
| M7 | 健康与运营增强 | 在现有 channel/log/used_quota/response_time 基础上做池看板和健康评分 |

### 14.11 最终结论

| 设计问题 | 最终答案 |
|---|---|
| 模块放哪里 | 默认放 `pkg/onecard`；只有强依赖业务 service 的代码放 `service/onecard` |
| 是否新增表 | 不新增核心表，复用现有 group、channel、ability、model_mapping |
| 是否新建账号池系统 | 不新建，继续使用现有 channel 作为账号承载单元 |
| 是否在老代码写 Codex 特判 | 不写，统一放进 `CodexProvider` 和 `CodexEndpointPolicy` |
| 如何体现继承 | 用 `BasePoolStrategy`、`BaseProvider` 这类基础结构体，被子类 embedding |
| 如何体现多态 | 主流程只依赖 `PoolStrategy`、`ProviderAdapter`、`EndpointPolicy` 接口 |
| 如何降低合并冲突 | 老代码只加少量 hook，复杂逻辑全在新包 |

## 15. 实现方案

这个章节定义一卡通的正式实现路径：以现有主链路作为架构底座，在旁边补 onecard 增强模块，不重做一套账号池、路由和计费系统。

### 15.1 最终实现路径

| 实现路径 | 做法 |
|---|---|
| 主链路 | 保留 `TokenAuth -> Distribute -> ChannelSelect -> Adaptor` |
| onecard 增强 | 补充导入、池策略、provider 封装、接口能力说明、计费上下文 |
| 数据复用 | 复用 `tokens`、`channels`、`abilities`、`model_mapping`、日志和计费体系 |
| 合主干策略 | 老代码只留薄 hook，复杂业务逻辑放进 onecard 独立模块 |

选择该路径的原因：

| 判断点 | 说明 |
|---|---|
| 路由模型已经匹配 | 当前已经是 `token.group + model -> abilities -> channel`，正好对应一卡通 |
| 鉴权基础可复用 | `TokenAuth` 已经能读取 token group；onecard 只补资源池合法性和计费前置检查 |
| 调度已经匹配 | `ChannelSelect` 已支持 priority、weight、retry、auto group |
| 计费已经匹配 | 现有日志、group ratio 和模型价格配置已能按池和模型计费 |
| 改造面最小 | 主要新增导入、Codex 接口边界提示、池看板，不重写主链路 |
| 合主干友好 | 老代码少动，业务差异封装到 onecard 子模块 |

所以后续开发的最优路径是：**以现有主链路作为正式架构底座，在它旁边补 onecard 增强模块，而不是重做一套账号池/路由系统**。

### 15.2 实现路线总览

| 实施步骤 | 目标 | 主要依赖现有功能 | 是否需要改代码 |
|---|---|---|---:|
| 步骤一 | 分组和倍率体系 | `GroupRatio`、`channel.group` | 配置为主 |
| 步骤二 | API Key 到资源池绑定 | `tokens.group`、`TokenAuth`、API Key group 合法性校验 | 配置/后台增强 |
| 步骤三 | 多 provider 一卡通路由 | `channels.models`、`abilities`、`model_mapping`、现有 adaptor | 极少 |
| 步骤四 | 批量导入账号到指定资源池 | `BatchInsertChannels`、`channel.Insert`、`AddAbilities` | 是，新增导入工具/接口 |
| 步骤五 | Codex 接口边界说明 | Codex adaptor 当前能力、渠道接口说明 | 是，主要是导入校验和 UI 提示 |
| 步骤六 | 订阅和权益卡打通 | 订阅权益、永久余额、API Key group 自助选择 | 少量配置或后台增强 |
| 步骤七 | 池运营看板 | `channels`、`logs`、`used_quota`、`response_time`、`status` | 是，新增查询/页面 |
| 步骤八 | onecard 策略模块沉淀 | 复用前面所有现有能力 | 是，但只做封装，不替换主流程 |

### 15.3 步骤一：分组和倍率体系

在现有分组体系里定义三个资源池。

| 配置项 | 固定值 | 作用 |
|---|---|---|
| `GroupRatio` | `free`、`plus`、`pro` | 让系统认识这三个实体资源分组，并可按组计费；`auto` 不配置倍率 |
| `UserUsableGroups` | 所有用户组都包含 `free`、`plus`、`pro`、`auto` | 所有用户组都可选择任意实体资源池或 auto group，不按用户身份限制池组 |
| `GroupGroupRatio` | 可选 | 如需按用户类型叠加特殊倍率，可作为扩展；默认按实际池组倍率 |
| `AutoGroups` | `["free", "plus", "pro"]` | 复用 new-api 原生 auto 分组，明确 auto 的访问顺序 |

配置约束固定为：一卡通启用时，所有用户组都必须可访问 `free`、`plus`、`pro`、`auto`；`GroupSpecialUsableGroup` 不得通过 `-:free`、`-:plus`、`-:pro`、`-:auto` 移除这些分组。保存配置或启动检查时发现不满足该约束，直接提示配置错误，避免 `auto` 被用户可用组过滤成不完整顺序。

API Key group 选择矩阵：

| 用户类型 | 可选择 token group | 说明 |
|---|---|---|
| `default` | `free`、`plus`、`pro`、`auto` | 用户自行选择池组，按实际调用池组倍率计费 |
| `vip` | `free`、`plus`、`pro`、`auto` | 用户自行选择池组，按实际调用池组倍率计费 |
| `svip` | `free`、`plus`、`pro`、`auto` | 用户自行选择池组，按实际调用池组倍率计费 |

当前业务决策允许用户自行选择 API Key group。只有 `auto` 执行 fallback；`free`、`plus`、`pro` 都是固定实体池组，不自动跨池。实际调用哪个池组，就按哪个池组倍率计费。

### 15.4 步骤二：API Key 绑定池组

使用现有 `tokens.group` 绑定资源池，不新增 API Key 类型。

| API Key 类型 | `tokens.group` | 实际使用池 |
|---|---|---|
| Free Key | `free` | free 渠道池 |
| Plus Key | `plus` | plus 渠道池 |
| Pro Key | `pro` | pro 渠道池 |
| Auto Key | `auto` | 复用 new-api 原生 auto，按 `AutoGroups = ["free", "plus", "pro"]` 选择渠道池 |

当前 `TokenAuth` 已经会做这些事情：

| 现有动作 | 对一卡通的价值 |
|---|---|
| 读取 token | 得到用户 API Key |
| 读取 `token.Group` | 得到该 Key 要走的实体资源池或 auto group |
| 校验 API Key group 合法性 | 允许 `free`、`plus`、`pro`、`auto`；实体池组校验 `GroupRatio`，`auto` 使用 new-api 原生逻辑跳过 `GroupRatio` |
| 写入 `ContextKeyUsingGroup` | 后续 distributor 用这个 group 选渠道 |
| 写入 `ContextKeyTokenGroup` | 日志、计费、模型列表可识别 token group |

所以这一步不需要重写鉴权，只需要确认配置和 UI 是否允许用户创建对应 group 的 token。

### 15.5 步骤三：channel/ability 一卡通路由

每个上游账号继续创建为一个普通 channel，关键是正确配置 `group`、`models`、`model_mapping`。

| 字段 | 配置方式 | 示例 |
|---|---|---|
| `channel.group` | 账号属于哪个池 | `free`、`plus`、`pro` |
| `channel.models` | 对用户暴露的真实模型名 | `gpt-5,gpt-5-codex,gemini-2.5-pro` |
| `model_mapping` | 仅处理 provider 特定上游名称差异 | 默认可为空 |
| `priority` | 同池优先级 | 默认 `0` |
| `weight` | 同优先级随机权重 | 默认 `0` 或按账号容量配置 |
| `tag` | 管理标签 | `codex-plus`、`claude-pro` |

创建或更新 channel 后，现有 `AddAbilities/UpdateAbilities` 会生成路由索引：

```text
channel.group + channel.models -> abilities(group, model, channel_id)
```

用户请求时，现有流程已经可以完成：

```text
API Key -> token.group -> request.model -> abilities(group, model) -> channel -> adaptor
```

这个路径不是“临时复用”，而是最短正式路径。因为如果重写路由系统，仍然要重新实现 group、model、channel、retry、quota、log 这些当前已经稳定存在的能力。

### 15.6 步骤四：批量导入账号

这是最值得新增的功能，因为 3000 个账号靠手点后台，手指会先申请离职。

新增导入能力，但落库仍然创建现有 channel。

| 导入能力 | 是否新增表 | 落库目标 |
|---|---:|---|
| Codex JSON 导入 | 否 | `channels` |
| OpenAI Key 导入 | 否 | `channels` |
| Claude Key 导入 | 否 | `channels` |
| Gemini Key 导入 | 否 | `channels` |
| 批量生成 abilities | 否 | 复用 `channel.AddAbilities` |

导入器只负责把不同账号格式转为统一的 `ChannelDraft`：

| 输入字段 | 转为 channel 字段 |
|---|---|
| `provider` | `type` |
| `pool` | `group` |
| `credential` | `key` |
| `models` | `models` |
| `upstream_model_mapping` | `model_mapping` |
| `tag` | `tag` |

### 15.7 步骤五：Codex 接口边界说明

Codex 渠道保持当前项目现状，不新增 `/v1/chat/completions` 到 Responses 的自动转换。用户如果选择 Codex 池，需要直接使用 `/v1/responses` 或 `/v1/responses/compact`。

固定产品规则：

| 规则 | 说明 |
|---|---|
| Codex 保持 Responses-only | 和当前项目行为一致 |
| UI/导入规则提示接口边界 | 明确提示 Codex 不支持 Chat Completions，减少用户误用 |
| 不做 Chat -> Responses 自动转换 | 避免增加主链路复杂度 |

接口能力提示链路：

```text
selected channel -> compatibility policy -> maybe convert request -> existing adaptor
```

### 15.8 步骤六：订阅和权益卡打通

当前项目已有订阅升级用户组的能力。一卡通中订阅主要提供可消费额度，不把用户类型和 API Key 资源池强绑定。

| 商品/权益 | 用户类型 | 可选 API Key group | 说明 |
|---|---|---|---|
| 永久余额 | 不强制改变 | `free`、`plus`、`pro`、`auto` | 用户可按预算选择池组，按实际调用池组倍率扣费 |
| 日卡/周卡/月卡 | 可选改变 | `free`、`plus`、`pro`、`auto` | 权益卡解决额度，不强制绑定单一池组 |

如果使用当前订阅计划的 `UpgradeGroup`，它只用于用户身份或运营标签，不作为 API Key group 的强限制。API Key group 仍由用户自行选择，账号池实际消耗按 `tokens.group -> 固定实体池或原生 auto -> using_group` 决定。

### 15.9 步骤七：池运营看板

池看板也尽量查现有表。

| 指标 | 数据来源 |
|---|---|
| 池总账号数 | `channels.group` |
| 可用账号数 | `channels.status` |
| 各 provider 数量 | `channels.type` |
| 池消耗 | `logs.group`、`channel.used_quota` |
| 单账号消耗 | `logs.channel_id`、`channels.used_quota` |
| 平均响应时间 | `channels.response_time` |
| 自动禁用数量 | `channels.status`、错误日志 |

### 15.10 步骤八：onecard 策略模块沉淀

onecard 模块不是重写主链路，而是把重复业务规则沉淀出来。

| onecard 子模块 | 职责 | 不做什么 |
|---|---|---|
| `pool` | 定义 free/plus/pro 池策略和默认配置 | 不替代 `token.group/channel.group` |
| `provider` | 用 `ProviderAdapter + BaseProvider + 子类` 标准化 Codex/OpenAI/Claude/Gemini 凭证和能力 | 不替代现有 adaptor，不在主流程写 provider if-else |
| `importer` | 只依赖 `ProviderAdapter` 父接口，批量导入并创建普通 channel | 不新增账号表，不直接解析各 provider 凭证 |
| `compat` | 封装 Codex Responses-only 能力说明和错误提示 | 不在 relay 主流程写满特判 |
| `health` | 基于现有字段计算健康状态 | 不重做日志系统 |

最终代码结构是“现有主流程为主，onecard 做增强”：

```text
TokenAuth(existing)
  -> Distribute(existing)
  -> ChannelSelect(existing abilities)
  -> onecard compat/import/strategy(optional)
  -> Adaptor(existing)
```

## 16. 永久余额卡、日卡、周卡、月卡售卖业务闭环设计

本节用于把一卡通资源池能力和商业化售卖闭环打通。目标是让用户可以购买“永久余额卡、日卡、周卡、月卡”，购买后获得可消费额度；用户可自行选择 `free`、`plus`、`pro` 实体资源池或 `auto` 原生自动池组，调用 API 时自动按实际调用池组倍率扣费，并在权益到期或额度用尽时返回明确错误。

核心原则：**永久余额卡走钱包余额，日卡/周卡/月卡走订阅权益卡；资源池仍然使用 `free`、`plus`、`pro`，不把用户身份组和 API Key 资源池组强行合并。**

### 16.1 业务目标

| 目标 | 说明 |
|---|---|
| 售卖永久余额 | 用户购买后增加永久余额，用完为止，不过期、不重置 |
| 售卖时效权益卡 | 用户购买日卡、周卡、月卡，获得有效期内的权益额度 |
| 自选资源池 | 用户可自行创建或使用 `free`、`plus`、`pro`、`auto` API Key group，按实际调用池组倍率扣费 |
| 统一扣费 | API 请求仍走当前 `BillingSession`，统一处理预扣、结算、退款 |
| 多卡可叠加 | 用户可以同时持有多张日卡、周卡、月卡 |
| 扣费可解释 | 多权益卡同时存在时，按明确顺序扣费，避免用户困惑 |
| 到期停止扣费 | 权益卡过期后自动失效；如果永久余额不足，请求直接报额度不足错误 |
| 低侵入实现 | 优先复用 `TopUp`、`SubscriptionPlan`、`UserSubscription`、`BillingSession`，不新建独立卡券系统 |

### 16.2 商品类型定义

| 商品类型 | 商品代码 | 数据承载 | 额度形态 | 是否过期 | 是否重置 | 典型用途 |
|---|---|---|---|---:|---:|---|
| 永久余额卡 | `wallet_balance` | `TopUp` + `users.quota` | 钱包余额 | 否 | 否 | 长期余额、权益卡不可用时回退扣费 |
| 日卡 | `day_card` | `SubscriptionPlan` + `UserSubscription` | 权益卡额度 | 是 | 24 小时周期 | 短期体验、试用 |
| 周卡 | `week_card` | `SubscriptionPlan` + `UserSubscription` | 权益卡额度 | 是 | 24 小时周期 | 中短期套餐 |
| 月卡 | `month_card` | `SubscriptionPlan` + `UserSubscription` | 权益卡额度 | 是 | 24 小时周期 | 主力订阅套餐 |

早期阶段不新增独立 `card_orders`、`cards`、`user_cards` 表。当前项目已有的表足够承载核心闭环；当运营复杂度超过现有模型表达能力时，再进入独立卡券系统阶段。

| 业务对象 | 复用表/模型 | 说明 |
|---|---|---|
| 永久余额订单 | `TopUp` | 记录充值订单和支付状态 |
| 永久余额账户 | `users.quota` | 记录用户永久可用余额 |
| 权益卡商品模板 | `SubscriptionPlan` | 记录日卡、周卡、月卡的价格、有效期、额度、池组权益 |
| 用户权益卡实例 | `UserSubscription` | 用户购买后生成的一张具体卡 |
| 订阅支付订单 | `SubscriptionOrder` | 记录权益卡购买订单 |
| 订阅预扣记录 | `SubscriptionPreConsumeRecord` | 记录单次请求扣了哪张权益卡，支持幂等退款 |
| 请求扣费会话 | `BillingSession` | 统一封装钱包和权益卡扣费 |

### 16.3 永久余额卡设计

永久余额卡本质是充值，不属于订阅权益卡。

| 设计项 | 最终方案 |
|---|---|
| 数据入口 | 复用现有充值入口 |
| 订单表 | `topups` |
| 到账字段 | `users.quota` |
| 到账方式 | 支付成功后调用 `IncreaseUserQuota` |
| 过期时间 | 无 |
| 重置规则 | 无 |
| 扣费来源 | `WalletFunding` |
| 是否影响资源池 | 默认不影响；如果运营需要，可后续增加充值赠送权益 |

永久余额购买闭环：

| 步骤 | 行为 | 代码承载 |
|---|---|---|
| 1 | 用户选择永久余额档位 | 钱包页金额选项 |
| 2 | 创建充值订单 | `TopUp` |
| 3 | 跳转支付 | `RequestEpay` / `RequestStripePay` / `RequestCreemPay` |
| 4 | 支付平台回调 | `EpayNotify` / Stripe webhook / Creem webhook |
| 5 | 验签和订单幂等 | `trade_no` + 订单锁 |
| 6 | 订单改为成功 | `TopUp.Status = success` |
| 7 | 增加用户余额 | `IncreaseUserQuota(userId, quotaToAdd)` |
| 8 | 记录充值日志 | `RecordTopupLog` |
| 9 | API 请求扣费 | `BillingSession` 选择 `WalletFunding` |

永久余额卡作为权益卡不可用时的回退资金来源。只有没有任何当前余额大于 0 的权益卡时，本次请求才会扣永久余额；永久余额也不足时返回额度不足错误。

### 16.4 日卡、周卡、月卡设计

日卡、周卡、月卡本质是有有效期的权益卡，统一使用 `SubscriptionPlan` 作为商品模板，用户购买成功后生成 `UserSubscription`。

| 卡类型 | `duration_unit` | `duration_value` | `quota_reset_period` | `quota_reset_custom_seconds` | `total_amount` 语义 | 默认展示池组 |
|---|---:|---:|---|---:|---|---|
| 日卡 | `day` | `1` | `custom` | `86400` | 每 24 小时周期额度 | `plus` 或 `pro` |
| 周卡 | `day` | `7` | `custom` | `86400` | 每 24 小时周期额度 | `plus` 或 `pro` |
| 月卡 | `month` | `1` | `custom` | `86400` | 每 24 小时周期额度 | `plus` 或 `pro` |

卡类商品统一采用“每 24 小时周期额度”模式。实现上直接复用 `new-api` 现有订阅重置能力：`quota_reset_period = custom`，`quota_reset_custom_seconds = 86400`。`total_amount` 在卡类商品里表示当前 24 小时周期内可用额度；周期结束后当前周期剩余额度作废，并重置下一份周期额度。

一卡通日卡、周卡、月卡必须配置 `total_amount > 0`，不支持 `total_amount <= 0` 的无限额度订阅卡。这样可以避免 `0` 同时表达“无限额度”和“没有额度”，计费口径保持简单。

管理员创建或更新日卡、周卡、月卡商品时必须校验 `total_amount > 0`；历史存量中不满足该条件的订阅计划禁止上架、禁止购买，并在后台展示配置错误。

24 小时周期规则如下：

| 卡类型 | 周期规则 |
|---|---|
| 日卡 | 自购买生效起 24 小时有效，只有 1 个周期 |
| 周卡 | 自购买生效起 7 个连续 24 小时周期，每个周期单独封顶 |
| 月卡 | 自购买生效起到下个自然月对应时间，每个 24 小时周期单独封顶 |
| 周期内未用完 | 剩余额度只在当前周期有效，跨周期作废 |
| 周期内提前用完 | 等下一个 24 小时周期开始后恢复下一份周期额度 |

注意：不要使用 `quota_reset_period = daily` 来实现日/周/月卡周期额度。`new-api` 当前 `daily` 是自然日 0 点重置；本方案需要的是从购买/生效时间开始滚动计算的 24 小时周期，因此必须使用 `custom + 86400`。

权益卡购买闭环：

| 步骤 | 行为 | 代码承载 |
|---|---|---|
| 1 | 管理员创建日/周/月卡商品 | `AdminCreateSubscriptionPlan` |
| 2 | 用户查看可购买卡 | `GetSubscriptionPlans` |
| 3 | 用户选择卡并发起支付 | `SubscriptionRequestEpay` / `SubscriptionRequestStripePay` / `SubscriptionRequestCreemPay` |
| 4 | 创建订阅订单 | `SubscriptionOrder` |
| 5 | 支付成功回调 | `SubscriptionEpayNotify` / Stripe webhook / Creem webhook |
| 6 | 完成订阅订单 | `CompleteSubscriptionOrder` |
| 7 | 创建用户权益卡 | `CreateUserSubscriptionFromPlanTx` |
| 8 | 如果配置了 `UpgradeGroup`，升级用户组 | `users.group = UpgradeGroup` |
| 9 | 写入升级前分组 | `PrevUserGroup` |
| 10 | API 请求按扣费规则消耗权益卡 | `SubscriptionFunding` |
| 11 | 到期或重置由定时任务维护 | `StartSubscriptionQuotaResetTask` |

### 16.5 free/plus/pro 池组和卡权益关系

用户身份组和 API Key 使用组是解耦的。`users.group` 表示用户身份或后台运营标签，`tokens.group` 表示某个 API Key 实际使用哪个资源池。

| 概念 | 固定命名 | 作用 |
|---|---|---|
| 用户身份组 | `default`、`vip`、`svip` | 表示用户身份、后台运营标签 |
| API Key 资源池组 | `free`、`plus`、`pro`、`auto` | 决定 API Key 实际走固定池还是原生 auto 自动池 |
| 渠道池组 | `free`、`plus`、`pro` | 表示上游账号属于哪个池 |
| 路由能力组 | `free`、`plus`、`pro` | `abilities(group, model)` 用于查候选渠道 |

API Key group 选择矩阵：

| 用户类型/权益状态 | 可选择的 API Key group | 说明 |
|---|---|---|
| `default` | `free`、`plus`、`pro`、`auto` | 用户自行选择池组，按实际调用池组倍率扣费 |
| `vip` | `free`、`plus`、`pro`、`auto` | 用户自行选择池组，按实际调用池组倍率扣费 |
| `svip` | `free`、`plus`、`pro`、`auto` | 用户自行选择池组，按实际调用池组倍率扣费 |
| 仅有永久余额 | `free`、`plus`、`pro`、`auto` | 余额足够即可选择更高倍率池组或 auto 自动池组 |

购买权益卡后不自动同步已有 API Key。购买权益卡只表示用户获得可消费额度，具体 API Key 使用哪个池组，由用户在创建或编辑 API Key 时自行选择。

| 场景 | 行为 |
|---|---|
| 用户购买权益卡 | 不修改已有 API Key group |
| 用户创建新 API Key | 用户自行选择 `free`、`plus`、`pro`、`auto` group |
| 用户编辑旧 API Key | 用户自行切换到 `free`、`plus`、`pro`、`auto` group |
| 权益过期 | 不批量修改 API Key；后续请求如果余额不足则按扣款顺序报错 |

### 16.6 扣费来源和扣款顺序

一卡通售卖闭环中，用户扣款顺序固定为：**先找一张当前余额大于 0 的权益卡；找到则本次请求完整扣这张权益卡并允许扣成负数；找不到可用权益卡时才扣永久余额；永久余额也不足时返回额度不足错误**。

| 顺序 | 扣费来源 | 行为 |
|---:|---|---|
| 1 | 权益卡 | 优先扣仍在有效期内、当前 24 小时周期未耗尽的日卡、周卡、月卡 |
| 2 | 永久余额 | 没有任何当前余额大于 0 的权益卡时，继续扣 `users.quota` |
| 3 | 报错 | 永久余额也不足时，返回额度不足错误 |

当前项目已有 `BillingPreference`，但一卡通扣款顺序由系统配置统一控制。新增 `SubscriptionFirstGroups` 配置项，用于维护“订阅权益卡优先支付”的 API Key group 列表；当请求的 `token.group` 命中该配置时，`BillingSession` 强制使用 `subscription_first`，否则继续按用户自己的 `BillingPreference` 执行。

| 配置项 | 类型 | 示例 | 行为 |
|---|---|---|---|
| `SubscriptionFirstGroups` | string array | `["free", "plus", "pro", "auto"]` | 命中列表的 API Key group 强制权益卡优先，未命中则使用用户偏好 |

实际请求扣费链路：

```text
API 请求
  -> TokenAuth 校验 API Key
  -> Distribute 选择渠道
  -> ModelPriceHelper 计算预扣额度
  -> PreConsumeBilling 创建 BillingSession
  -> 如果 token.group 命中 SubscriptionFirstGroups，则强制 subscription_first
  -> 优先选择一张当前余额大于 0 的 SubscriptionFunding，找不到再回退 WalletFunding
  -> 请求上游
  -> SettleBilling 按实际 usage 多退少补
  -> 失败则 Refund
```

扣费影响的数据：

| 扣费来源 | 影响字段 |
|---|---|
| 永久余额 | `users.quota` 减少 |
| 权益卡 | `user_subscriptions.amount_used` 增加 |
| API Key 额度 | `tokens.remain_quota` 减少 |
| 用户统计 | `users.used_quota`、`users.request_count` 增加 |
| 渠道统计 | `channels.used_quota` 增加 |
| 消费日志 | 记录 user、token、model、channel、group、billing source |

### 16.7 多权益卡扣费顺序

用户可能同时拥有多张日卡、周卡、月卡。必须定义清楚扣哪张，否则账单会像一团意大利面，还是没放盐的那种。

固定规则：**优先扣当前权益周期最早结束且当前余额大于 0 的卡；如果没有重置周期，则扣最早过期的卡。**

| 排序优先级 | 字段/计算值 | 说明 |
|---:|---|---|
| 1 | `current_period_end` | 当前权益周期结束时间，优先使用 `next_reset_time`，没有则用 `end_time` |
| 2 | `end_time` | 卡最终过期时间 |
| 3 | `id` | 稳定排序，避免随机跳动 |

| 卡配置 | `current_period_end` |
|---|---|
| `quota_reset_period = never` | `end_time` |
| `quota_reset_period != never` 且 `next_reset_time > 0` | `min(next_reset_time, end_time)` |
| `next_reset_time = 0` | `end_time` |

当前余额计算口径固定为：`current_remain = amount_total - amount_used`。选择权益卡时只判断 `current_remain > 0`，不要求 `current_remain >= 本次请求预扣额度`。

允许扣成负数后，代码口径必须同步调整：

| 代码点 | 当前逻辑 | 固定改法 |
|---|---|---|
| `PreConsumeUserSubscription` 选卡判断 | `remain < amount` 时跳过该卡 | 改为 `current_remain <= 0` 才跳过；`current_remain > 0` 时允许 `amount_used += amount` 超过 `amount_total` |
| `PostConsumeUserSubscriptionDelta` 正向补扣 | `newUsed > amount_total` 时报错 | 所有正向 `delta` 都允许 `amount_used > amount_total`，包括 `Settle` 和 `Reserve` 触发的订阅补扣；负向退款仍然保持 `amount_used` 不低于 0 |
| 退款幂等 | `SubscriptionPreConsumeRecord.request_id` 唯一 | 保持不变；一次请求只会产生一条权益卡预扣记录 |

示例：

| 用户持有卡 | 当前时间 | 当前周期结束 | 扣费顺序 |
|---|---|---|---|
| 日卡，明天 21:00 到期 | 2026-05-17 21:00 | 2026-05-18 21:00 | 第二 |
| 月卡，每天 23:00 重置 | 2026-05-17 21:00 | 2026-05-17 23:00 | 第一 |

解释：虽然日卡更快整体到期，但月卡当前 24 小时周期更快结束，所以先扣月卡，避免当天额度浪费。

与当前代码差异：

| 当前实现 | 固定优化 |
|---|---|
| `PreConsumeUserSubscription` 当前按 `end_time asc, id asc` 选择卡 | 改为事务内重置后按 `current_period_end asc, end_time asc, id asc` 排序 |
| 当前一次请求只扣一张权益卡 | 保留，一次请求只允许命中一张权益卡 |
| 当前权益卡不足会尝试下一张 | 改为只选择当前余额大于 0 的第一张卡；选中后允许该卡被扣成负数 |

实现顺序固定为：

| 步骤 | 行为 |
|---:|---|
| 1 | 在事务内使用 `FOR UPDATE` 查询用户所有 active 且未过期的权益卡 |
| 2 | 对每张候选卡先执行 `maybeResetUserSubscriptionWithPlanTx`，确保 `amount_used`、`next_reset_time` 是当前周期最新值 |
| 3 | 在内存中计算每张卡的 `current_period_end = min(next_reset_time, end_time)`；没有 `next_reset_time` 时使用 `end_time` |
| 4 | 按 `current_period_end asc, end_time asc, id asc` 排序 |
| 5 | 从排序后的权益卡中选择第一张当前余额大于 0 的卡 |
| 6 | 用这一张卡预扣本次请求的完整额度，并创建一条 `SubscriptionPreConsumeRecord` |
| 7 | 允许该权益卡被扣成负数；负数表示当前 24 小时周期额度已透支，后续请求不再选择这张卡，直到下个周期重置 |
| 8 | 如果没有任何当前余额大于 0 的权益卡，则不扣权益卡，直接回退永久余额 |

一次请求只使用一张权益卡，不做多卡组合扣费，也不做“权益卡 + 永久余额”混合扣费。这个规则能继续复用当前 `SubscriptionPreConsumeRecord.request_id` 唯一索引和 `BillingSession` 单资金来源模型，账务链路简单、幂等、好排查。

周期重置时继续执行 `amount_used = 0`，透支产生的负数状态自然消失；该卡重新参与下一周期的排序和选择。

### 16.8 权益到期、重置和回退

订阅维护任务继续复用当前实现。

| 场景 | 固定行为 |
|---|---|
| 权益卡到期 | `ExpireDueSubscriptions` 将状态改为 `expired` |
| 权益卡取消 | 管理员调用 `AdminInvalidateUserSubscription`，状态改为 `cancelled` |
| 权益卡删除 | 管理员调用 `AdminDeleteUserSubscription` |
| 额度重置 | `ResetDueSubscriptions` 将 `AmountUsed` 清零并计算下一次重置 |
| 用户组回退 | 如果使用 `UpgradeGroup` 做运营标签，可按现有逻辑从 `UpgradeGroup` 回退到 `PrevUserGroup` |

API Key group 回退不自动强制批量改旧 Key。Key 可以保留原 group；后续请求按照权益卡和永久余额扣费，余额不足时返回额度不足错误。

| 事件 | 固定处理 |
|---|---|
| plus/pro 权益过期 | 不修改 API Key group |
| 用户继续用 plus/pro Key | 继续按该池组或 fallback 后实际池组倍率计费 |
| 权益卡和永久余额都不足 | 返回额度不足错误 |
| 用户想降级使用 | 前端允许把 Key 切回 `free` |

### 16.9 前端产品闭环

钱包页固定拆成两个清晰区域。

| 区域 | 展示内容 | 对应数据 |
|---|---|---|
| 永久余额 | 当前余额、充值档位、充值记录 | `users.quota`、`TopUp` |
| 权益卡 | 日卡/周卡/月卡商品、我的权益卡、扣费顺序 | `SubscriptionPlan`、`UserSubscription`、onecard 计费策略 |

权益卡用户侧展示使用 `max(current_remain, 0)` 作为可用余额；当真实 `current_remain < 0` 时，展示“本周期已透支，等待下次重置”。后台、日志和排障页面保留真实负数，方便核账。

用户侧关键页面：

| 页面/组件 | 功能 |
|---|---|
| 钱包余额卡片 | 展示永久余额、充值入口 |
| 权益卡商品列表 | 展示日卡、周卡、月卡价格、额度、有效期、默认展示池组 |
| 我的权益卡 | 展示当前持有卡、剩余额度、到期时间、下一次重置时间 |
| 扣费顺序说明 | 显示固定扣费顺序和单卡扣费规则 |
| API Key 管理 | 展示每个 Key 的 group、池组倍率、fallback 后实际 using group |
| 一键切换池组 | 将某个 API Key 切换到用户选择的 `free`、`plus`、`pro` 池组 |

### 16.10 后台运营闭环

管理员需要能完成商品配置、订单查看、权益补发、问题排查。

| 后台能力 | 阶段性实现方式 |
|---|---|
| 创建永久余额档位 | 复用支付设置里的充值金额档位 |
| 创建日卡/周卡/月卡 | 复用订阅套餐管理 |
| 配置默认展示池组 | 使用 `UpgradeGroup` 或新增展示字段映射到 `free/plus/pro`，只做展示和引导，不限制用户选择 |
| 查看用户权益 | 复用用户订阅列表 |
| 手动补发权益卡 | `AdminBindSubscription` |
| 取消权益卡 | `AdminInvalidateUserSubscription` |
| 删除异常权益 | `AdminDeleteUserSubscription` |
| 查看充值记录 | `TopUp` 列表 |
| 查看订阅订单 | `SubscriptionOrder` 列表，必要时补后台页面 |
| 查看消费日志 | 现有日志，补充 billing source / subscription id 展示更佳 |

### 16.11 数据模型增强方案

早期阶段不强制新增字段，但完整闭环需要在商品形态稳定后给 `SubscriptionPlan` 增加轻量商品字段。

| 字段 | 类型 | 实施阶段 | 说明 |
|---|---|---:|---|
| `product_type` | string | 第三步 | `day_card`、`week_card`、`month_card` |
| `pool_group` | string | 第三步 | 商品默认展示的 API Key group，仅用于展示和引导，不做强限制 |
| `display_badge` | string | 第三步 | 前端展示标签 |
| `metadata` | text/json | 第三步 | 商品展示配置、权益说明 |

早期阶段可以先不改表，使用现有字段推导；第三步补齐商品字段后，以显式字段为准。

| 推导规则 | 卡类型 |
|---|---|
| `duration_unit = day` 且 `duration_value = 1` | 日卡 |
| `duration_unit = day` 且 `duration_value = 7` | 周卡 |
| `duration_unit = month` 且 `duration_value = 1` | 月卡 |
| `TopUp` 充值档位 | 永久余额卡 |

### 16.12 安全、幂等和风控

| 风险 | 处理方式 |
|---|---|
| 支付回调重复 | 订单状态 + `trade_no` 唯一 + 订单锁 |
| 权益重复创建 | `CompleteSubscriptionOrder` 事务内锁定订单，成功订单幂等返回 |
| 订阅重复扣费 | `SubscriptionPreConsumeRecord.request_id` 唯一 |
| 上游失败未退款 | `BillingSession.Refund` 统一退款 |
| 钱包多退 | 钱包退款不做盲目重试 |
| 订阅多退 | 通过 `request_id` 幂等退款 |
| 混合退款 | 一次请求只有一个 `FundingSource`，退款只处理本次命中的权益卡或永久余额 |
| 用户误选高倍率池 | 前端明确展示池组倍率，API 日志记录 `requested_group` 和 `using_group` |
| 权益到期仍使用高级 Key | 不拦截池组；按固定扣款顺序扣费，余额不足时报错 |
| 用户滥用试用日卡 | `MaxPurchasePerUser` 限制购买次数 |

### 16.13 完整功能范围与实施顺序

本方案目标是完整实现永久余额卡、日卡、周卡、月卡的售卖、支付、生效、扣费、退款、到期、回退、前端展示和后台运营闭环。实施时可以分阶段推进，但分阶段只是交付顺序，不代表后续能力不做。

| 功能 | 最终是否实现 | 实施阶段 | 说明 |
|---|---:|---|---|
| 永久余额卡 | 是 | 第一步 | 复用现有充值 |
| 日卡/周卡/月卡 | 是 | 第一步 | 复用订阅计划 |
| 权益卡购买支付 | 是 | 第一步 | 复用现有订阅支付 |
| 多权益卡扣费顺序优化 | 是 | 第一步 | 改为当前周期最早结束优先 |
| API Key group 合法性校验 | 是 | 第一步 | 限制为 `free`、`plus`、`pro`、`auto`，并校验 auto 访问顺序配置 |
| 前端权益卡展示 | 是 | 第二步 | 钱包页展示商品和我的权益 |
| 运营后台补发/取消 | 是 | 第二步 | 复用现有订阅管理并补足展示 |
| API Key group 自助选择 | 是 | 第二步 | 用户自行选择 `free`、`plus`、`pro`、`auto` 池组 |
| `SubscriptionPlan.product_type` | 是 | 第三步 | 商品形态稳定后补字段，提升运营可读性 |
| `SubscriptionPlan.pool_group` | 是 | 第三步 | 明确权益卡商品的默认展示池组 |
| 商品展示 `metadata` | 是 | 第三步 | 支持标签、卖点、权益说明 |
| 订阅订单后台列表 | 是 | 第三步 | 方便客服和财务排查 |
| 日/周/月卡统计看板 | 是 | 第三步 | 统计销量、收入、消耗、留存 |
| 独立卡券系统 | 视复杂度决定 | 第四步 | 只有现有模型表达不了时再新建 |

### 16.14 最终业务闭环

完整闭环如下：

```text
管理员配置商品
  -> 用户购买永久余额卡或权益卡
  -> 支付成功
  -> 永久余额进入 users.quota / 权益卡生成 UserSubscription
  -> 用户创建或切换 API Key group
  -> API 请求按 token.group 路由到实体池或原生 auto 自动池组
  -> BillingSession 按 SubscriptionFirstGroups 配置决定是否强制 subscription_first
  -> 请求成功后结算，多退少补
  -> 请求失败后退款
  -> 日志记录消费、渠道、资源池、扣费来源
  -> 权益到期/重置任务维护状态
  -> 用户续费、充值或切换 API Key group
```

| 业务对象 | 最终方案 |
|---|---|
| 永久余额卡 | 复用 `TopUp + users.quota`，作为永久钱包余额 |
| 日卡/周卡/月卡 | 复用 `SubscriptionPlan + UserSubscription`，作为时效权益卡 |
| API 请求扣费 | 复用 `BillingSession`，命中后台配置的 `SubscriptionFirstGroups` 时强制 `subscription_first` |
| 池组路由 | 继续使用 `free/plus/pro` 实体池，并用 `auto` 表达原生自动池组路由 |
| 多权益卡选择 | 按当前权益周期最早结束优先；一次请求只扣一张余额大于 0 的权益卡，并允许该卡被扣成负数 |
| 实施策略 | 分阶段交付完整闭环，优先复用现有支付、订阅、计费、日志能力 |

## 17. 分阶段实施顺序

以下阶段用于安排开发顺序。最终目标是完整实现第 16 节定义的业务闭环，不把后续阶段理解为可永久省略的范围。

| 阶段 | 目标 | 交付内容 |
|---|---|---|
| 第一步：打通核心闭环 | 让用户可以购买并使用永久余额卡、日卡、周卡、月卡 | 复用 `TopUp`、`SubscriptionPlan`、`UserSubscription`、`BillingSession`；配置 free/plus/pro；实现多权益卡排序和单卡扣费；增加 API Key group 合法性校验 |
| 第二步：补齐产品体验 | 让用户看得懂、切得动、用得明白 | 钱包页拆分永久余额和权益卡；展示我的权益；展示固定扣费顺序；提供 API Key group 自助选择和倍率提示 |
| 第三步：补齐运营后台 | 让管理员能配置、排查、补发、统计 | 补充订阅订单列表；完善商品类型展示；补发/取消/删除权益；增加卡销量、收入、消耗统计 |
| 第四步：增强精细能力 | 处理规模化运营 | 商品字段 `product_type/pool_group/metadata`；池健康和成本优化；批量导入更多 provider |

分阶段依赖关系：

| 能力 | 依赖 | 说明 |
|---|---|---|
| 永久余额卡售卖 | 现有充值系统 | 可最先落地 |
| 日/周/月卡售卖 | 现有订阅系统 | 可与永久余额卡并行 |
| 扣费闭环 | `BillingSession` | 必须在用户正式使用前完成 |
| 池组路由 | `tokens.group`、`channels.group`、`abilities` | 必须和扣费闭环一起完成 |
| 前端展示 | 后端商品和权益 API | 后端规则稳定后实现 |
| 运营统计 | 支付订单、消费日志、订阅记录 | 依赖真实数据沉淀 |

## 18. 已确认决策

本节记录已经确认的业务决策。总体原则是：**简单、好用、少侵入、可解释**。

### 18.1 已确认决策

| 问题 | 决策 |
|---|---|
| 任意用户类型是否可选择任意资源池 | 允许。用户类型继续使用 `{ "default": 1, "svip": 1, "vip": 1 }`；账号池组继续使用 `free`、`plus`、`pro`；不同用户创建 API Key 时可自行选择任意资源池，因为不同池组收费倍率不同 |
| 用户请求模型名是否使用真实模型名 | 使用真实模型名。管理端和用户端展示的模型名称完全一致 |
| Codex 是否必须兼容 `/v1/chat/completions` | 不兼容，保持现状。Codex 仍只支持 `/v1/responses`、`/v1/responses/compact` 等当前已支持入口 |
| 账号凭证来源格式 | 采用主流 sub2api 账号集合格式，核心为 `accounts[].credentials`；导入时优先要求 `accounts[].credentials.access_token`，账号 ID 可从 `chatgpt_account_id`、`account_id` 或 access token 中解析，`refresh_token` 作为可选增强字段 |
| 是否需要按用户独享账号 | 不需要。所有用户共用 `free`、`plus`、`pro` 三个账号池 |
| 订阅超额后如何处理 | 扣款顺序固定为：先找一张当前余额大于 0 的权益卡；找到则本次请求完整扣这张卡并允许负数；找不到可用权益卡时才扣永久余额，永久余额不足则报错 |
| 购买权益卡后是否自动同步已有 API Key group | 不自动同步。购买权益卡只表示用户获得可消费额度，具体 API Key 使用哪个池组由用户自己选择 |
| 是否启用 24 小时周期重置型月卡 | 启用。使用 `new-api` 现有 `custom` 模式实现：`quota_reset_period = custom`，`quota_reset_custom_seconds = 86400`；不是自然日 0 点重置 |
| 是否允许跨池 fallback | 允许，但只在用户选择 `auto` 时执行：`free -> plus -> pro`；实际调用哪个池组，就按哪个池组倍率计费 |
| 原有 `default` 分组如何处理 | 与一卡通 `free` 池无关，保持旧逻辑即可 |
| 一卡通模块放 `pkg` 还是 `service` | 做成独立模块，尽量避免侵入 `new-api` 现有代码，降低后续从主分支合代码的冲突风险 |
| 后续是否需要新增 onecard 专属表 | 如果现有 `new-api` 功能体系已经可以支持一卡通业务，就不新增；只有表达不了完整业务闭环时再评估 |

sub2api 导入格式：

参考依据：sub2api 账号模型使用 `accounts` 作为账号集合，账号凭证集中放在 `credentials` JSON 中；OAuth 类型凭证通常包含 `access_token`、`refresh_token`、`expires_at` 等字段。公开代码参考：`https://github.com/Wei-Shaw/sub2api/blob/main/backend/ent/schema/account.go`。

```json
{
  "type": "sub2api-data",
  "accounts": [
    {
      "name": "user@example.com",
      "type": "openai",
      "credentials": {
        "access_token": "xxx",
        "refresh_token": "xxx",
        "chatgpt_account_id": "e9360506-6ad4-4753-b44f-f9929a276fbb",
        "expires_at": 1776737274
      },
      "extra": {
        "email": "user@example.com"
      }
    }
  ]
}
```

| 字段 | 要求 | 说明 |
|---|---|---|
| `accounts` | 必填 | 账号集合数组 |
| `accounts[].credentials.access_token` | 必填 | 当前请求 Codex/OpenAI 账号能力的核心凭证 |
| `accounts[].credentials.refresh_token` | 可选增强 | 用于自动续签；缺失时可导入但账号长期稳定性下降 |
| `accounts[].credentials.chatgpt_account_id` | 可提供 | 优先作为 ChatGPT/Codex account id |
| `accounts[].credentials.account_id` | 可选 | `chatgpt_account_id` 为空时可作为回退 |
| `accounts[].name` / `accounts[].extra.email` / `accounts[].credentials.email` | 可提供 | 用于后台识别账号 |

### 18.2 跨池 fallback 决策

跨池 fallback 指的是：当用户的 API Key 选择 `auto` group 时，系统复用 new-api 原生自动分组能力，并按 `AutoGroups = ["free", "plus", "pro"]` 顺序查找可用账号池。

`free`、`plus`、`pro` 是固定实体池组，不自动跨池。只有 `auto` 执行 fallback，并且只向更高倍率池组升级，不做降级 fallback。

| API Key group | 资源池选择行为 | 是否允许 |
|---|---|---|
| `free` | 只使用 free 池 | 是 |
| `plus` | 只使用 plus 池 | 是 |
| `pro` | 只使用 pro 池 | 是 |
| `auto` | free 池不可用时尝试 plus，plus 也不可用时尝试 pro | 是 |
| `plus -> free` | plus 池不可用时降级到 free 池 | 否 |
| `pro -> plus` | pro 池不可用时降级到 plus 池 | 否 |

计费规则：**按照实际调用成功的池组计费**。

| 用户 API Key group | 实际调用池组 | 计费倍率 |
|---|---|---|
| `free` | `free` | 按 `free` 分组倍率 |
| `plus` | `plus` | 按 `plus` 分组倍率 |
| `pro` | `pro` | 按 `pro` 分组倍率 |
| `auto` | `free` | 按 `free` 分组倍率 |
| `auto` | `plus` | 按 `plus` 分组倍率 |
| `auto` | `pro` | 按 `pro` 分组倍率 |

触发条件固定保持简单：**只有 API Key group 为 `auto`，并且当前顺序池组没有可用账号时触发 fallback**。不要因为单个请求报错就立刻跨池，否则重试、计费和用户感知都会变复杂。

| 当前池组状态 | 是否 fallback |
|---|---:|
| `auto` 当前顺序池组没有任何可用 channel | 是 |
| `auto` 当前顺序池组该模型没有可用 channel | 是 |
| 当前池组 channel 临时请求失败，但池内仍有其他可用 channel | 否，先走池内重试 |
| 固定实体池组 `free`、`plus`、`pro` 没有可用账号 | 否，返回明确错误 |
| `auto` 所有顺序池组都没有可用账号 | 否，返回明确错误 |

日志必须记录两个 group：

| 字段 | 含义 |
|---|---|
| `requested_group` | 用户 API Key 原始选择的池组 |
| `using_group` | 实际调用并计费的池组 |

## 19. 最终方案

| 设计项 | 最终决策 |
|---|---|
| 用户侧 | 一个 API Key |
| 主协议 | OpenAI-compatible `/v1/chat/completions` |
| 功能形态 | 保留现有主流程，`onecard` 作为增强模块；Facade 单点接入 |
| 数据复用 | 不新建账号池系统，复用 `tokens`、`channels`、`groups`、`abilities`、`model_mapping` |
| 面向对象方式 | interface 定义父能力，base struct 复用公共逻辑，子类策略多态实现 |
| 内部路由 | `token.group + model -> abilities -> channel` |
| 资源池 | `free` / `plus` / `pro` |
| `default` 分组处理 | 与 `free` 无关，不迁移、不映射、不作为别名 |
| 渠道池 | 管理员按运营需要把 GPT/Claude/Gemini 等账号放入不同资源池 |
| 用户类型 | 继续使用 `{ "default": 1, "svip": 1, "vip": 1 }` |
| 模型名 | 使用真实模型名，管理端和用户端展示完全一致 |
| Codex | 保持现状，不新增 Chat Completions 兼容；Codex 只走 Responses 相关入口 |
| 账号存储 | 一个上游账号一个 channel |
| 账号池使用 | 所有用户共用 `free`、`plus`、`pro` 三个账号池 |
| API Key group | 任意用户类型都可自行选择 `free`、`plus`、`pro`、`auto`，不按用户类型限制 |
| 购买权益后 API Key | 不自动修改已有 API Key group |
| 售卖闭环 | 永久余额卡走 `TopUp`，日卡/周卡/月卡走 `SubscriptionPlan` |
| 计费 | 按真实模型名和实际调用池组 `using_group` 倍率计费 |
| 跨池 fallback | 用户选择 `auto` 时执行 `free -> plus -> pro`，按照实际调用池组倍率计费 |
| 扣款顺序 | 先找一张当前余额大于 0 的权益卡；找到则本次请求完整扣这张卡并允许负数；找不到可用权益卡时才扣永久余额，永久余额不足时报错 |
| 多权益卡选择 | 当前权益周期最早结束优先；一次请求只扣一张当前余额大于 0 的权益卡，允许该卡被扣成负数 |
| 卡周期重置 | 日卡、周卡、月卡使用 `custom + 86400`，按购买/生效时间起每 24 小时为周期重置 |
| 模块位置 | onecard 做成独立模块，尽量少侵入 `new-api` 主流程 |
| 专属表 | 现有体系可支持时不新增 onecard 专属表 |
| 运维 | 池健康、失败率、自动禁用、用量看板 |

该方案最大优点是：高度复用当前 new-api 现有架构，同时把新增逻辑封装在独立 onecard 模块内。老代码只增加少量 hook，业务差异通过 free/plus/pro 池组策略、provider 子类和兼容策略子类多态实现，不需要推翻现有渠道调度和计费体系，后续从主干合代码也更轻松。
