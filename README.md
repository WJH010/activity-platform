# Activity Platform

基于 Go + Gin 框架开发的智能活动管理平台，采用分层架构设计，提供活动发布报名、实时群聊、消息通知、AI Agent 智能助手等核心能力，支撑高并发活动场景与长连接稳定通信。

## 技术栈

| 类别 | 技术 |
|------|------|
| 语言 | Go 1.24 |
| Web 框架 | Gin |
| ORM | GORM |
| 数据库 | MySQL |
| 缓存 | Redis、本地内存缓存（singleflight 防击穿） |
| 对象存储 | MinIO |
| 实时通信 | WebSocket（gorilla/websocket） |
| 向量数据库 | Milvus 2.6 |
| 向量模型 | BGE-M3（1024 维） |
| 日志 | Logrus + 日志轮转 |

## 项目结构

```
activity-platform/
├── cmd/
│   └── main.go                          # 应用入口
├── internal/
│   ├── agent/                            # AI Agent 模块
│   │   ├── controller/                   #   对话、会话、Skill、LLM配置控制器
│   │   ├── engine/                       #   ReAct 引擎（流式/非流式循环）
│   │   ├── llm/                          #   LLM Provider（OpenAI 兼容协议）
│   │   ├── model/                        #   数据模型
│   │   ├── rag/                          #   RAG 语义检索
│   │   │   ├── embedding/                #     BGE-M3 向量化服务
│   │   │   ├── milvus/                   #     Milvus 客户端 & Collection 管理
│   │   │   ├── pipeline/                 #     数据同步（全量/增量）& 文本处理
│   │   │   └── search/                   #     混合检索服务（Dense + BM25 + RRF）
│   │   ├── repository/                   #   数据访问层
│   │   ├── service/                      #   业务逻辑层
│   │   └── skill/                        #   Skill 系统
│   │       ├── builtin/                  #     内置 Skill（活动/文章/用户/语义搜索）
│   │       ├── dynamic_executor.go       #     动态 Skill（HTTP 调用）
│   │       └── registry.go               #     Skill 注册中心
│   ├── article/                          # 文章模块
│   ├── cache/                            # 通用本地缓存（泛型 + singleflight）
│   ├── chat/                             # 实时群聊模块（WebSocket Hub）
│   ├── config/                           # 配置加载
│   ├── database/                         # 数据库初始化
│   ├── event/                            # 活动模块
│   │   ├── controller/
│   │   ├── dto/
│   │   ├── model/
│   │   ├── repository/
│   │   ├── service/
│   │   └── stock/                        #   库存服务（Redis Lua 原子预扣减）
│   ├── file/                             # 文件上传模块（MinIO）
│   ├── message/                          # 站内消息模块
│   ├── middleware/                       # 中间件（JWT认证、角色权限、日志、Recovery、RequestId）
│   ├── notice/                           # 公告模块
│   ├── redis/                            # Redis 客户端
│   ├── routes/                           # 统一路由注册
│   ├── user/                             # 用户模块（微信登录、JWT 双 Token）
│   └── utils/                            # 工具包（统一响应、错误码、日志、验证器）
├── go.mod
└── go.sum
```

## 核心功能

### 活动管理

- 活动创建、编辑、删除（管理员）
- 活动列表分页查询，支持状态筛选与标题搜索
- 活动报名 / 取消报名，自动加入活动聊天群组
- 自定义报名信息字段（姓名、手机号、邮箱等）

### 实时群聊

- 基于 WebSocket + Hub 模式的实时群聊
- 心跳保活（Ping/Pong）与连接超时检测
- 个人通知频道，支持站内消息实时推送
- 群组管理（创建、加入、移除成员）

### AI Agent 智能助手

- **ReAct 引擎**：实现思考 → 工具调用 → 观察 → 继续的推理循环，支持流式响应与多轮对话
- **Skill 系统**：内置 10 个 Skill，支持动态扩展
  - 活动查询 / 详情 / 报名 / 取消报名
  - 文章查询 / 详情 / 创建
  - 用户信息 / 已报名活动
  - 语义搜索
- **LLM 集成**：OpenAI 兼容协议，支持多 Provider 配置与用户偏好切换
- **会话管理**：多会话支持，自动生成会话标题，历史消息持久化

### RAG 语义检索

- **向量化**：BGE-M3 模型（1024 维），通过 OpenAI 兼容 Embedding API 调用
- **混合检索**：稠密向量（HNSW + COSINE）+ BM25 稀疏检索，RRF 融合排序
- **数据同步**：MySQL → Milvus 全量 / 增量同步，定时增量同步，HTML 清洗与智能分块
- **检索范围**：支持文章、活动、全部三种检索范围

### 认证与权限

- 微信小程序登录（双 Token 机制：Access Token + Refresh Token）
- JWT + Redis Token 黑名单，支持令牌主动失效
- 角色权限控制（超级管理员 / 管理员 / 普通用户）

## 关键设计

### 活动报名并发安全

采用 **Redis Lua 脚本原子预扣减 + DB 条件更新** 的双层保障策略：

```
请求 → Redis Lua 原子预扣减（快速拒绝已满场景）
  ↓ 预扣成功
  → DB 事务（业务校验 + 创建报名记录 + 条件更新计数）
    ↓ 事务失败
    → Redis 库存回补
```

- **Redis 层**：Lua 脚本保证"读取-判断-扣减"原子性，快速拦截超卖请求
- **DB 层**：`WHERE current_registrants < max_registrants` 条件更新作为最终一致性兜底
- **降级策略**：Redis 不可用时自动降级为纯 DB 行锁模式，保障服务可用性
- **本地缓存**：singleflight 防缓存击穿，减少 DB 查询压力

### k6 压测结果

测试场景：100 个活动名额，900 并发用户

| 指标 | 值 |
|------|-----|
| 总请求数 | 62,881 |
| 请求吞吐量 | 3,133.6 req/s |
| 检查通过率 | 100%（62,881/62,881） |
| 期望响应平均延迟 | 9.08 ms |
| 期望响应 P95 延迟 | 13.62 ms |
| 期望响应 P90 延迟 | 11.64 ms |
| 迭代平均耗时 | 215.88 ms |
| 运行时长 | 20.1 s |

> 压测阶段：5s 爬升至 900 VU → 10s 持续 900 VU → 5s 降至 0。100 个名额被正确扣减，其余请求均返回"报名人数已满"，零超卖。

### AI Agent ReAct 引擎

```
用户消息 → 系统提示词 + 历史消息 → LLM
                                    ↓
                              推理 + 工具调用
                                    ↓
                         Skill Registry 执行工具
                                    ↓
                            工具结果 → LLM 继续推理
                                    ↓
                              最终回复（流式输出）
```

- 最大 ReAct 轮数可配置（默认 10 轮）
- 工具结果 8K 字符硬截断，防止上下文溢出
- 支持推理模型（Reasoning Content）的思考过程保留与回传

### RAG 数据同步流水线

```
MySQL（文章/活动）
  → TextProcessor（HTML 清洗 → 段落优先分块 → 句子边界兜底）
    → BGE-M3 Embedding API（批量向量化）
      → Milvus（Upsert，自动生成 BM25 稀疏向量）
```

- 全量同步：首次启动自动触发
- 增量同步：定时轮询 `updated_at` 字段，仅同步变更数据
- 同步状态持久化到 DB，支持查看同步进度

## 快速开始

### 环境依赖

- Go 1.24+
- MySQL 8.0+
- Redis 7.0+
- MinIO
- Milvus 2.6+（可选，不启动则语义检索功能降级）
- BGE-M3 Embedding 服务（可选，如 Infinity / TEI / Ollama）

### 配置

创建配置文件：
config.yaml
docker-compose.infra.yaml
配置项参考：
config-demo.yaml
docker-compose-demo.infra.yaml


### 启动

```bash
go run cmd/main.go
```