# Go 项目结构可视化分析平台 - 后端开发完整指南

> 本文档为后端开发的完整需求和技术规范

## ⚠️ 重要提醒

**在开始开发前，请务必先阅读《前后端对接规范文档》(integration-specification.md)！**

该文档定义了：
- 🔴 字段命名规范（snake_case）
- 🔴 日期时间格式（RFC3339）
- 🔴 枚举值定义（必须与前端一致）
- 🔴 错误码定义（必须与前端一致）
- 🔴 WebSocket 消息格式
- 🔴 API 接口契约

不遵守对接规范将导致前后端无法对接！

---

## 目录
- [一、项目概述](#一项目概述)
- [二、系统架构](#二系统架构)
- [三、数据库设计](#三数据库设计)
- [四、API 接口定义](#四api-接口定义)
- [五、核心业务逻辑](#五核心业务逻辑)
- [六、配置管理](#六配置管理)
- [七、开发任务](#七开发任务)
- [八、开发规范](#八开发规范)

---

## 一、项目概述

### 1.1 项目背景

为 Go 开发者提供一个基于 AI 的项目结构分析和可视化平台的后端服务。

**核心功能：**
- 用户认证与授权（邮箱密码 + GitHub OAuth + 微信 OAuth）
- AI 自动分析 Go 项目结构
- 分析任务调度与执行
- 实时进度推送（WebSocket）
- 数据存储与管理
- 社区功能（分享、点赞、评论）
- 配额管理与订阅

### 1.2 核心依赖

**已完成的模块：**
- `anal_go_agent/pkg`: Go 库，用于分析 Go 项目结构体依赖关系
- 数据格式: `visualizer_output.json`（与前端 struct_element 项目约定）

**关键特性：**
- 支持两种场景：公开仓库 + 本地项目
- 起始结构体由用户输入
- 模型用户自选（受订阅级别限制）
- 用户可以手动创建和编辑 AI 生成的框图
- 一个用户可以有多个分析项目
- 提供自动保存功能
- 评论支持一级回复，纯文本
- 进度信息展示当前步骤和已耗时（无百分比）
- 任务关闭页面后继续执行，重新打开能看到结果

### 1.3 技术栈

**后端框架与库：**
- Go 1.22+
- Gin Web Framework
- GORM (ORM)
- gorilla/websocket
- JWT (golang-jwt/jwt)
- go-redis/redis
- aliyun-oss-go-sdk
- golang.org/x/oauth2

**数据库与存储：**
- MySQL 8.0（主数据库）
- Redis 7.0（缓存 + 任务队列）
- 阿里云 OSS（框图 JSON + 用户头像）

**第三方服务：**
- GitHub OAuth
- 微信 OAuth (V1.1)
- OpenAI API / Anthropic API

---

## 二、系统架构

### 2.1 整体架构图

```
┌─────────────────────────────────────┐
│          前端应用（React）            │
└────────────┬────────────────────────┘
             │ HTTP / WebSocket
             ▼
┌─────────────────────────────────────┐
│       Nginx (反向代理)               │
└────────────┬────────────────────────┘
             │
        ┌────┴────┐
        │         │
        ▼         ▼
   ┌────────┐ ┌──────────────┐
   │ 前端   │ │   后端 API    │◄────┐
   │ React  │ │  Gin Server   │     │
   └────────┘ └──────┬────────┘     │
                     │               │
            ┌────────┼───────┬───────┴────┐
            │        │       │            │
            ▼        ▼       ▼            ▼
       ┌────────┐ ┌──────┐ ┌───────┐  ┌────────┐
       │ MySQL  │ │Redis │ │  OSS  │  │ Worker │
       └────────┘ └──┬───┘ └───────┘  └───┬────┘
                     │                     │
                     └─────── Queue ───────┘
                              │
                              ▼
                     ┌─────────────────┐
                     │ anal_go_agent   │
                     │   (分析引擎)     │
                     └─────────────────┘
```

### 2.2 项目目录结构

```
go-analyzer-backend/
├── cmd/
│   ├── server/              # API 服务入口
│   │   └── main.go
│   └── worker/              # Worker 服务入口
│       └── main.go
├── internal/
│   ├── api/
│   │   ├── handler/         # HTTP 请求处理器
│   │   │   ├── auth.go      # 认证相关
│   │   │   ├── user.go      # 用户相关
│   │   │   ├── analysis.go  # 分析项目相关
│   │   │   ├── community.go # 广场相关
│   │   │   ├── comment.go   # 评论相关
│   │   │   └── websocket.go # WebSocket
│   │   ├── middleware/      # 中间件
│   │   │   ├── auth.go      # JWT 认证
│   │   │   ├── quota.go     # 配额检查
│   │   │   ├── cors.go      # 跨域
│   │   │   └── ratelimit.go # 限流
│   │   └── router.go        # 路由配置
│   ├── service/             # 业务逻辑层
│   │   ├── auth_service.go
│   │   ├── user_service.go
│   │   ├── analysis_service.go
│   │   ├── community_service.go
│   │   ├── comment_service.go
│   │   ├── quota_service.go
│   │   └── analyzer/
│   │       └── analyzer_service.go  # 封装 anal_go_agent
│   ├── repository/          # 数据访问层
│   │   ├── user_repo.go
│   │   ├── analysis_repo.go
│   │   ├── comment_repo.go
│   │   ├── interaction_repo.go
│   │   └── job_repo.go
│   ├── model/               # 数据模型
│   │   ├── user.go
│   │   ├── analysis.go
│   │   ├── comment.go
│   │   ├── interaction.go
│   │   ├── job.go
│   │   ├── subscription.go
│   │   └── dto/             # 数据传输对象
│   │       ├── auth_dto.go
│   │       ├── analysis_dto.go
│   │       ├── community_dto.go
│   │       └── common_dto.go
│   ├── pkg/                 # 工具包
│   │   ├── oauth/           # OAuth 客户端
│   │   │   ├── github.go
│   │   │   └── wechat.go
│   │   ├── jwt/             # JWT 工具
│   │   │   └── jwt.go
│   │   ├── oss/             # OSS 客户端
│   │   │   └── client.go
│   │   ├── ws/              # WebSocket Hub
│   │   │   └── hub.go
│   │   ├── queue/           # Redis 队列
│   │   │   └── queue.go
│   │   ├── email/           # 邮件发送
│   │   │   └── email.go
│   │   └── validator/       # 数据验证
│   │       └── validator.go
│   └── config/              # 配置管理
│       └── config.go
├── migrations/              # 数据库迁移文件
│   ├── 001_create_users.sql
│   ├── 002_create_analyses.sql
│   ├── 003_create_comments.sql
│   ├── 004_create_interactions.sql
│   ├── 005_create_analysis_jobs.sql
│   └── 006_create_subscriptions.sql
├── scripts/                 # 脚本
│   ├── migrate.sh
│   └── seed.sh
├── .env.example             # 环境变量示例
├── .gitignore
├── Dockerfile.server
├── Dockerfile.worker
├── docker-compose.yml
├── go.mod
├── go.sum
├── Makefile
└── README.md
```

---

## 三、数据库设计

### 3.1 用户表 (users)

```sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    username VARCHAR(50) UNIQUE NOT NULL COMMENT '用户名',
    email VARCHAR(100) UNIQUE COMMENT '邮箱',
    password_hash VARCHAR(255) COMMENT '密码哈希（OAuth用户为空）',
    avatar_url VARCHAR(500) COMMENT '头像URL（OSS）',
    bio TEXT COMMENT '个人简介',
    
    -- OAuth 信息
    github_id VARCHAR(50) UNIQUE COMMENT 'GitHub ID',
    wechat_openid VARCHAR(100) UNIQUE COMMENT '微信OpenID',
    
    -- 配额信息
    subscription_level ENUM('free', 'basic', 'pro') DEFAULT 'free' COMMENT '套餐级别',
    daily_quota INT DEFAULT 5 COMMENT '每日配额',
    quota_used_today INT DEFAULT 0 COMMENT '今日已用配额',
    quota_reset_at DATETIME COMMENT '配额重置时间',
    subscription_expires_at DATETIME COMMENT '订阅过期时间',
    
    -- 邮箱验证
    email_verified BOOLEAN DEFAULT FALSE COMMENT '邮箱是否验证',
    verification_code VARCHAR(100) COMMENT '验证码',
    verification_expires_at DATETIME COMMENT '验证码过期时间',
    
    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_github_id (github_id),
    INDEX idx_wechat_openid (wechat_openid),
    INDEX idx_email (email),
    INDEX idx_verification_code (verification_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';
```

**说明：**
- `password_hash`: 使用 bcrypt 加密
- OAuth 用户的 `password_hash` 为 NULL
- `subscription_level`: 免费/基础/专业版
- `quota_reset_at`: 每日凌晨自动重置
- `email_verified`: 注册时需要验证邮箱

### 3.2 分析项目表 (analyses)

```sql
CREATE TABLE analyses (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    
    -- 基本信息
    title VARCHAR(200) NOT NULL COMMENT '项目名称',
    description TEXT COMMENT '项目描述',
    
    -- 分析配置
    creation_type ENUM('ai', 'manual') NOT NULL COMMENT '创建方式',
    repo_url VARCHAR(500) COMMENT 'GitHub仓库地址',
    start_struct VARCHAR(100) COMMENT '起始结构体',
    analysis_depth INT COMMENT '分析深度',
    model_name VARCHAR(50) COMMENT '使用的模型',
    
    -- 数据存储
    diagram_oss_url VARCHAR(500) COMMENT '框图JSON的OSS地址',
    diagram_size INT COMMENT '压缩后大小(bytes)',
    
    -- 分析任务状态
    status ENUM('draft', 'pending', 'analyzing', 'completed', 'failed') DEFAULT 'draft' COMMENT '状态',
    error_message TEXT COMMENT '错误信息',
    started_at DATETIME COMMENT '分析开始时间',
    completed_at DATETIME COMMENT '分析完成时间',
    
    -- 分享状态
    is_public BOOLEAN DEFAULT FALSE COMMENT '是否公开分享',
    shared_at DATETIME COMMENT '分享时间',
    share_title VARCHAR(200) COMMENT '分享标题',
    share_description TEXT COMMENT '分享描述',
    tags JSON COMMENT '标签数组',
    
    -- 统计数据
    view_count INT DEFAULT 0 COMMENT '浏览数',
    like_count INT DEFAULT 0 COMMENT '点赞数',
    comment_count INT DEFAULT 0 COMMENT '评论数',
    bookmark_count INT DEFAULT 0 COMMENT '收藏数',
    
    -- 时间戳
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_is_public (is_public),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    INDEX idx_shared_at (shared_at),
    FULLTEXT INDEX ft_share_title_desc (share_title, share_description)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='分析项目表';
```

**说明：**
- `creation_type`: ai（AI分析）/ manual（手动创建）
- `status`: draft（草稿）/ pending（待分析）/ analyzing（分析中）/ completed（完成）/ failed（失败）
- `is_public`: true 表示已分享到广场
- `tags`: JSON 数组，如 `["Web框架", "微服务"]`

### 3.3 互动表 (interactions)

```sql
CREATE TABLE interactions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    analysis_id BIGINT NOT NULL COMMENT '分析ID',
    type ENUM('like', 'bookmark') NOT NULL COMMENT '互动类型',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE,
    UNIQUE KEY uk_user_analysis_type (user_id, analysis_id, type),
    INDEX idx_analysis_type (analysis_id, type),
    INDEX idx_user_id (user_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='互动表（点赞、收藏）';
```

**说明：**
- `type`: like（点赞）/ bookmark（收藏）
- 唯一索引确保一个用户对同一分析只能点赞/收藏一次

### 3.4 评论表 (comments)

```sql
CREATE TABLE comments (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL COMMENT '用户ID',
    analysis_id BIGINT NOT NULL COMMENT '分析ID',
    parent_id BIGINT COMMENT '父评论ID（一级回复）',
    content TEXT NOT NULL COMMENT '评论内容（纯文本，最大500字符）',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE,
    FOREIGN KEY (parent_id) REFERENCES comments(id) ON DELETE CASCADE,
    INDEX idx_analysis_id (analysis_id),
    INDEX idx_parent_id (parent_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='评论表';
```

**说明：**
- `parent_id`: NULL 表示一级评论，非NULL表示回复
- 只支持一级回复，不支持嵌套
- `content`: 纯文本，不支持 Markdown

### 3.5 分析任务队列表 (analysis_jobs)

```sql
CREATE TABLE analysis_jobs (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    analysis_id BIGINT NOT NULL COMMENT '分析ID',
    user_id BIGINT NOT NULL COMMENT '用户ID',
    
    -- 任务配置
    repo_url VARCHAR(500) NOT NULL,
    start_struct VARCHAR(100) NOT NULL,
    depth INT NOT NULL,
    model_name VARCHAR(50) NOT NULL,
    
    -- 任务状态
    status ENUM('queued', 'processing', 'completed', 'failed', 'cancelled') DEFAULT 'queued',
    current_step VARCHAR(200) COMMENT '当前步骤',
    error_message TEXT,
    
    -- 时间统计
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    started_at DATETIME COMMENT '开始处理时间',
    completed_at DATETIME COMMENT '完成时间',
    elapsed_seconds INT COMMENT '耗时（秒）',
    
    FOREIGN KEY (analysis_id) REFERENCES analyses(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_status (status),
    INDEX idx_user_id (user_id),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='分析任务队列表';
```

**说明：**
- `status`: queued（排队）/ processing（处理中）/ completed（完成）/ failed（失败）/ cancelled（取消）
- `current_step`: 实时更新当前步骤，如"正在解析结构体"
- `elapsed_seconds`: 总耗时（秒）

### 3.6 订阅记录表 (subscriptions)

```sql
CREATE TABLE subscriptions (
    id BIGINT PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT NOT NULL,
    plan ENUM('basic', 'pro') NOT NULL COMMENT '套餐',
    amount DECIMAL(10, 2) COMMENT '金额',
    daily_quota INT COMMENT '每日配额',
    started_at DATETIME NOT NULL COMMENT '生效时间',
    expires_at DATETIME NOT NULL COMMENT '过期时间',
    status ENUM('active', 'expired', 'cancelled') DEFAULT 'active',
    payment_method ENUM('wechat', 'alipay') COMMENT '支付方式',
    transaction_id VARCHAR(100) COMMENT '交易ID',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_expires_at (expires_at),
    INDEX idx_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='订阅记录表';
```

**说明：**
- `plan`: basic（基础版 ¥19.9/月）/ pro（专业版 ¥49.9/月）
- `status`: active（生效中）/ expired（已过期）/ cancelled（已取消）
- 支付相关字段在 V1.2 版本使用
## 四、API 接口定义

### 4.1 通用响应格式

**成功响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": { ... }
}
```

**错误响应：**
```json
{
  "code": 错误码,
  "message": "错误描述",
  "data": null
}
```

**错误码定义：**
```go
const (
    CodeSuccess       = 0    // 成功
    CodeParamError    = 1000 // 参数错误
    CodeAuthFailed    = 1001 // 认证失败
    CodePermissionDenied = 1002 // 权限不足
    CodeResourceNotFound = 1003 // 资源不存在
    CodeQuotaExceeded    = 1004 // 配额不足
    CodeDuplicateAction  = 1005 // 重复操作
    CodeServerError      = 5000 // 服务器内部错误
)
```

---

### 4.2 认证相关 API

#### POST /api/v1/auth/register
邮箱密码注册

**请求体：**
```json
{
  "username": "string (3-50字符)",
  "email": "string (有效邮箱)",
  "password": "string (8-32字符，含大小写字母和数字)"
}
```

**响应：**
```json
{
  "code": 0,
  "message": "注册成功，请查收验证邮件",
  "data": {
    "user_id": 1
  }
}
```

**验证规则：**
- 用户名：3-50字符，仅字母、数字、下划线
- 邮箱：有效格式
- 密码：8-32字符，至少包含大小写字母和数字

**业务逻辑：**
1. 验证参数
2. 检查邮箱是否已存在
3. 检查用户名是否已存在
4. 加密密码（bcrypt）
5. 创建用户记录
6. 生成验证码
7. 发送验证邮件
8. 返回 user_id

---

#### POST /api/v1/auth/verify-email
验证邮箱

**请求体：**
```json
{
  "code": "string (验证码)"
}
```

**响应：**
```json
{
  "code": 0,
  "message": "邮箱验证成功",
  "data": {
    "token": "jwt_token_string",
    "user": {
      "id": 1,
      "username": "user1",
      "email": "user@example.com",
      "avatar_url": "",
      "subscription_level": "free"
    }
  }
}
```

---

#### POST /api/v1/auth/login
邮箱密码登录

**请求体：**
```json
{
  "email": "string",
  "password": "string"
}
```

**响应：**
```json
{
  "code": 0,
  "message": "登录成功",
  "data": {
    "token": "jwt_token_string",
    "user": {
      "id": 1,
      "username": "user1",
      "email": "user@example.com",
      "avatar_url": "https://oss.example.com/avatars/1.jpg",
      "bio": "Go developer",
      "subscription_level": "free"
    }
  }
}
```

**业务逻辑：**
1. 验证邮箱存在
2. 验证邮箱已验证
3. 比对密码（bcrypt）
4. 生成 JWT Token（有效期 7 天）
5. 返回 token 和用户信息

---

#### GET /api/v1/auth/github
GitHub OAuth 登录（重定向）

**功能：**
- 重定向到 GitHub 授权页面
- 携带 client_id、redirect_uri、scope

**重定向 URL：**
```
https://github.com/login/oauth/authorize?client_id=xxx&redirect_uri=xxx&scope=user:email
```

---

#### GET /api/v1/auth/github/callback
GitHub OAuth 回调

**查询参数：**
- code: GitHub 返回的授权码

**业务逻辑：**
1. 用 code 换取 access_token
2. 用 access_token 获取 GitHub 用户信息
3. 检查 github_id 是否已存在
   - 存在：直接登录
   - 不存在：创建新用户
4. 生成 JWT Token
5. 重定向到前端并携带 token

**重定向 URL：**
```
https://frontend.example.com/auth/callback?token=jwt_token
```

---

#### GET /api/v1/auth/wechat
微信 OAuth 登录（V1.1 版本）

#### GET /api/v1/auth/wechat/callback
微信 OAuth 回调（V1.1 版本）

---

### 4.3 用户相关 API

#### GET /api/v1/user/profile
获取当前用户信息

**认证：** 需要 JWT Token

**响应：**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "username": "user1",
    "email": "user@example.com",
    "avatar_url": "https://oss.example.com/avatars/1.jpg",
    "bio": "Go developer",
    "subscription_level": "free",
    "email_verified": true,
    "quota_info": {
      "daily_quota": 5,
      "quota_used_today": 3,
      "quota_remaining": 2,
      "quota_reset_at": "2025-01-21T00:00:00Z"
    },
    "subscription_info": null,
    "created_at": "2025-01-15T10:00:00Z"
  }
}
```

---

#### PUT /api/v1/user/profile
更新用户信息

**认证：** 需要

**请求体：**
```json
{
  "username": "new_name (可选)",
  "bio": "new bio (可选)"
}
```

**响应：**
```json
{
  "code": 0,
  "message": "更新成功",
  "data": {
    "id": 1,
    "username": "new_name",
    "bio": "new bio"
  }
}
```

---

#### POST /api/v1/user/avatar
上传头像

**认证：** 需要

**请求：** multipart/form-data
- file: 图片文件（jpg/png，最大 5MB）

**响应：**
```json
{
  "code": 0,
  "message": "上传成功",
  "data": {
    "avatar_url": "https://oss.example.com/avatars/1.jpg"
  }
}
```

**处理流程：**
1. 验证文件格式（jpg/png）
2. 验证文件大小（≤ 5MB）
3. 压缩图片（最大 800x800）
4. 上传到 OSS
5. 删除旧头像（如果有）
6. 更新用户头像 URL

---

### 4.4 分析项目相关 API

#### GET /api/v1/analyses
获取我的分析列表

**认证：** 需要

**查询参数：**
- page: 页码（默认 1）
- page_size: 每页数量（默认 20，最大 100）
- search: 搜索关键词（可选）
- status: 状态过滤（可选：draft, completed, failed）

**响应：**
```json
{
  "code": 0,
  "data": {
    "total": 10,
    "page": 1,
    "page_size": 20,
    "items": [
      {
        "id": 1,
        "title": "Gin 路由分析",
        "creation_type": "ai",
        "status": "completed",
        "is_public": true,
        "view_count": 100,
        "like_count": 20,
        "comment_count": 5,
        "created_at": "2025-01-20T10:00:00Z",
        "updated_at": "2025-01-20T10:30:00Z"
      }
    ]
  }
}
```

---

#### POST /api/v1/analyses
创建分析项目

**认证：** 需要

**请求体（AI 分析）：**
```json
{
  "title": "Gin 路由分析",
  "creation_type": "ai",
  "repo_url": "https://github.com/gin-gonic/gin",
  "start_struct": "Engine",
  "analysis_depth": 3,
  "model_name": "gpt-3.5-turbo"
}
```

**请求体（手动创建）：**
```json
{
  "title": "我的架构设计",
  "creation_type": "manual",
  "diagram_data": {
    "structs": [...],
    "connections": [...]
  }
}
```

**响应：**
```json
{
  "code": 0,
  "message": "创建成功",
  "data": {
    "analysis_id": 123,
    "job_id": 456  // 仅 AI 分析返回
  }
}
```

**业务逻辑（AI 分析）：**
1. 验证用户配额
   - 检查今日配额是否足够
   - 不足返回 1004 错误
2. 验证深度限制
   - 免费：≤ 3
   - 基础：≤ 5
   - 专业：≤ 10
3. 验证模型权限
   - 免费：gpt-3.5, claude-haiku
   - 基础：+ gpt-4o-mini
   - 专业：所有模型
4. 创建 Analysis 记录（status: pending）
5. 扣除配额
6. 创建 Job 记录（status: queued）
7. 加入 Redis 队列
8. 返回 analysis_id 和 job_id

**业务逻辑（手动创建）：**
1. 创建 Analysis 记录（status: draft）
2. 如果提供了 diagram_data：
   - 压缩 JSON (gzip)
   - 上传到 OSS
   - 更新 diagram_oss_url
3. 返回 analysis_id

**错误处理：**
- 配额不足：1004
- 模型权限不足：1002
- 深度超限：1000
- 仓库 URL 无效：1000

---

#### GET /api/v1/analyses/:id
获取分析详情

**认证：** 需要（仅自己的分析）

**响应：**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "title": "Gin 路由分析",
    "description": "",
    "creation_type": "ai",
    "repo_url": "https://github.com/gin-gonic/gin",
    "start_struct": "Engine",
    "analysis_depth": 3,
    "model_name": "gpt-3.5-turbo",
    "diagram_oss_url": "https://oss.example.com/diagrams/1.json.gz",
    "diagram_size": 102400,
    "status": "completed",
    "is_public": false,
    "view_count": 0,
    "like_count": 0,
    "comment_count": 0,
    "started_at": "2025-01-20T10:00:00Z",
    "completed_at": "2025-01-20T10:05:00Z",
    "created_at": "2025-01-20T10:00:00Z",
    "updated_at": "2025-01-20T10:30:00Z"
  }
}
```

---

#### PUT /api/v1/analyses/:id
更新分析项目

**认证：** 需要（仅自己的）

**请求体：**
```json
{
  "title": "新标题 (可选)",
  "description": "新描述 (可选)",
  "diagram_data": {
    "structs": [...],
    "connections": [...]
  }
}
```

**响应：**
```json
{
  "code": 0,
  "message": "更新成功",
  "data": {
    "id": 1,
    "updated_at": "2025-01-20T11:00:00Z"
  }
}
```

**业务逻辑：**
1. 验证权限（只能更新自己的）
2. 更新基本信息（title, description）
3. 如果提供 diagram_data：
   - 压缩 JSON
   - 上传到 OSS（覆盖旧文件）
   - 更新 diagram_oss_url 和 diagram_size
4. 更新 updated_at
5. 返回成功

---

#### DELETE /api/v1/analyses/:id
删除分析项目

**认证：** 需要（仅自己的）

**响应：**
```json
{
  "code": 0,
  "message": "删除成功"
}
```

**业务逻辑：**
1. 验证权限
2. 删除 OSS 上的文件
3. 删除数据库记录（级联删除评论、互动）
4. 如果有进行中的任务，取消任务
5. 返回成功

---

#### POST /api/v1/analyses/:id/share
分享到广场

**认证：** 需要

**请求体：**
```json
{
  "share_title": "Gin 框架路由模块分析",
  "share_description": "详细分析了 Gin 的路由实现原理",
  "tags": ["Web框架", "路由", "Gin"]
}
```

**响应：**
```json
{
  "code": 0,
  "message": "分享成功"
}
```

**业务逻辑：**
1. 验证权限（只能分享自己的）
2. 验证分析状态（只能分享 completed 的）
3. 更新字段：
   - is_public = true
   - shared_at = now
   - share_title, share_description, tags
4. 返回成功

---

#### DELETE /api/v1/analyses/:id/share
取消分享

**认证：** 需要

**响应：**
```json
{
  "code": 0,
  "message": "已取消分享"
}
```

**业务逻辑：**
- 设置 is_public = false
- 清空 shared_at

---

#### GET /api/v1/analyses/:id/job-status
获取分析任务状态

**认证：** 需要

**响应：**
```json
{
  "code": 0,
  "data": {
    "job_id": 456,
    "analysis_id": 123,
    "status": "processing",
    "current_step": "正在解析结构体",
    "elapsed_seconds": 45,
    "error_message": null,
    "started_at": "2025-01-20T10:00:00Z"
  }
}
```

---

### 4.5 广场相关 API

#### GET /api/v1/community/analyses
获取广场分析列表

**认证：** 不需要

**查询参数：**
- page: 页码（默认 1）
- page_size: 每页数量（默认 20）
- sort: 排序方式（latest / hot，默认 latest）
- tags: 标签过滤（逗号分隔，可选）

**响应：**
```json
{
  "code": 0,
  "data": {
    "total": 100,
    "page": 1,
    "page_size": 20,
    "items": [
      {
        "id": 1,
        "share_title": "Gin 框架路由分析",
        "share_description": "详细分析...",
        "tags": ["Web框架", "路由"],
        "author": {
          "id": 10,
          "username": "gopher",
          "avatar_url": "..."
        },
        "view_count": 100,
        "like_count": 20,
        "comment_count": 5,
        "bookmark_count": 3,
        "shared_at": "2025-01-20T10:00:00Z"
      }
    ]
  }
}
```

**排序逻辑：**
- latest: `ORDER BY shared_at DESC`
- hot: `ORDER BY (like_count * 3 + comment_count * 2 + view_count) DESC`

---

#### GET /api/v1/community/analyses/:id
获取广场分析详情

**认证：** 不需要（已登录则返回互动状态）

**响应：**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "share_title": "Gin 框架路由分析",
    "share_description": "详细分析...",
    "tags": ["Web框架", "路由"],
    "author": {
      "id": 10,
      "username": "gopher",
      "avatar_url": "...",
      "bio": "..."
    },
    "diagram_oss_url": "https://oss.example.com/diagrams/1.json.gz",
    "creation_type": "ai",
    "repo_url": "https://github.com/gin-gonic/gin",
    "view_count": 101,
    "like_count": 20,
    "comment_count": 5,
    "bookmark_count": 3,
    "shared_at": "2025-01-20T10:00:00Z",
    "user_interaction": {
      "liked": false,
      "bookmarked": false
    }
  }
}
```

**业务逻辑：**
1. 查询分析详情（必须 is_public = true）
2. 增加 view_count（使用 Redis 计数器，定期写入数据库）
3. 如果已登录，查询当前用户的互动状态

---

#### POST /api/v1/analyses/:id/like
点赞

**认证：** 需要

**响应：**
```json
{
  "code": 0,
  "message": "点赞成功",
  "data": {
    "liked": true,
    "like_count": 21
  }
}
```

**业务逻辑：**
1. 检查是否已点赞
2. 如果未点赞：
   - 插入 interactions 记录（type: like）
   - 增加 analyses.like_count
3. 返回新的点赞状态（幂等性）

---

#### DELETE /api/v1/analyses/:id/like
取消点赞

**认证：** 需要

**响应：**
```json
{
  "code": 0,
  "message": "已取消点赞",
  "data": {
    "liked": false,
    "like_count": 20
  }
}
```

**业务逻辑：**
1. 删除 interactions 记录
2. 减少 analyses.like_count

---

#### POST /api/v1/analyses/:id/bookmark
收藏

**认证：** 需要

**响应：**
```json
{
  "code": 0,
  "message": "收藏成功",
  "data": {
    "bookmarked": true,
    "bookmark_count": 4
  }
}
```

（逻辑同点赞）

#### DELETE /api/v1/analyses/:id/bookmark
取消收藏

---

### 4.6 评论相关 API

#### GET /api/v1/analyses/:id/comments
获取评论列表

**认证：** 不需要

**查询参数：**
- page: 页码（默认 1）
- page_size: 每页数量（默认 20）

**响应：**
```json
{
  "code": 0,
  "data": {
    "total": 10,
    "page": 1,
    "page_size": 20,
    "items": [
      {
        "id": 1,
        "user": {
          "id": 20,
          "username": "commenter",
          "avatar_url": "..."
        },
        "content": "分析得很好！",
        "parent_id": null,
        "replies": [
          {
            "id": 2,
            "user": {
              "id": 10,
              "username": "author",
              "avatar_url": "..."
            },
            "content": "谢谢！",
            "parent_id": 1,
            "created_at": "2025-01-20T11:05:00Z"
          }
        ],
        "created_at": "2025-01-20T11:00:00Z"
      }
    ]
  }
}
```

**数据组装逻辑：**
1. 查询一级评论（parent_id IS NULL）
2. 查询二级回复（parent_id IN 一级评论ID）
3. 组装成树形结构
4. 按时间倒序排列

---

#### POST /api/v1/analyses/:id/comments
发表评论

**认证：** 需要

**请求体：**
```json
{
  "content": "很棒的分析！",
  "parent_id": null  // 回复时填写父评论ID
}
```

**响应：**
```json
{
  "code": 0,
  "message": "评论成功",
  "data": {
    "id": 3,
    "user": {
      "id": 15,
      "username": "viewer",
      "avatar_url": "..."
    },
    "content": "很棒的分析！",
    "parent_id": null,
    "created_at": "2025-01-20T11:10:00Z"
  }
}
```

**验证：**
- content: 1-500 字符，纯文本
- parent_id: 如果不为空，验证父评论存在且属于同一个分析

**业务逻辑：**
1. 验证 content 长度
2. 如果是回复，验证 parent_id 有效性
3. 插入评论记录
4. 增加 analyses.comment_count
5. 返回新评论

---

#### DELETE /api/v1/comments/:id
删除评论

**认证：** 需要（仅自己的评论）

**响应：**
```json
{
  "code": 0,
  "message": "删除成功"
}
```

**业务逻辑：**
1. 验证权限（只能删除自己的）
2. 级联删除子回复
3. 减少 analyses.comment_count
4. 返回成功

---

### 4.7 配额相关 API

#### GET /api/v1/quota/info
获取配额信息

**认证：** 需要

**响应：**
```json
{
  "code": 0,
  "data": {
    "subscription_level": "free",
    "daily_quota": 5,
    "quota_used_today": 3,
    "quota_remaining": 2,
    "quota_reset_at": "2025-01-21T00:00:00Z",
    "subscription_expires_at": null
  }
}
```

---

### 4.8 模型相关 API

#### GET /api/v1/models
获取可用模型列表

**认证：** 不需要

**响应：**
```json
{
  "code": 0,
  "data": {
    "models": [
      {
        "name": "gpt-3.5-turbo",
        "display_name": "GPT-3.5 Turbo",
        "required_level": "free",
        "description": "基础模型，适合简单分析",
        "speed": "fast",
        "quality": "good"
      },
      {
        "name": "gpt-4o-mini",
        "display_name": "GPT-4o Mini",
        "required_level": "basic",
        "description": "中级模型，平衡速度和质量",
        "speed": "medium",
        "quality": "very_good"
      },
      {
        "name": "gpt-4",
        "display_name": "GPT-4",
        "required_level": "pro",
        "description": "高级模型，适合复杂分析",
        "speed": "slow",
        "quality": "excellent"
      },
      {
        "name": "claude-sonnet",
        "display_name": "Claude Sonnet",
        "required_level": "pro",
        "description": "高级模型，适合复杂分析",
        "speed": "medium",
        "quality": "excellent"
      }
    ]
  }
}
```

---

### 4.9 WebSocket 接口

#### WS /api/v1/ws
建立 WebSocket 连接

**认证：** JWT Token（通过查询参数）

**连接 URL：**
```
ws://api.example.com/api/v1/ws?token=jwt_token_string
```

**消息类型：**

1. **分析进度更新**
```json
{
  "type": "analysis_progress",
  "data": {
    "job_id": 456,
    "analysis_id": 123,
    "status": "processing",
    "current_step": "正在分析依赖关系",
    "elapsed_seconds": 60
  }
}
```

2. **分析完成**
```json
{
  "type": "analysis_completed",
  "data": {
    "job_id": 456,
    "analysis_id": 123,
    "diagram_oss_url": "https://oss.example.com/diagrams/123.json.gz",
    "elapsed_seconds": 120
  }
}
```

3. **分析失败**
```json
{
  "type": "analysis_failed",
  "data": {
    "job_id": 456,
    "analysis_id": 123,
    "error_message": "结构体未找到：Engine",
    "elapsed_seconds": 30
  }
}
```

**服务端推送时机：**
- Worker 更新 job.current_step 时
- 分析完成时
- 分析失败时
## 五、核心业务逻辑

### 5.1 分析任务流程

#### API Server 部分

```go
package service

// CreateAnalysis 创建分析任务
func (s *AnalysisService) CreateAnalysis(req *dto.CreateAnalysisRequest) (*dto.CreateAnalysisResponse, error) {
    // 1. 验证配额
    hasQuota, err := s.quotaService.CheckQuota(req.UserID)
    if err != nil {
        return nil, err
    }
    if !hasQuota {
        return nil, errors.New("今日配额已用完")
    }

    // 2. 验证深度限制
    user, err := s.userRepo.GetByID(req.UserID)
    if err != nil {
        return nil, err
    }

    maxDepth := s.getMaxDepthByLevel(user.SubscriptionLevel)
    if req.Depth > maxDepth {
        return nil, fmt.Errorf("分析深度超过限制，当前套餐最大深度：%d", maxDepth)
    }

    // 3. 验证模型权限
    if !s.checkModelPermission(user.SubscriptionLevel, req.ModelName) {
        return nil, errors.New("当前套餐无法使用该模型，请升级")
    }

    // 4. 创建 Analysis 记录
    analysis := &model.Analysis{
        UserID:        req.UserID,
        Title:         req.Title,
        CreationType:  req.CreationType,
        RepoURL:       req.RepoURL,
        StartStruct:   req.StartStruct,
        AnalysisDepth: req.Depth,
        ModelName:     req.ModelName,
        Status:        "pending",
    }

    if req.CreationType == "manual" {
        analysis.Status = "draft"
    }

    if err := s.analysisRepo.Create(analysis); err != nil {
        return nil, err
    }

    // 5. 如果是手动创建且提供了数据，上传到 OSS
    if req.CreationType == "manual" && req.DiagramData != nil {
        ossURL, size, err := s.uploadDiagramToOSS(analysis.ID, req.DiagramData)
        if err != nil {
            return nil, err
        }
        analysis.DiagramOSSURL = ossURL
        analysis.DiagramSize = size
        analysis.Status = "completed"
        s.analysisRepo.Update(analysis)
    }

    // 6. 如果是 AI 分析，创建任务
    var jobID int64
    if req.CreationType == "ai" {
        // 扣除配额
        if err := s.quotaService.UseQuota(req.UserID); err != nil {
            return nil, err
        }

        // 创建 Job 记录
        job := &model.AnalysisJob{
            AnalysisID: analysis.ID,
            UserID:     req.UserID,
            RepoURL:    req.RepoURL,
            StartStruct: req.StartStruct,
            Depth:      req.Depth,
            ModelName:  req.ModelName,
            Status:     "queued",
        }

        if err := s.jobRepo.Create(job); err != nil {
            return nil, err
        }
        jobID = job.ID

        // 加入 Redis 队列
        if err := s.queue.Push(job.ID); err != nil {
            return nil, err
        }
    }

    return &dto.CreateAnalysisResponse{
        AnalysisID: analysis.ID,
        JobID:      jobID,
    }, nil
}

func (s *AnalysisService) getMaxDepthByLevel(level string) int {
    switch level {
    case "free":
        return 3
    case "basic":
        return 5
    case "pro":
        return 10
    default:
        return 3
    }
}

func (s *AnalysisService) checkModelPermission(level, modelName string) bool {
    config := s.config.GetModelConfig(modelName)
    if config == nil {
        return false
    }

    switch level {
    case "free":
        return config.RequiredLevel == "free"
    case "basic":
        return config.RequiredLevel == "free" || config.RequiredLevel == "basic"
    case "pro":
        return true
    default:
        return false
    }
}
```

#### Worker 部分

```go
package worker

type Worker struct {
    queue         *queue.Queue
    jobRepo       repository.JobRepository
    analysisRepo  repository.AnalysisRepository
    analyzerSvc   *analyzer.Service
    ossClient     *oss.Client
    wsHub         *ws.Hub
}

func (w *Worker) Start() {
    for {
        // 从队列获取任务
        jobID, err := w.queue.Pop()
        if err != nil {
            time.Sleep(1 * time.Second)
            continue
        }

        go w.ProcessJob(jobID)
    }
}

func (w *Worker) ProcessJob(jobID int64) {
    // 1. 获取任务
    job, err := w.jobRepo.GetByID(jobID)
    if err != nil {
        log.Error("Failed to get job:", err)
        return
    }

    // 2. 更新状态为处理中
    job.Status = "processing"
    job.StartedAt = time.Now()
    w.jobRepo.Update(job)

    // 3. 更新 Analysis 状态
    w.analysisRepo.UpdateStatus(job.AnalysisID, "analyzing")

    // 4. Clone 仓库到临时目录
    tempDir, err := w.cloneRepo(job.RepoURL)
    if err != nil {
        w.handleJobFailure(job, fmt.Sprintf("克隆仓库失败: %v", err))
        return
    }
    defer os.RemoveAll(tempDir)

    // 5. 调用分析库
    result, err := w.analyzerSvc.Analyze(context.Background(), &analyzer.Config{
        ProjectPath: tempDir,
        StartStruct: job.StartStruct,
        Depth:       job.Depth,
        ModelName:   job.ModelName,
        OnProgress: func(step string) {
            // 更新当前步骤
            job.CurrentStep = step
            w.jobRepo.UpdateStep(job.ID, step)

            // 计算耗时
            elapsed := int(time.Since(job.StartedAt).Seconds())

            // 推送进度消息
            w.wsHub.SendToUser(job.UserID, &ws.Message{
                Type: "analysis_progress",
                Data: map[string]interface{}{
                    "job_id":          job.ID,
                    "analysis_id":     job.AnalysisID,
                    "status":          "processing",
                    "current_step":    step,
                    "elapsed_seconds": elapsed,
                },
            })
        },
    })

    if err != nil {
        w.handleJobFailure(job, fmt.Sprintf("分析失败: %v", err))
        return
    }

    // 6. 成功处理
    // 压缩 JSON
    compressed, err := compressJSON(result.VisualizerJSON)
    if err != nil {
        w.handleJobFailure(job, fmt.Sprintf("压缩失败: %v", err))
        return
    }

    // 上传到 OSS
    ossURL, err := w.ossClient.UploadDiagram(job.AnalysisID, compressed)
    if err != nil {
        w.handleJobFailure(job, fmt.Sprintf("上传OSS失败: %v", err))
        return
    }

    // 更新记录
    elapsed := int(time.Since(job.StartedAt).Seconds())
    job.Status = "completed"
    job.CompletedAt = time.Now()
    job.ElapsedSeconds = elapsed
    w.jobRepo.Update(job)

    w.analysisRepo.Update(&model.Analysis{
        ID:            job.AnalysisID,
        Status:        "completed",
        DiagramOSSURL: ossURL,
        DiagramSize:   len(compressed),
        CompletedAt:   &job.CompletedAt,
    })

    // 推送完成消息
    w.wsHub.SendToUser(job.UserID, &ws.Message{
        Type: "analysis_completed",
        Data: map[string]interface{}{
            "job_id":          job.ID,
            "analysis_id":     job.AnalysisID,
            "diagram_oss_url": ossURL,
            "elapsed_seconds": elapsed,
        },
    })

    log.Infof("Job %d completed in %d seconds", job.ID, elapsed)
}

func (w *Worker) handleJobFailure(job *model.AnalysisJob, errorMsg string) {
    // 更新 Job 状态
    elapsed := int(time.Since(job.StartedAt).Seconds())
    job.Status = "failed"
    job.ErrorMessage = errorMsg
    job.CompletedAt = time.Now()
    job.ElapsedSeconds = elapsed
    w.jobRepo.Update(job)

    // 更新 Analysis 状态
    w.analysisRepo.Update(&model.Analysis{
        ID:           job.AnalysisID,
        Status:       "failed",
        ErrorMessage: errorMsg,
    })

    // 退还配额
    w.quotaService.RefundQuota(job.UserID)

    // 推送失败消息
    w.wsHub.SendToUser(job.UserID, &ws.Message{
        Type: "analysis_failed",
        Data: map[string]interface{}{
            "job_id":          job.ID,
            "analysis_id":     job.AnalysisID,
            "error_message":   errorMsg,
            "elapsed_seconds": elapsed,
        },
    })

    log.Errorf("Job %d failed: %s", job.ID, errorMsg)
}

func (w *Worker) cloneRepo(repoURL string) (string, error) {
    tempDir, err := os.MkdirTemp("", "go-analyzer-*")
    if err != nil {
        return "", err
    }

    cmd := exec.Command("git", "clone", "--depth", "1", repoURL, tempDir)
    if err := cmd.Run(); err != nil {
        os.RemoveAll(tempDir)
        return "", err
    }

    return tempDir, nil
}

func compressJSON(data []byte) ([]byte, error) {
    var buf bytes.Buffer
    gw := gzip.NewWriter(&buf)
    if _, err := gw.Write(data); err != nil {
        return nil, err
    }
    if err := gw.Close(); err != nil {
        return nil, err
    }
    return buf.Bytes(), nil
}
```

---

### 5.2 配额管理

```go
package service

type QuotaService struct {
    userRepo repository.UserRepository
    redis    *redis.Client
}

// CheckQuota 检查配额
func (s *QuotaService) CheckQuota(userID int64) (bool, error) {
    user, err := s.userRepo.GetByID(userID)
    if err != nil {
        return false, err
    }

    // 检查是否需要重置
    if time.Now().After(user.QuotaResetAt) {
        if err := s.resetUserQuota(userID); err != nil {
            return false, err
        }
        user, _ = s.userRepo.GetByID(userID)
    }

    return user.QuotaUsedToday < user.DailyQuota, nil
}

// UseQuota 使用配额
func (s *QuotaService) UseQuota(userID int64) error {
    return s.userRepo.IncrementQuotaUsed(userID)
}

// RefundQuota 退还配额（分析失败时）
func (s *QuotaService) RefundQuota(userID int64) error {
    return s.userRepo.DecrementQuotaUsed(userID)
}

// resetUserQuota 重置用户配额
func (s *QuotaService) resetUserQuota(userID int64) error {
    nextReset := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
    return s.userRepo.ResetQuota(userID, nextReset)
}

// ResetAllQuotas 重置所有用户配额（定时任务）
func (s *QuotaService) ResetAllQuotas() error {
    nextReset := time.Now().Add(24 * time.Hour).Truncate(24 * time.Hour)
    return s.userRepo.ResetAllQuotas(nextReset)
}
```

**定时任务：**
```go
package main

import (
    "github.com/robfig/cron/v3"
)

func startCronJobs(quotaService *service.QuotaService) {
    c := cron.New()

    // 每天凌晨 00:00 重置配额
    c.AddFunc("0 0 * * *", func() {
        if err := quotaService.ResetAllQuotas(); err != nil {
            log.Error("Failed to reset quotas:", err)
        } else {
            log.Info("Successfully reset all quotas")
        }
    })

    c.Start()
}
```

---

### 5.3 WebSocket Hub 实现

```go
package ws

import (
    "github.com/gorilla/websocket"
    "sync"
)

type Hub struct {
    // userID -> *websocket.Conn
    connections map[int64]*websocket.Conn
    mu          sync.RWMutex

    // 注册/注销通道
    register   chan *Client
    unregister chan *Client

    // 广播通道
    broadcast chan *Message
}

type Client struct {
    UserID int64
    Conn   *websocket.Conn
}

type Message struct {
    UserID int64
    Type   string
    Data   interface{}
}

func NewHub() *Hub {
    return &Hub{
        connections: make(map[int64]*websocket.Conn),
        register:    make(chan *Client),
        unregister:  make(chan *Client),
        broadcast:   make(chan *Message, 256),
    }
}

func (h *Hub) Run() {
    for {
        select {
        case client := <-h.register:
            h.mu.Lock()
            h.connections[client.UserID] = client.Conn
            h.mu.Unlock()
            log.Infof("User %d connected", client.UserID)

        case client := <-h.unregister:
            h.mu.Lock()
            delete(h.connections, client.UserID)
            h.mu.Unlock()
            client.Conn.Close()
            log.Infof("User %d disconnected", client.UserID)

        case msg := <-h.broadcast:
            h.mu.RLock()
            conn, ok := h.connections[msg.UserID]
            h.mu.RUnlock()

            if ok {
                if err := conn.WriteJSON(msg); err != nil {
                    log.Errorf("Failed to send message to user %d: %v", msg.UserID, err)
                }
            }
        }
    }
}

func (h *Hub) Register(client *Client) {
    h.register <- client
}

func (h *Hub) Unregister(client *Client) {
    h.unregister <- client
}

func (h *Hub) SendToUser(userID int64, msg *Message) {
    msg.UserID = userID
    h.broadcast <- msg
}
```

**WebSocket Handler：**
```go
package handler

func (h *Handler) HandleWebSocket(c *gin.Context) {
    // 验证 JWT Token
    token := c.Query("token")
    userID, err := h.jwtService.ValidateToken(token)
    if err != nil {
        c.JSON(401, gin.H{"error": "Unauthorized"})
        return
    }

    // 升级连接
    upgrader := websocket.Upgrader{
        CheckOrigin: func(r *http.Request) bool {
            return true // 生产环境需要验证 Origin
        },
    }

    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
        log.Error("Failed to upgrade connection:", err)
        return
    }

    client := &ws.Client{
        UserID: userID,
        Conn:   conn,
    }

    h.wsHub.Register(client)
    defer h.wsHub.Unregister(client)

    // 保持连接
    for {
        _, _, err := conn.ReadMessage()
        if err != nil {
            break
        }
    }
}
```

---

### 5.4 OSS 操作封装

```go
package oss

import (
    "bytes"
    "compress/gzip"
    "fmt"
    "image/jpeg"

    "github.com/aliyun/aliyun-oss-go-sdk/oss"
    "github.com/disintegration/imaging"
)

type Client struct {
    client    *oss.Client
    bucket    *oss.Bucket
    cdnDomain string
}

func NewClient(endpoint, accessKeyID, accessKeySecret, bucketName, cdnDomain string) (*Client, error) {
    client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
    if err != nil {
        return nil, err
    }

    bucket, err := client.Bucket(bucketName)
    if err != nil {
        return nil, err
    }

    return &Client{
        client:    client,
        bucket:    bucket,
        cdnDomain: cdnDomain,
    }, nil
}

// UploadDiagram 上传框图数据（gzip 压缩）
func (c *Client) UploadDiagram(analysisID int64, data []byte) (string, error) {
    // Gzip 压缩
    var buf bytes.Buffer
    gw := gzip.NewWriter(&buf)
    if _, err := gw.Write(data); err != nil {
        return "", err
    }
    if err := gw.Close(); err != nil {
        return "", err
    }

    // 生成对象键（分片存储）
    objectKey := fmt.Sprintf("diagrams/%d/%d.json.gz",
        analysisID/10000, analysisID)

    // 上传
    err := c.bucket.PutObject(objectKey, bytes.NewReader(buf.Bytes()),
        oss.ContentType("application/gzip"),
        oss.ContentEncoding("gzip"),
    )
    if err != nil {
        return "", err
    }

    // 返回 CDN URL
    return fmt.Sprintf("%s/%s", c.cdnDomain, objectKey), nil
}

// UploadAvatar 上传用户头像
func (c *Client) UploadAvatar(userID int64, imageData []byte) (string, error) {
    // 解码图片
    img, err := imaging.Decode(bytes.NewReader(imageData))
    if err != nil {
        return "", err
    }

    // 调整大小（最大 800x800，保持比例）
    resized := imaging.Fit(img, 800, 800, imaging.Lanczos)

    // 编码为 JPEG
    var buf bytes.Buffer
    if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: 85}); err != nil {
        return "", err
    }

    // 生成对象键
    objectKey := fmt.Sprintf("avatars/%d.jpg", userID)

    // 上传（公共读）
    err = c.bucket.PutObject(objectKey, bytes.NewReader(buf.Bytes()),
        oss.ContentType("image/jpeg"),
        oss.ACL(oss.ACLPublicRead),
    )
    if err != nil {
        return "", err
    }

    return fmt.Sprintf("%s/%s", c.cdnDomain, objectKey), nil
}

// DeleteObject 删除对象
func (c *Client) DeleteObject(url string) error {
    // 从 URL 提取 objectKey
    objectKey := strings.TrimPrefix(url, c.cdnDomain+"/")
    return c.bucket.DeleteObject(objectKey)
}
```

---

## 六、配置管理

### 6.1 配置文件 (config.yaml)

```yaml
server:
  host: 0.0.0.0
  port: 8080
  mode: release  # debug, release

database:
  driver: mysql
  host: localhost
  port: 3306
  username: root
  password: password
  database: go_analyzer
  max_idle_conns: 10
  max_open_conns: 100
  log_mode: false

redis:
  host: localhost
  port: 6379
  password: ""
  db: 0
  pool_size: 10

oss:
  endpoint: oss-cn-hangzhou.aliyuncs.com
  access_key_id: YOUR_ACCESS_KEY
  access_key_secret: YOUR_SECRET_KEY
  bucket_name: go-analyzer
  cdn_domain: https://cdn.example.com

jwt:
  secret: your_jwt_secret_key_here
  expire_hours: 168  # 7 days

oauth:
  github:
    client_id: YOUR_GITHUB_CLIENT_ID
    client_secret: YOUR_GITHUB_CLIENT_SECRET
    redirect_uri: http://localhost:8080/api/v1/auth/github/callback
  wechat:
    app_id: YOUR_WECHAT_APP_ID
    app_secret: YOUR_WECHAT_APP_SECRET
    redirect_uri: http://localhost:8080/api/v1/auth/wechat/callback

models:
  - name: gpt-3.5-turbo
    display_name: GPT-3.5 Turbo
    required_level: free
    api_key: YOUR_OPENAI_API_KEY
    api_provider: openai
    description: 基础模型，适合简单分析
  - name: claude-haiku
    display_name: Claude Haiku
    required_level: free
    api_key: YOUR_ANTHROPIC_API_KEY
    api_provider: anthropic
    description: 基础模型，快速分析
  - name: gpt-4o-mini
    display_name: GPT-4o Mini
    required_level: basic
    api_key: YOUR_OPENAI_API_KEY
    api_provider: openai
    description: 中级模型，平衡速度和质量
  - name: gpt-4
    display_name: GPT-4
    required_level: pro
    api_key: YOUR_OPENAI_API_KEY
    api_provider: openai
    description: 高级模型，适合复杂分析
  - name: claude-sonnet
    display_name: Claude Sonnet
    required_level: pro
    api_key: YOUR_ANTHROPIC_API_KEY
    api_provider: anthropic
    description: 高级模型，高质量分析

email:
  smtp_host: smtp.gmail.com
  smtp_port: 587
  username: your-email@gmail.com
  password: your-app-password
  from: noreply@example.com

queue:
  analysis_queue: analysis_jobs
  max_workers: 5

cors:
  allowed_origins:
    - http://localhost:3000
    - https://example.com
  allowed_methods:
    - GET
    - POST
    - PUT
    - DELETE
  allowed_headers:
    - Authorization
    - Content-Type

subscription:
  levels:
    free:
      daily_quota: 5
      max_depth: 3
    basic:
      daily_quota: 30
      max_depth: 5
      price: 19.9
    pro:
      daily_quota: 100
      max_depth: 10
      price: 49.9
```

### 6.2 环境变量 (.env)

```bash
# 服务器
SERVER_PORT=8080
GIN_MODE=release

# 数据库
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=password
DB_NAME=go_analyzer

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# OSS
OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
OSS_ACCESS_KEY_ID=
OSS_ACCESS_KEY_SECRET=
OSS_BUCKET_NAME=go-analyzer
OSS_CDN_DOMAIN=https://cdn.example.com

# JWT
JWT_SECRET=your_jwt_secret

# OAuth
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GITHUB_REDIRECT_URI=

# LLM API Keys
OPENAI_API_KEY=
ANTHROPIC_API_KEY=

# 邮件
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=
SMTP_PASSWORD=
EMAIL_FROM=noreply@example.com

# 前端地址（CORS）
FRONTEND_URL=http://localhost:3000
```

---

## 七、开发任务

### Phase 1: 基础架构（Week 1-2）

#### 数据库
- [ ] 编写 SQL 迁移脚本（6 个表）
- [ ] 创建种子数据（测试用）
- [ ] 设置数据库连接池
- [ ] 配置 GORM 日志

#### 项目骨架
- [ ] 初始化 Go module
- [ ] 搭建目录结构
- [ ] 配置管理（config.yaml + 环境变量）
- [ ] 日志系统（zap）
- [ ] 错误处理中间件
- [ ] 响应统一封装

#### 认证系统
- [ ] JWT 生成和验证
- [ ] 邮箱密码注册
- [ ] 邮件验证
- [ ] 邮箱密码登录
- [ ] GitHub OAuth 登录
- [ ] 认证中间件
- [ ] 密码加密（bcrypt）

---

### Phase 2: 核心功能（Week 3-4）

#### 分析功能
- [ ] 集成 anal_go_agent/pkg
- [ ] 实现 Redis 队列
- [ ] Worker 进程
  - [ ] 从队列消费任务
  - [ ] Clone GitHub 仓库
  - [ ] 调用分析库
  - [ ] 处理进度回调
  - [ ] 上传结果到 OSS
  - [ ] 更新数据库
  - [ ] 错误处理和重试

#### WebSocket
- [ ] 实现 Hub
- [ ] 连接管理
- [ ] 进度推送
- [ ] 心跳检测

#### OSS
- [ ] 封装 OSS 客户端
- [ ] 框图上传/下载
- [ ] 头像上传
- [ ] 文件删除
- [ ] CDN 配置

#### 用户 API
- [ ] 获取用户信息
- [ ] 更新用户信息
- [ ] 上传头像
- [ ] 密码重置

#### 分析项目 API
- [ ] 创建分析（AI + 手动）
- [ ] 获取分析列表
- [ ] 获取分析详情
- [ ] 更新分析
- [ ] 删除分析
- [ ] 分享/取消分享
- [ ] 获取任务状态

---

### Phase 3: 社区功能（Week 5-6）

#### 广场 API
- [ ] 获取广场列表（分页、排序）
- [ ] 获取分析详情（浏览数 +1）
- [ ] 点赞/取消点赞
- [ ] 收藏/取消收藏
- [ ] 浏览数统计（Redis 优化）

#### 评论 API
- [ ] 获取评论列表（含回复）
- [ ] 发表评论
- [ ] 删除评论
- [ ] 评论数统计

#### 配额管理
- [ ] 配额检查中间件
- [ ] 配额使用/退还
- [ ] 每日重置定时任务
- [ ] 获取配额信息 API

#### 模型管理
- [ ] 模型配置加载
- [ ] 获取模型列表 API
- [ ] 权限验证

---

### Phase 4: 测试与部署（Week 7-8）

#### 测试
- [ ] 单元测试（Repository 层）
- [ ] 集成测试（Service 层）
- [ ] API 测试（Handler 层）
- [ ] WebSocket 测试

#### 文档
- [ ] Swagger API 文档
- [ ] 部署文档
- [ ] 开发文档
- [ ] API 使用示例

#### 部署
- [ ] Dockerfile（server + worker）
- [ ] docker-compose.yml
- [ ] Kubernetes 配置
- [ ] CI/CD 配置
- [ ] 监控告警

---

## 八、开发规范

### 8.1 代码规范
- 遵循 Go 官方代码风格
- 使用 golangci-lint 进行代码检查
- 所有导出的函数和类型必须有注释
- 错误处理：不要忽略错误
- 使用 context 传递请求上下文

### 8.2 Git 提交规范
```
feat: 新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式调整
refactor: 重构
test: 测试相关
chore: 构建、配置相关
perf: 性能优化
```

### 8.3 API 设计规范
- RESTful 风格
- 使用 HTTP 状态码
- 统一响应格式
- 版本控制（/api/v1）
- 敏感操作需要二次确认

### 8.4 安全规范
- 所有密码使用 bcrypt 加密
- JWT Token 过期时间：7 天
- HTTPS only（生产环境）
- SQL 注入防护（使用 GORM）
- XSS 防护（前端责任）
- CORS 配置严格
- 限流保护

### 8.5 性能规范
- 数据库查询使用索引
- Redis 缓存热点数据
- 避免 N+1 查询
- 分页查询避免全表扫描
- OSS 使用 CDN 加速

---

## 九、启动命令

### 9.1 开发环境

```bash
# 安装依赖
go mod download

# 启动 API 服务
go run cmd/server/main.go

# 启动 Worker
go run cmd/worker/main.go

# 或使用 Makefile
make dev-server
make dev-worker
```

### 9.2 生产环境

```bash
# 构建
make build

# 运行
./bin/server
./bin/worker
```

### 9.3 Docker

```bash
# 构建镜像
docker build -f Dockerfile.server -t go-analyzer-server .
docker build -f Dockerfile.worker -t go-analyzer-worker .

# 使用 docker-compose
docker-compose up -d

# 查看日志
docker-compose logs -f
```

### 9.4 数据库迁移

```bash
# 执行迁移
make migrate-up

# 回滚
make migrate-down

# 创建新迁移
make migrate-create name=add_new_table
```

---

**文档版本**: v1.0  
**最后更新**: 2025-01-20  
**维护者**: Backend Team

祝开发顺利！ 🚀
