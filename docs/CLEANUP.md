# 本地存储清理机制

## 概述

系统提供了自动清理本地存储的工具，可以定期清理过期的上传文件和已迁移到OSS的diagram文件。

## 清理策略

### 1. 上传文件清理
- **路径**: `/tmp/uploads/{upload_id}/`
- **默认保留时间**: 24小时
- **说明**: 用户上传的源代码压缩包，分析完成后即可删除

### 2. Diagram文件清理
- **路径**: `/tmp/uploads/diagrams/{analysis_id}.json`
- **默认保留时间**: 7天
- **清理条件**:
  - 文件已迁移到OSS（`diagram_oss_url` 以 `https://` 开头）
  - 文件修改时间超过保留期限
- **说明**: 作为OSS的临时备份，确认迁移成功后可以删除

## 使用方法

### 手动运行（推荐先用dry-run测试）

```bash
# 1. 测试清理（不实际删除）
docker exec anal_worker /app/cleanup \
  -dry-run=true \
  -upload-expire=24 \
  -diagram-expire=7

# 2. 实际执行清理
docker exec anal_worker /app/cleanup \
  -dry-run=false \
  -upload-expire=24 \
  -diagram-expire=7
```

### 使用清理脚本

```bash
# 给脚本添加执行权限
chmod +x scripts/cleanup.sh

# 测试运行（dry-run）
./scripts/cleanup.sh

# 实际清理
./scripts/cleanup.sh --execute

# 自定义参数
./scripts/cleanup.sh --execute --upload-expire 12 --diagram-expire 3
```

## 参数说明

| 参数 | 说明 | 默认值 |
|-----|------|--------|
| `-dry-run` | 测试模式，不实际删除 | `true` |
| `-upload-expire` | 上传文件保留时间（小时） | `24` |
| `-diagram-expire` | diagram文件保留时间（天） | `7` |
| `-clean-uploads` | 是否清理上传文件 | `true` |
| `-clean-diagrams` | 是否清理diagram文件 | `true` |

## 设置定时任务

### 方案1：使用宿主机 cron

编辑 crontab：
```bash
crontab -e
```

添加定时任务（每天凌晨2点执行）：
```cron
0 2 * * * /path/to/anal_go_server/scripts/cleanup.sh --execute >> /var/log/anal_cleanup.log 2>&1
```

### 方案2：使用Docker内部cron

1. 创建 crontab 文件：
```bash
# /etc/cron.d/anal-cleanup
0 2 * * * root /app/cleanup -dry-run=false -upload-expire=24 -diagram-expire=7 >> /var/log/cleanup.log 2>&1
```

2. 修改 Dockerfile.worker 添加cron：
```dockerfile
RUN apk --no-cache add ca-certificates tzdata git dcron

# 添加 crontab
COPY scripts/crontab /etc/cron.d/anal-cleanup
RUN chmod 0644 /etc/cron.d/anal-cleanup && \
    crontab /etc/cron.d/anal-cleanup
```

### 方案3：使用外部调度工具

如果你有外部调度系统（如Kubernetes CronJob、Jenkins等），可以配置定期调用：
```bash
kubectl create cronjob anal-cleanup \
  --image=alpine \
  --schedule="0 2 * * *" \
  -- /bin/sh -c "docker exec anal_worker /app/cleanup -dry-run=false"
```

## 清理报告示例

```
🧹 Starting cleanup task...
Mode: dry-run=false

📦 Cleaning expired upload files (older than 24 hours)...
  - abc123... (0.94 MB, 41h old)
  - def456... (1.20 MB, 38h old)
Found 2 expired upload directories (total: 2.14 MB)

📊 Cleaning diagrams migrated to OSS...
Found 5 analyses migrated to OSS
  - 23.json (7.70 KB, migrated to OSS, 8 days old)
  - 24.json (3.50 KB, migrated to OSS, 8 days old)
Found 2 diagram files to clean (total: 11.20 KB)

📈 Scanning current disk usage...

============================================================
📊 Cleanup Summary
============================================================
Total files: 3150
Total size: 12.54 MB
Deleted files: 4
Freed space: 2.15 MB

✅ Cleanup completed!
============================================================
```

## 监控建议

1. **定期检查日志**：确保清理任务正常执行
2. **监控磁盘使用**：设置磁盘使用率告警（如超过80%）
3. **备份验证**：清理前确认重要数据已上传到OSS

## 安全措施

1. **默认dry-run模式**：防止误删除
2. **保留时间限制**：diagram文件默认保留7天，确保OSS迁移稳定
3. **数据库验证**：只清理已确认迁移到OSS的文件
4. **详细日志**：记录所有删除操作

## 恢复机制

如果误删除了重要文件：

1. **OSS存储的文件**：直接从OSS恢复（永久存储）
2. **本地文件**：如果有Docker volume备份，可以从备份恢复
3. **源代码文件**：可以要求用户重新上传

## 常见问题

**Q: 为什么不直接删除本地文件？**
A: 保留7天作为备份，确保OSS迁移成功且稳定后再删除。

**Q: 清理会影响正在进行的分析吗？**
A: 不会。清理只针对过期文件，正在进行的分析使用的文件不会被删除。

**Q: 可以手动删除单个文件吗？**
A: 可以，但不推荐。建议使用清理工具，它会验证数据安全性。

## 最佳实践

1. **首次使用先dry-run**：熟悉清理流程
2. **逐步缩短保留时间**：从7天开始，稳定后可以调整为3天
3. **定期运行**：建议每天凌晨执行
4. **监控OSS成本**：OSS存储虽便宜但也有成本，定期清理不需要的分析记录
