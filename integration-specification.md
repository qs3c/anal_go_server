# 前后端对接规范文档

> **重要性等级：🔴 CRITICAL**
> 
> 本文档定义了前后端必须严格遵守的接口规范和数据格式。违反此规范将导致前后端无法对接！

---

## 📋 目录

- [一、核心原则](#一核心原则)
- [二、数据格式约定](#二数据格式约定)
- [三、API 接口契约](#三api-接口契约)
- [四、WebSocket 消息契约](#四websocket-消息契约)
- [五、文件上传规范](#五文件上传规范)
- [六、错误处理契约](#六错误处理契约)
- [七、对接测试清单](#七对接测试清单)

---

## 一、核心原则

### 1.1 命名规范

**🔴 CRITICAL: 必须严格遵守**

| 位置 | 规范 | 示例 |
|------|------|------|
| 后端 Go 结构体字段 | PascalCase | `UserID`, `CreatedAt` |
| 后端 JSON 字段（序列化后） | snake_case | `user_id`, `created_at` |
| 前端 TypeScript 接口 | camelCase | `userId`, `createdAt` |
| 数据库字段 | snake_case | `user_id`, `created_at` |

**后端序列化配置（Go）：**
```go
type User struct {
    ID        int64     `json:"id"`
    Username  string    `json:"username"`
    Email     string    `json:"email"`
    CreatedAt time.Time `json:"created_at"`
}
```

**前端类型定义（TypeScript）：**
```typescript
interface User {
  id: number
  username: string
  email: string
  created_at: string  // ⚠️ 注意：保持 snake_case 与后端一致
}
```

### 1.2 日期时间格式

**🔴 统一使用 RFC3339 格式（ISO 8601）**

**后端返回：**
```go
// 使用 time.Time，自动序列化为 RFC3339
CreatedAt: time.Now()  // 输出: "2025-01-20T10:30:00Z"
```

**前端解析：**
```typescript
import dayjs from 'dayjs'

const date = dayjs(user.created_at)  // 直接解析
```

### 1.3 布尔值

**🔴 统一使用 JSON 布尔值（true/false）**

```json
{
  "is_public": true,
  "email_verified": false
}
```

❌ **禁止使用：** 0/1, "true"/"false", yes/no

### 1.4 枚举值

**🔴 统一使用字符串枚举，值全小写**

```json
{
  "subscription_level": "free",  // ✅ 正确
  "status": "completed"          // ✅ 正确
}
```

❌ **禁止：** "FREE", "Free", 0, 1

---

## 二、数据格式约定

### 2.1 统一响应格式

**后端返回格式（Go）：**
```go
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data"`
}

// 成功响应
{
    "code": 0,
    "message": "success",
    "data": { ... }
}

// 错误响应
{
    "code": 1001,
    "message": "认证失败",
    "data": null
}
```

**前端类型定义（TypeScript）：**
```typescript
interface ApiResponse<T = any> {
  code: number
  message: string
  data: T
}
```

### 2.2 分页格式

**请求参数：**
```typescript
// 前端发送
{
  page: 1,        // 页码，从 1 开始
  page_size: 20   // 每页数量
}
```

**响应格式：**
```json
{
  "code": 0,
  "data": {
    "total": 100,
    "page": 1,
    "page_size": 20,
    "items": [...]
  }
}
```

### 2.3 时间戳格式

**🔴 CRITICAL: 所有时间字段使用 RFC3339 字符串**

```json
{
  "created_at": "2025-01-20T10:30:00Z",
  "updated_at": "2025-01-20T11:00:00Z",
  "quota_reset_at": "2025-01-21T00:00:00Z"
}
```

❌ **禁止使用 Unix 时间戳（数字）**

### 2.4 空值处理

**规则：**
- 字符串空值：`""`（空字符串）
- 对象空值：`null`
- 数组空值：`[]`（空数组）

```json
{
  "bio": "",           // 未填写的文本字段
  "avatar_url": null,  // 未上传的对象
  "tags": []           // 未选择的数组
}
```

---

## 三、API 接口契约

### 3.1 认证 Token

**请求头格式：**
```
Authorization: Bearer <jwt_token>
```

**前端发送（Axios）：**
```typescript
config.headers.Authorization = `Bearer ${token}`
```

**后端解析（Gin）：**
```go
authHeader := c.GetHeader("Authorization")
tokenString := strings.TrimPrefix(authHeader, "Bearer ")
```

### 3.2 错误码定义

**🔴 前后端必须使用相同的错误码**

| 错误码 | 含义 | 前端处理 |
|--------|------|----------|
| 0 | 成功 | 正常处理 data |
| 1000 | 参数错误 | 显示错误信息 |
| 1001 | 认证失败 | 跳转登录页 |
| 1002 | 权限不足 | 显示提示 |
| 1003 | 资源不存在 | 显示 404 |
| 1004 | 配额不足 | 提示升级 |
| 1005 | 重复操作 | 显示提示 |
| 5000 | 服务器错误 | 显示通用错误 |

**后端定义（Go）：**
```go
const (
    CodeSuccess          = 0
    CodeParamError       = 1000
    CodeAuthFailed       = 1001
    CodePermissionDenied = 1002
    CodeResourceNotFound = 1003
    CodeQuotaExceeded    = 1004
    CodeDuplicateAction  = 1005
    CodeServerError      = 5000
)
```

**前端使用（TypeScript）：**
```typescript
export const ErrorCode = {
  SUCCESS: 0,
  PARAM_ERROR: 1000,
  AUTH_FAILED: 1001,
  PERMISSION_DENIED: 1002,
  RESOURCE_NOT_FOUND: 1003,
  QUOTA_EXCEEDED: 1004,
  DUPLICATE_ACTION: 1005,
  SERVER_ERROR: 5000,
} as const
```

### 3.3 关键接口契约

#### 3.3.1 POST /api/v1/auth/login

**前端请求：**
```typescript
{
  email: "user@example.com",
  password: "Password123"
}
```

**后端响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "token": "eyJhbGci...",
    "user": {
      "id": 1,
      "username": "user1",
      "email": "user@example.com",
      "avatar_url": "",
      "bio": "",
      "subscription_level": "free"
    }
  }
}
```

#### 3.3.2 POST /api/v1/analyses

**前端请求（AI 分析）：**
```typescript
{
  title: "Gin 路由分析",
  creation_type: "ai",
  repo_url: "https://github.com/gin-gonic/gin",
  start_struct: "Engine",
  analysis_depth: 3,
  model_name: "gpt-3.5-turbo"
}
```

**后端响应：**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "analysis_id": 123,
    "job_id": 456
  }
}
```

#### 3.3.3 GET /api/v1/community/analyses

**前端请求：**
```typescript
// Query params
{
  page: 1,
  page_size: 20,
  sort: "latest",  // "latest" | "hot"
  tags: "Web框架,路由"  // 逗号分隔
}
```

**后端响应：**
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
          "avatar_url": "https://..."
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

### 3.4 枚举值对照表

**🔴 前后端必须使用相同的枚举值**

#### 订阅级别 (subscription_level)
```
"free" | "basic" | "pro"
```

#### 分析创建类型 (creation_type)
```
"ai" | "manual"
```

#### 分析状态 (status)
```
"draft" | "pending" | "analyzing" | "completed" | "failed"
```

#### 任务状态 (job status)
```
"queued" | "processing" | "completed" | "failed" | "cancelled"
```

#### 互动类型 (interaction type)
```
"like" | "bookmark"
```

#### 排序方式 (sort)
```
"latest" | "hot"
```

---

## 四、WebSocket 消息契约

### 4.1 连接格式

**WebSocket URL：**
```
ws://api.example.com/api/v1/ws?token=<jwt_token>
```

**前端连接：**
```typescript
const wsURL = `${import.meta.env.VITE_WS_URL}/ws?token=${token}`
const ws = new WebSocket(wsURL)
```

### 4.2 消息格式

**🔴 所有 WebSocket 消息必须是 JSON 格式**

#### 分析进度消息

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

#### 分析完成消息

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

#### 分析失败消息

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

**前端类型定义：**
```typescript
type ProgressMessageType = 
  | 'analysis_progress' 
  | 'analysis_completed' 
  | 'analysis_failed'

interface ProgressMessage {
  type: ProgressMessageType
  data: {
    job_id: number
    analysis_id: number
    status?: string
    current_step?: string
    elapsed_seconds?: number
    diagram_oss_url?: string
    error_message?: string
  }
}
```

---

## 五、文件上传规范

### 5.1 头像上传

**请求格式：**
```
POST /api/v1/user/avatar
Content-Type: multipart/form-data

file: <binary>
```

**前端代码：**
```typescript
const formData = new FormData()
formData.append('file', file)

await api.post('/user/avatar', formData, {
  headers: {
    'Content-Type': 'multipart/form-data',
  },
})
```

**后端处理（Gin）：**
```go
file, err := c.FormFile("file")  // 字段名必须是 "file"
```

**响应格式：**
```json
{
  "code": 0,
  "data": {
    "avatar_url": "https://oss.example.com/avatars/1.jpg"
  }
}
```

---

## 六、错误处理契约

### 6.1 HTTP 状态码

**🔴 后端必须返回以下状态码：**

| 状态码 | 含义 | 前端处理 |
|--------|------|----------|
| 200 | 成功 | 解析 data |
| 400 | 参数错误 | 显示错误信息 |
| 401 | 未认证 | 跳转登录 |
| 403 | 无权限 | 显示提示 |
| 404 | 不存在 | 显示 404 |
| 500 | 服务器错误 | 显示通用错误 |

### 6.2 错误响应格式

**后端返回：**
```json
{
  "code": 1001,
  "message": "Token 已过期，请重新登录",
  "data": null
}
```

**前端拦截器处理：**
```typescript
api.interceptors.response.use(
  (response) => response.data,
  (error) => {
    if (error.response?.status === 401) {
      message.error('登录已过期，请重新登录')
      useAuthStore.getState().logout()
      window.location.href = '/login'
    }
    return Promise.reject(error)
  }
)
```

---

## 七、对接测试清单

### 7.1 认证流程测试

**测试步骤：**

1. **注册流程**
   - [ ] 前端发送注册请求，字段命名正确
   - [ ] 后端返回 user_id
   - [ ] 验证邮件发送成功

2. **登录流程**
   - [ ] 前端发送 email + password
   - [ ] 后端返回 token 和 user 对象
   - [ ] 前端保存 token 到 localStorage
   - [ ] 前端解析 user 对象字段正确

3. **Token 验证**
   - [ ] 前端在请求头添加 `Authorization: Bearer <token>`
   - [ ] 后端正确解析 token
   - [ ] Token 过期时后端返回 401
   - [ ] 前端收到 401 后跳转登录

4. **GitHub OAuth**
   - [ ] 前端重定向到后端 OAuth 地址
   - [ ] 后端回调后重定向到前端，携带 token
   - [ ] 前端解析 token 并登录成功

### 7.2 分析流程测试

1. **创建 AI 分析**
   - [ ] 前端发送正确的字段（creation_type, repo_url, etc.）
   - [ ] 后端返回 analysis_id 和 job_id
   - [ ] 前端收到响应后跳转到进度页

2. **WebSocket 连接**
   - [ ] 前端连接 WebSocket，URL 格式正确
   - [ ] 后端接受连接，解析 token 成功
   - [ ] 心跳机制正常

3. **实时进度推送**
   - [ ] 后端推送进度消息，格式符合契约
   - [ ] 前端解析消息，字段名正确
   - [ ] 前端显示 current_step 和 elapsed_seconds

4. **分析完成**
   - [ ] 后端推送完成消息，包含 diagram_oss_url
   - [ ] 前端收到后跳转到编辑器
   - [ ] 前端从 OSS 加载数据成功

5. **OSS 数据加载**
   - [ ] 前端请求 OSS URL
   - [ ] 前端解压 gzip 数据
   - [ ] 前端解析 JSON 成功
   - [ ] struct_element 渲染成功

### 7.3 社区功能测试

1. **广场列表**
   - [ ] 前端发送分页参数（page, page_size）
   - [ ] 后端返回分页数据（total, items）
   - [ ] 前端解析列表项字段正确
   - [ ] tags 数组解析正确
   - [ ] 时间字段格式正确

2. **点赞功能**
   - [ ] 前端发送点赞请求
   - [ ] 后端返回新的点赞状态和数量
   - [ ] 前端更新 UI

3. **评论功能**
   - [ ] 前端发送评论内容
   - [ ] 后端返回评论对象
   - [ ] 前端插入评论到列表
   - [ ] 回复功能正常（parent_id 传递正确）

### 7.4 文件上传测试

1. **头像上传**
   - [ ] 前端使用 FormData 发送文件
   - [ ] 字段名为 "file"
   - [ ] Content-Type 设置正确
   - [ ] 后端返回 avatar_url
   - [ ] 前端更新用户头像

---

## 八、开发检查清单

### 8.1 后端开发者必查

- [ ] 所有 JSON 字段使用 snake_case
- [ ] 时间字段序列化为 RFC3339 格式
- [ ] 枚举值使用小写字符串
- [ ] 错误码与契约一致
- [ ] WebSocket 消息格式符合契约
- [ ] 文件上传字段名为 "file"
- [ ] CORS 配置正确
- [ ] 返回的 HTTP 状态码正确

### 8.2 前端开发者必查

- [ ] TypeScript 接口字段名与后端一致（snake_case）
- [ ] 时间解析使用 dayjs
- [ ] 枚举值与后端一致
- [ ] 错误码处理完整
- [ ] WebSocket 消息类型定义正确
- [ ] FormData 字段名为 "file"
- [ ] Authorization 头格式正确 (Bearer)
- [ ] 响应拦截器处理 401

---

## 九、快速参考

### 9.1 字段命名速查

```
后端 Go:     UserID, CreatedAt
后端 JSON:   user_id, created_at
前端 TS:     user_id, created_at  (保持一致)
数据库:      user_id, created_at
```

### 9.2 时间格式速查

```
后端发送: "2025-01-20T10:30:00Z"
前端解析: dayjs(dateString)
前端显示: dayjs(dateString).format('YYYY-MM-DD HH:mm:ss')
```

### 9.3 枚举值速查

```go
// 后端
subscription_level: "free" | "basic" | "pro"
creation_type:      "ai" | "manual"
status:             "draft" | "pending" | "analyzing" | "completed" | "failed"
```

```typescript
// 前端（完全一致）
subscription_level: "free" | "basic" | "pro"
creation_type:      "ai" | "manual"
status:             "draft" | "pending" | "analyzing" | "completed" | "failed"
```

---

## 十、故障排查

### 问题：前端收到的字段是 undefined

**原因：** 字段命名不一致

**解决：**
1. 检查后端 JSON tag 是否为 snake_case
2. 检查前端接口字段名是否与后端一致
3. 使用浏览器 Network 查看实际返回的字段名

### 问题：时间显示为 Invalid Date

**原因：** 时间格式不符合 ISO 8601

**解决：**
1. 后端确保使用 time.Time 类型并自动序列化
2. 前端使用 dayjs 解析
3. 检查后端是否返回了 Unix 时间戳（数字）

### 问题：WebSocket 连接失败

**原因：** Token 格式或 URL 格式错误

**解决：**
1. 检查 URL 是否以 ws:// 或 wss:// 开头
2. 检查 token 是否正确传递
3. 检查后端是否正确解析 query 参数中的 token

### 问题：文件上传失败

**原因：** 字段名不一致或 Content-Type 错误

**解决：**
1. 确认前端 FormData 字段名为 "file"
2. 确认 Content-Type 设置为 multipart/form-data
3. 确认后端使用 c.FormFile("file") 获取

---

**文档版本**: v1.0  
**最后更新**: 2025-01-20  
**维护者**: 架构组

⚠️ **重要提醒**：本文档是前后端对接的唯一依据，任何不符合此规范的实现都将导致对接失败！
