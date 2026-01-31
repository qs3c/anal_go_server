# Worker 核心逻辑实现设计

> 创建日期：2026-01-27
> 状态：已批准

## 概述

实现 Worker 核心逻辑，集成 anal_go_agent 完成实际的 Go 项目结构分析。

## 设计决策

| 决策点 | 方案 |
|--------|------|
| 克隆策略 | 临时目录 + 浅克隆 (`git clone --depth 1`) |
| 进度推送 | 阶段式，映射到百分比 (0-100%) |
| LLM Key | 从 config.models 获取 |
| 跨进程通信 | Redis Pub/Sub |

## 处理流程

```
1. 从 Redis 队列获取任务 (JobMessage)
2. 更新状态为 processing，推送进度 0%
3. 浅克隆仓库到临时目录，推送进度 20%
4. 创建 anal_go_agent 分析器，解析项目，推送进度 40%
5. 执行 AI 分析，推送进度 60%
6. 生成 VisualizerJSON，上传到 OSS，推送进度 80%
7. 更新数据库（Analysis.diagram_url），推送进度 100%
8. 清理临时目录
```

## 进度阶段

| Step | Progress | Message |
|------|----------|---------|
| cloning | 20 | 正在克隆仓库 |
| parsing | 40 | 正在解析项目结构 |
| analyzing | 60 | 正在进行 AI 分析 |
| uploading | 80 | 正在上传结果 |
| done | 100 | 分析完成 |

## 新增文件

```
internal/
├── pkg/
│   └── pubsub/
│       └── pubsub.go          # Redis Pub/Sub 封装
├── worker/
│   ├── processor.go           # 任务处理核心逻辑
│   └── git.go                 # Git 克隆操作
```

## 消息结构

```go
type ProgressMessage struct {
    Type       string `json:"type"`        // "job_progress"
    UserID     int64  `json:"user_id"`
    AnalysisID int64  `json:"analysis_id"`
    JobID      int64  `json:"job_id"`
    Status     string `json:"status"`      // processing, completed, failed
    Step       string `json:"step"`        // cloning, parsing, analyzing, uploading, done
    Progress   int    `json:"progress"`    // 0-100
    Message    string `json:"message"`     // 描述信息
    Error      string `json:"error"`       // 失败时的错误信息
}
```

## 依赖

go.mod 需要添加：
```go
replace github.com/user/go-struct-analyzer => /Users/albert/Desktop/fromGithub/anal_go_agent
```
