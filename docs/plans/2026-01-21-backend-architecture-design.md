# Go 项目结构分析平台 - 后端架构设计

> 创建日期：2026-01-21
> 状态：已批准

## 1. 项目概述

基于 Go 的 AI 项目结构分析平台后端服务，提供用户认证、分析任务调度、实时进度推送、社区功能等。

## 2. 技术选型

| 项目 | 选择 |
|------|------|
| Module | `github.com/qs3c/anal_go_server` |
| 框架 | Gin + GORM |
| 数据库 | MySQL 8.0 |
| 缓存/队列 | Redis 7.0 |
| 存储 | 阿里云 OSS |
| 分析库 | anal_go_agent (go.mod replace 本地路径) |
| 进程通信 | Redis Pub/Sub |

## 3. 项目结构

```
anal_go_server/
├── cmd/
│   ├── server/main.go          # API 服务入口
│   └── worker/main.go          # Worker 服务入口
├── internal/
│   ├── api/
│   │   ├── handler/            # HTTP 处理器
│   │   ├── middleware/         # 中间件
│   │   └── router.go           # 路由配置
│   ├── service/                # 业务逻辑层
│   ├── repository/             # 数据访问层
│   ├── model/                  # 数据模型 + DTO
│   └── pkg/                    # 内部工具包
│       ├── jwt/
│       ├── oss/
│       ├── ws/
│       ├── queue/
│       └── response/           # 统一响应封装
├── config/
│   └── config.go               # 配置管理
├── migrations/                 # SQL 迁移文件
├── config.yaml                 # 配置文件
├── .env.example
├── Makefile
├── go.mod
└── go.sum
```

## 4. 核心架构

### 4.1 双进程架构

```
┌─────────────────┐         ┌─────────────────┐
│   API Server    │         │     Worker      │
│   (cmd/server)  │         │   (cmd/worker)  │
└────────┬────────┘         └────────┬────────┘
         │                           │
         │  1. 创建任务              │
         │  2. 写入 DB               │
         │  3. 推入 Redis 队列 ──────►│
         │                           │  4. 从队列取任务
         │                           │  5. Clone 仓库
         │                           │  6. 调用 anal_go_agent
         │                           │  7. 上传结果到 OSS
         │  ◄─── WebSocket 推送 ─────│  8. 更新 DB + 推送
         │                           │
```

### 4.2 WebSocket 跨进程通信（Redis Pub/Sub）

```
┌────────┐  Publish   ┌─────────┐  Subscribe  ┌────────────┐
│ Worker │ ─────────► │  Redis  │ ──────────► │ API Server │
└────────┘            └─────────┘             └─────┬──────┘
                                                    │ WebSocket
                                                    ▼
                                              ┌──────────┐
                                              │ Frontend │
                                              └──────────┘
```

## 5. 数据模型设计

### 5.1 统一响应格式

```go
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
}
```

### 5.2 错误码

| 错误码 | 含义 |
|--------|------|
| 0 | 成功 |
| 1000 | 参数错误 |
| 1001 | 认证失败 |
| 1002 | 权限不足 |
| 1003 | 资源不存在 |
| 1004 | 配额不足 |
| 1005 | 重复操作 |
| 5000 | 服务器错误 |

### 5.3 JSON 字段命名

所有 JSON 字段使用 snake_case（遵循前后端对接规范）。

## 6. anal_go_agent 集成

### 6.1 go.mod 配置

```go
module github.com/qs3c/anal_go_server

go 1.22

require (
    github.com/user/go-struct-analyzer v0.0.0
)

replace github.com/user/go-struct-analyzer => /Users/albert/Desktop/fromGithub/anal_go_agent
```

### 6.2 调用方式

```go
func (s *AnalyzerService) RunAnalysis(job *model.AnalysisJob, onProgress func(step string)) (*AnalysisResult, error) {
    a, err := analyzer.New(analyzer.Options{
        ProjectPath: job.TempDir,
        StartStruct: job.StartStruct,
        MaxDepth:    job.Depth,
        LLMProvider: s.getLLMProvider(job.ModelName),
        APIKey:      s.getAPIKey(job.ModelName),
        LLMModel:    job.ModelName,
        EnableCache: true,
    })

    onProgress("正在解析项目结构")
    result, err := a.Analyze()

    onProgress("正在生成可视化数据")
    visualizerJSON, err := a.GenerateVisualizerJSON()

    return &AnalysisResult{
        VisualizerJSON: []byte(visualizerJSON),
        TotalStructs:   result.TotalStructs,
        TotalDeps:      result.TotalDeps,
    }, nil
}
```

## 7. 开发阶段

### Phase 1：基础架构
- 项目初始化（go mod, 目录结构, 配置管理）
- 数据库（迁移文件, GORM 模型, Repository）
- 基础工具（响应封装, JWT, bcrypt）
- 认证系统（注册/登录/邮箱验证, GitHub OAuth, 中间件）

### Phase 2：核心功能
- Redis 队列（Push/Pop, Pub/Sub）
- OSS 客户端（框图上传, 头像上传）
- WebSocket Hub（连接管理, Redis 订阅）
- Worker 进程（队列消费, anal_go_agent 集成, 进度推送）
- 分析 API（CRUD, 分享, 任务状态）

### Phase 3：社区功能
- 广场 API
- 点赞/收藏
- 评论系统
- 配额管理
- 模型列表 API

### Phase 4：测试与完善
- 单元测试
- 集成测试
- Dockerfile
- docker-compose

## 8. 参考文档

- `backend-complete-guide.md` - 完整需求规范
- `integration-specification.md` - 前后端对接规范
